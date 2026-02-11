package export

import (
	"strings"
	"testing"

	"github.com/sadopc/gottp/internal/protocol"
)

func TestAsCurl_GET(t *testing.T) {
	req := &protocol.Request{
		Method:  "GET",
		URL:     "https://api.example.com/users",
		Headers: map[string]string{"Accept": "application/json"},
	}

	result := AsCurl(req)
	if !strings.HasPrefix(result, "curl") {
		t.Error("should start with 'curl'")
	}
	if strings.Contains(result, "-X") {
		t.Error("GET should not have -X flag")
	}
	if !strings.Contains(result, "Accept: application/json") {
		t.Error("should contain Accept header")
	}
	if !strings.Contains(result, "https://api.example.com/users") {
		t.Error("should contain URL")
	}
}

func TestAsCurl_POST(t *testing.T) {
	req := &protocol.Request{
		Method:  "POST",
		URL:     "https://api.example.com/users",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"name":"test"}`),
	}

	result := AsCurl(req)
	if !strings.Contains(result, "-X POST") {
		t.Error("should have -X POST")
	}
	if !strings.Contains(result, `-d '{"name":"test"}'`) {
		t.Errorf("should contain body data, got: %s", result)
	}
}

func TestAsCurl_WithAuth(t *testing.T) {
	req := &protocol.Request{
		Method:  "GET",
		URL:     "https://api.example.com/me",
		Headers: map[string]string{},
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "mytoken123",
		},
	}

	result := AsCurl(req)
	if !strings.Contains(result, "Authorization: Bearer mytoken123") {
		t.Error("should contain Bearer auth header")
	}
}

func TestAsCurl_WithParams(t *testing.T) {
	req := &protocol.Request{
		Method:  "GET",
		URL:     "https://api.example.com/search",
		Headers: map[string]string{},
		Params:  map[string]string{"q": "test", "limit": "10"},
	}

	result := AsCurl(req)
	if !strings.Contains(result, "q=test") {
		t.Error("should contain query param q")
	}
	if !strings.Contains(result, "limit=10") {
		t.Error("should contain query param limit")
	}
}
