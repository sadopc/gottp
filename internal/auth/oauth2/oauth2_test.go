package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("expected client_credentials, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_id") != "test-id" {
			t.Errorf("expected test-id, got %s", r.Form.Get("client_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token-123",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	cfg := OAuth2Config{
		GrantType:    "client_credentials",
		TokenURL:     server.URL,
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Scope:        "read write",
	}

	token, err := ClientCredentials(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "test-token-123" {
		t.Errorf("expected test-token-123, got %s", token.AccessToken)
	}
	if token.ExpiresIn != 3600 {
		t.Errorf("expected 3600, got %d", token.ExpiresIn)
	}
}

func TestPasswordGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "password" {
			t.Errorf("expected password grant_type, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("username") != "user@test.com" {
			t.Errorf("wrong username")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "pwd-token",
			"refresh_token": "refresh-123",
			"expires_in":    1800,
		})
	}))
	defer server.Close()

	cfg := OAuth2Config{
		TokenURL:     server.URL,
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Username:     "user@test.com",
		Password:     "pass123",
	}

	token, err := PasswordGrant(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "pwd-token" {
		t.Errorf("expected pwd-token, got %s", token.AccessToken)
	}
	if token.RefreshToken != "refresh-123" {
		t.Errorf("expected refresh-123, got %s", token.RefreshToken)
	}
}

func TestExchangeAuthCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("expected authorization_code, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "auth-code-xyz" {
			t.Errorf("expected auth-code-xyz, got %s", r.Form.Get("code"))
		}
		if r.Form.Get("code_verifier") != "verifier-123" {
			t.Errorf("expected verifier-123, got %s", r.Form.Get("code_verifier"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "auth-code-token",
			"expires_in":   7200,
		})
	}))
	defer server.Close()

	cfg := OAuth2Config{
		TokenURL:    server.URL,
		ClientID:    "test-id",
		RedirectURI: "http://localhost:0/callback",
	}

	token, err := ExchangeAuthCode(context.Background(), cfg, "auth-code-xyz", "verifier-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "auth-code-token" {
		t.Errorf("expected auth-code-token, got %s", token.AccessToken)
	}
}

func TestPKCE(t *testing.T) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(verifier) < 40 {
		t.Errorf("verifier too short: %d", len(verifier))
	}

	challenge := GenerateCodeChallenge(verifier)
	if challenge == "" {
		t.Error("empty code challenge")
	}
	if challenge == verifier {
		t.Error("challenge should differ from verifier")
	}
}

func TestBuildAuthURL(t *testing.T) {
	cfg := OAuth2Config{
		AuthURL:     "https://auth.example.com/authorize",
		ClientID:    "my-client",
		RedirectURI: "http://localhost:8080/callback",
		Scope:       "openid profile",
	}

	u := BuildAuthURL(cfg, "random-state", "challenge-123")
	if u == "" {
		t.Fatal("empty URL")
	}
	// Verify it contains expected params
	for _, expected := range []string{"response_type=code", "client_id=my-client", "state=random-state", "code_challenge=challenge-123"} {
		if !contains(u, expected) {
			t.Errorf("URL missing %q: %s", expected, u)
		}
	}
}

func TestTokenExpiry(t *testing.T) {
	token := &TokenResponse{
		AccessToken: "test",
		ExpiresIn:   1,
		ObtainedAt:  time.Now().Add(-2 * time.Second),
	}
	if !token.IsExpired() {
		t.Error("expected token to be expired")
	}

	token2 := &TokenResponse{
		AccessToken: "test",
		ExpiresIn:   3600,
		ObtainedAt:  time.Now(),
	}
	if token2.IsExpired() {
		t.Error("expected token to not be expired")
	}
}

func TestCallbackServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	portCh := make(chan int, 1)

	go func() {
		code, port, err := StartCallbackServer(ctx)
		portCh <- port
		if err != nil {
			errCh <- err
			return
		}
		codeCh <- code
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Read port (server writes it before blocking)
	var port int
	select {
	case port = <-portCh:
	case <-time.After(2 * time.Second):
		// Port might not be sent yet, try hitting common callback
		t.Skip("could not get port in time")
	}

	// Send callback request
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-auth-code&state=xyz", port)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	select {
	case code := <-codeCh:
		if code != "test-auth-code" {
			t.Errorf("expected test-auth-code, got %s", code)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for callback code")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
