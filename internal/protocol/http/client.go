package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/serdar/gottp/internal/auth/awsv4"
	"github.com/serdar/gottp/internal/protocol"
)

// Client implements the HTTP protocol.
type Client struct {
	httpClient *http.Client
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

	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: c.httpClient.CheckRedirect,
		Transport:     c.httpClient.Transport,
	}

	// Execute
	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
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
	}, nil
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
			awsv4.Sign(req, body, cfg, time.Now())
		}
	}
}
