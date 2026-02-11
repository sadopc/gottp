package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/serdar/gottp/internal/protocol"
)

func TestParseNoProxyAndShouldBypassProxy(t *testing.T) {
	hosts := parseNoProxy("example.com, .internal, LOCALHOST ,")
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d (%v)", len(hosts), hosts)
	}

	if !shouldBypassProxy("example.com", hosts) {
		t.Fatal("expected exact host to bypass proxy")
	}
	if !shouldBypassProxy("api.internal", hosts) {
		t.Fatal("expected suffix wildcard host to bypass proxy")
	}
	if !shouldBypassProxy("localhost", hosts) {
		t.Fatal("expected case-insensitive host to bypass proxy")
	}
	if shouldBypassProxy("golang.org", hosts) {
		t.Fatal("did not expect unrelated host to bypass proxy")
	}
}

func TestBuildTransport_HTTPProxyWithNoProxy(t *testing.T) {
	c := New()
	c.SetProxy("http://proxy.example.com:8080", "example.com,.internal")

	rt, err := c.buildTransport("")
	if err != nil {
		t.Fatalf("buildTransport failed: %v", err)
	}

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", rt)
	}
	if tr.Proxy == nil {
		t.Fatal("expected transport proxy function to be set")
	}

	reqBypassExact, _ := http.NewRequest("GET", "http://example.com/v1", nil)
	proxyURL, err := tr.Proxy(reqBypassExact)
	if err != nil {
		t.Fatalf("proxy func returned error for exact bypass: %v", err)
	}
	if proxyURL != nil {
		t.Fatalf("expected nil proxy for exact bypass, got %v", proxyURL)
	}

	reqBypassSuffix, _ := http.NewRequest("GET", "http://api.internal/v1", nil)
	proxyURL, err = tr.Proxy(reqBypassSuffix)
	if err != nil {
		t.Fatalf("proxy func returned error for suffix bypass: %v", err)
	}
	if proxyURL != nil {
		t.Fatalf("expected nil proxy for suffix bypass, got %v", proxyURL)
	}

	reqProxy, _ := http.NewRequest("GET", "http://golang.org", nil)
	proxyURL, err = tr.Proxy(reqProxy)
	if err != nil {
		t.Fatalf("proxy func returned error for proxied host: %v", err)
	}
	if proxyURL == nil || proxyURL.Host != "proxy.example.com:8080" {
		t.Fatalf("expected proxy host proxy.example.com:8080, got %v", proxyURL)
	}
}

func TestBuildTransport_PerRequestOverride(t *testing.T) {
	c := New()
	// Global proxy would be invalid if used.
	c.SetProxy("://bad-url", "")

	rt, err := c.buildTransport("http://override.proxy:9090")
	if err != nil {
		t.Fatalf("buildTransport should use per-request override and succeed: %v", err)
	}

	tr := rt.(*http.Transport)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	p, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if p == nil || p.Host != "override.proxy:9090" {
		t.Fatalf("expected override proxy host override.proxy:9090, got %v", p)
	}
}

func TestBuildTransportErrors(t *testing.T) {
	c := New()

	c.SetProxy("://bad-url", "")
	if _, err := c.buildTransport(""); err == nil {
		t.Fatal("expected parsing proxy URL error")
	}

	c.SetProxy("ftp://proxy.example.com", "")
	if _, err := c.buildTransport(""); err == nil {
		t.Fatal("expected unsupported proxy scheme error")
	}
}

func TestBuildTransportAppliesTLSConfig(t *testing.T) {
	c := New()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	c.SetTLSConfig(tlsCfg)

	rt, err := c.buildTransport("")
	if err != nil {
		t.Fatalf("buildTransport failed: %v", err)
	}

	tr := rt.(*http.Transport)
	if tr.TLSClientConfig != tlsCfg {
		t.Fatal("expected TLS config pointer to be applied to transport")
	}
}

func TestSetProxyClearsConfigWhenEmpty(t *testing.T) {
	c := New()
	c.SetProxy("http://proxy.example.com:8080", "")
	if c.proxyConf == nil {
		t.Fatal("expected proxy configuration to be set")
	}

	c.SetProxy("", "")
	if c.proxyConf != nil {
		t.Fatal("expected proxy configuration to be cleared")
	}
}

func TestApplyAuth_APIKeyOAuth2AWSAndNone(t *testing.T) {
	// apikey in query
	reqQuery := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/users"}, Header: make(http.Header)}
	applyAuth(reqQuery, &protocol.AuthConfig{Type: "apikey", APIIn: "query", APIKey: "token", APIValue: "abc"}, nil)
	if got := reqQuery.URL.Query().Get("token"); got != "abc" {
		t.Fatalf("expected query token=abc, got %q", got)
	}

	// apikey in header
	reqHeader := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/users"}, Header: make(http.Header)}
	applyAuth(reqHeader, &protocol.AuthConfig{Type: "apikey", APIIn: "header", APIKey: "X-API-Key", APIValue: "abc"}, nil)
	if got := reqHeader.Header.Get("X-API-Key"); got != "abc" {
		t.Fatalf("expected X-API-Key header to be set, got %q", got)
	}

	// oauth2 bearer
	reqOAuth := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/users"}, Header: make(http.Header)}
	applyAuth(reqOAuth, &protocol.AuthConfig{
		Type: "oauth2",
		OAuth2: &protocol.OAuth2AuthConfig{
			AccessToken: "oauth-token",
		},
	}, nil)
	if got := reqOAuth.Header.Get("Authorization"); got != "Bearer oauth-token" {
		t.Fatalf("expected oauth Authorization header, got %q", got)
	}

	// aws v4
	reqAWS := &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "service.amazonaws.com", Path: "/"}, Header: make(http.Header)}
	applyAuth(reqAWS, &protocol.AuthConfig{
		Type: "awsv4",
		AWSAuth: &protocol.AWSAuthConfig{
			AccessKeyID:     "AKIDEXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
			Region:          "us-east-1",
			Service:         "execute-api",
		},
	}, []byte("{}"))
	if got := reqAWS.Header.Get("Authorization"); !strings.Contains(got, "AWS4-HMAC-SHA256") {
		t.Fatalf("expected AWS SigV4 Authorization header, got %q", got)
	}
	if reqAWS.Header.Get("X-Amz-Date") == "" {
		t.Fatal("expected X-Amz-Date header to be set by SigV4")
	}

	// none/nil should not panic or set auth
	reqNone := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/users"}, Header: make(http.Header)}
	applyAuth(reqNone, nil, nil)
	if got := reqNone.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected no Authorization header for nil auth, got %q", got)
	}
}

func TestExecute_DigestRetry(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		if strings.HasPrefix(r.Header.Get("Authorization"), "Digest ") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}

		w.Header().Set("WWW-Authenticate", `Digest realm="test", nonce="abc123", qop="auth", algorithm=MD5`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := New()
	resp, err := c.Execute(context.Background(), &protocol.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Auth: &protocol.AuthConfig{
			Type:           "digest",
			DigestUsername: "user",
			DigestPassword: "pass",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 after digest retry, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Fatalf("expected at least two calls (challenge + retry), got %d", callCount)
	}
}
