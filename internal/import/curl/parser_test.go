package curl

import "testing"

func TestParseCurl_SimpleGET(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "GET" {
		t.Errorf("expected GET, got %s", req.Method)
	}
	if req.URL != "https://api.example.com/users" {
		t.Errorf("expected URL, got %s", req.URL)
	}
}

func TestParseCurl_POST_WithBody(t *testing.T) {
	req, err := ParseCurl(`curl -X POST -H 'Content-Type: application/json' -d '{"name":"test"}' https://api.example.com/users`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("expected POST, got %s", req.Method)
	}
	if string(req.Body) != `{"name":"test"}` {
		t.Errorf("unexpected body: %s", string(req.Body))
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %v", req.Headers)
	}
}

func TestParseCurl_BasicAuth(t *testing.T) {
	req, err := ParseCurl(`curl -u admin:secret https://api.example.com/private`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Auth == nil {
		t.Fatal("expected auth config")
	}
	if req.Auth.Type != "basic" {
		t.Errorf("expected basic auth, got %s", req.Auth.Type)
	}
	if req.Auth.Username != "admin" || req.Auth.Password != "secret" {
		t.Errorf("unexpected credentials: %s:%s", req.Auth.Username, req.Auth.Password)
	}
}

func TestParseCurl_MultipleHeaders(t *testing.T) {
	req, err := ParseCurl(`curl -H "Accept: application/json" -H "Authorization: Bearer token123" https://api.example.com`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Headers["Accept"] != "application/json" {
		t.Errorf("missing Accept header")
	}
	if req.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("missing Authorization header")
	}
}

func TestParseCurl_ImplicitPOST(t *testing.T) {
	req, err := ParseCurl(`curl -d 'data=value' https://api.example.com`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("expected implicit POST, got %s", req.Method)
	}
}

func TestParseCurl_LineContinuation(t *testing.T) {
	input := "curl \\\n  -X PUT \\\n  -H 'Content-Type: text/plain' \\\n  -d 'hello' \\\n  https://example.com"
	req, err := ParseCurl(input)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "PUT" {
		t.Errorf("expected PUT, got %s", req.Method)
	}
	if req.URL != "https://example.com" {
		t.Errorf("unexpected URL: %s", req.URL)
	}
}

func TestParseCurl_Empty(t *testing.T) {
	_, err := ParseCurl("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseCurl_NoURL(t *testing.T) {
	_, err := ParseCurl("curl -H 'Accept: */*'")
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestTokenize(t *testing.T) {
	tokens := tokenize(`curl -H 'Content-Type: application/json' -d '{"key":"val"}' "https://example.com"`)
	expected := []string{
		"curl",
		"-H",
		"Content-Type: application/json",
		"-d",
		`{"key":"val"}`,
		"https://example.com",
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i := range expected {
		if tokens[i] != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tokens[i], expected[i])
		}
	}
}
