package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"github.com/serdar/gottp/internal/auth/awsv4"
	"github.com/serdar/gottp/internal/auth/digest"
	"github.com/serdar/gottp/internal/core/cookies"
	"github.com/serdar/gottp/internal/protocol"
	"golang.org/x/net/proxy"
)

// ProxyConfig holds proxy settings.
type ProxyConfig struct {
	URL     string // http://, https://, or socks5:// proxy URL
	NoProxy string // comma-separated list of hosts to bypass proxy
}

// Client implements the HTTP protocol.
type Client struct {
	httpClient *http.Client
	proxyConf  *ProxyConfig
	cookieJar  *cookies.Jar
	tlsConfig  *tls.Config
}

// New creates a new HTTP client.
func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// SetTimeout sets the default client timeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

// SetProxy configures proxy settings for the client.
func (c *Client) SetProxy(proxyURL, noProxy string) {
	if proxyURL == "" {
		c.proxyConf = nil
		return
	}
	c.proxyConf = &ProxyConfig{URL: proxyURL, NoProxy: noProxy}
}

// SetCookieJar sets the cookie jar for automatic cookie handling.
func (c *Client) SetCookieJar(jar *cookies.Jar) {
	c.cookieJar = jar
}

// SetTLSConfig sets the TLS configuration for mTLS and certificate management.
func (c *Client) SetTLSConfig(cfg *tls.Config) {
	c.tlsConfig = cfg
}

func (c *Client) Name() string { return "http" }

func (c *Client) Validate(req *protocol.Request) error {
	if req.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if req.Method == "" {
		return fmt.Errorf("method is required")
	}
	_, err := url.Parse(req.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	return nil
}

func (c *Client) Execute(ctx context.Context, req *protocol.Request) (*protocol.Response, error) {
	if err := c.Validate(req); err != nil {
		return nil, err
	}

	// Build URL with query params
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if len(req.Params) > 0 {
		q := u.Query()
		for k, v := range req.Params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Build body
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Apply auth
	applyAuth(httpReq, req.Auth, req.Body)

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Build transport with proxy and TLS settings
	transport, err := c.buildTransport(req.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("configuring transport: %w", err)
	}

	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: c.httpClient.CheckRedirect,
		Transport:     transport,
	}

	// Set cookie jar if configured
	if c.cookieJar != nil {
		client.Jar = c.cookieJar.GetJar()
	}

	// Set up httptrace for detailed timing
	var dnsStart, connStart, tlsStart, gotConn, gotFirstByte time.Time
	var dnsDuration, connDuration, tlsDuration time.Duration

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			dnsDuration = time.Since(dnsStart)
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			connDuration = time.Since(connStart)
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tlsDuration = time.Since(tlsStart)
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			gotConn = time.Now()
		},
		GotFirstResponseByte: func() {
			gotFirstByte = time.Now()
		},
	}

	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))

	// Execute
	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read body
	transferStart := time.Now()
	respBody, err := io.ReadAll(resp.Body)
	transferDuration := time.Since(transferStart)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Handle digest auth: if response is 401 with WWW-Authenticate: Digest and auth type is "digest", retry
	if resp.StatusCode == http.StatusUnauthorized && req.Auth != nil && req.Auth.Type == "digest" {
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if strings.HasPrefix(wwwAuth, "Digest ") || strings.HasPrefix(wwwAuth, "digest ") {
			ch, parseErr := digest.ParseChallenge(wwwAuth)
			if parseErr == nil {
				// Compute the request URI (path + query)
				digestURI := u.RequestURI()

				// Build the Authorization header
				authHeader := digest.Authorize(
					req.Auth.DigestUsername,
					req.Auth.DigestPassword,
					req.Method,
					digestURI,
					ch,
				)

				// Rebuild the request for retry
				var retryBody io.Reader
				if len(req.Body) > 0 {
					retryBody = bytes.NewReader(req.Body)
				}
				retryReq, retryErr := http.NewRequestWithContext(ctx, req.Method, u.String(), retryBody)
				if retryErr == nil {
					// Copy original headers
					for k, v := range req.Headers {
						retryReq.Header.Set(k, v)
					}
					retryReq.Header.Set("Authorization", authHeader)

					// Reset timing for the retry request
					dnsStart, connStart, tlsStart, gotConn, gotFirstByte = time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}
					dnsDuration, connDuration, tlsDuration = 0, 0, 0

					retryReq = retryReq.WithContext(httptrace.WithClientTrace(retryReq.Context(), trace))

					retryStart := time.Now()
					retryResp, retryDoErr := client.Do(retryReq)
					retryDuration := time.Since(retryStart)
					if retryDoErr == nil {
						resp.Body.Close()
						resp = retryResp
						duration = retryDuration
						start = retryStart

						transferStart = time.Now()
						respBody, err = io.ReadAll(resp.Body)
						transferDuration = time.Since(transferStart)
						if err != nil {
							resp.Body.Close()
							return nil, fmt.Errorf("reading digest retry response: %w", err)
						}
					}
				}
			}
		}
	}

	// Build timing detail
	var ttfb time.Duration
	if !gotConn.IsZero() && !gotFirstByte.IsZero() {
		ttfb = gotFirstByte.Sub(gotConn)
	}

	timing := &protocol.TimingDetail{
		DNSLookup:    dnsDuration,
		TCPConnect:   connDuration,
		TLSHandshake: tlsDuration,
		TTFB:         ttfb,
		Transfer:     transferDuration,
		Total:        duration,
	}

	return &protocol.Response{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Header,
		Body:        respBody,
		ContentType: resp.Header.Get("Content-Type"),
		Duration:    duration,
		Size:        int64(len(respBody)),
		Proto:       resp.Proto,
		TLS:         resp.TLS != nil,
		Timing:      timing,
	}, nil
}

// buildTransport creates an http.Transport configured with proxy and TLS settings.
// perRequestProxy overrides the client-level proxy config if non-empty.
func (c *Client) buildTransport(perRequestProxy string) (http.RoundTripper, error) {
	transport := &http.Transport{
		// Sensible defaults
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Apply TLS config
	if c.tlsConfig != nil {
		transport.TLSClientConfig = c.tlsConfig
	}

	// Determine effective proxy URL (per-request overrides global)
	proxyURL := perRequestProxy
	noProxy := ""
	if proxyURL == "" && c.proxyConf != nil {
		proxyURL = c.proxyConf.URL
		noProxy = c.proxyConf.NoProxy
	}

	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy URL: %w", err)
		}

		switch parsed.Scheme {
		case "socks5", "socks5h":
			// SOCKS5 proxy via x/net/proxy
			var auth *proxy.Auth
			if parsed.User != nil {
				password, _ := parsed.User.Password()
				auth = &proxy.Auth{
					User:     parsed.User.Username(),
					Password: password,
				}
			}
			dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("creating SOCKS5 dialer: %w", err)
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		case "http", "https":
			// HTTP/HTTPS proxy
			if noProxy != "" {
				noProxyHosts := parseNoProxy(noProxy)
				transport.Proxy = func(r *http.Request) (*url.URL, error) {
					if shouldBypassProxy(r.URL.Hostname(), noProxyHosts) {
						return nil, nil
					}
					return parsed, nil
				}
			} else {
				transport.Proxy = http.ProxyURL(parsed)
			}
		default:
			return nil, fmt.Errorf("unsupported proxy scheme: %s", parsed.Scheme)
		}
	}

	return transport, nil
}

// parseNoProxy splits a comma-separated no-proxy string into trimmed host entries.
func parseNoProxy(noProxy string) []string {
	parts := strings.Split(noProxy, ",")
	hosts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			hosts = append(hosts, strings.ToLower(p))
		}
	}
	return hosts
}

// shouldBypassProxy checks whether a host should bypass the proxy.
func shouldBypassProxy(host string, noProxyHosts []string) bool {
	host = strings.ToLower(host)
	for _, h := range noProxyHosts {
		if h == host {
			return true
		}
		// Support wildcard suffix matching (e.g., .example.com)
		if strings.HasPrefix(h, ".") && strings.HasSuffix(host, h) {
			return true
		}
	}
	return false
}

func applyAuth(req *http.Request, auth *protocol.AuthConfig, body []byte) {
	if auth == nil || auth.Type == "none" {
		return
	}
	switch auth.Type {
	case "basic":
		encoded := base64.StdEncoding.EncodeToString(
			[]byte(auth.Username + ":" + auth.Password),
		)
		req.Header.Set("Authorization", "Basic "+encoded)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "apikey":
		if auth.APIIn == "query" {
			q := req.URL.Query()
			q.Set(auth.APIKey, auth.APIValue)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(auth.APIKey, auth.APIValue)
		}
	case "oauth2":
		if auth.OAuth2 != nil && auth.OAuth2.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+auth.OAuth2.AccessToken)
		}
	case "awsv4":
		if auth.AWSAuth != nil {
			cfg := awsv4.AWSConfig{
				AccessKeyID:    auth.AWSAuth.AccessKeyID,
				SecretAccessKey: auth.AWSAuth.SecretAccessKey,
				SessionToken:   auth.AWSAuth.SessionToken,
				Region:         auth.AWSAuth.Region,
				Service:        auth.AWSAuth.Service,
			}
			_ = awsv4.Sign(req, body, cfg, time.Now())
		}
	}
}
