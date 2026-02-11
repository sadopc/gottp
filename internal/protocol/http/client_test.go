package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serdar/gottp/internal/protocol"
)

func TestClient_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("page") != "1" {
			t.Error("expected page=1 query param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := New()
	resp, err := client.Execute(context.Background(), &protocol.Request{
		Method:  "GET",
		URL:     server.URL + "/test",
		Params:  map[string]string{"page": "1"},
		Headers: map[string]string{"Accept": "application/json"},
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", resp.ContentType)
	}
	if resp.Duration == 0 {
		t.Error("duration should be > 0")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("body unmarshal failed: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", body["status"])
	}
}

func TestClient_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var data map[string]string
		json.Unmarshal(body, &data)
		if data["name"] != "test" {
			t.Errorf("expected name=test, got %s", data["name"])
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Execute(context.Background(), &protocol.Request{
		Method:  "POST",
		URL:     server.URL + "/users",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"name":"test"}`),
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestClient_BearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("expected Bearer mytoken, got %q", auth)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New()
	_, err := client.Execute(context.Background(), &protocol.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "mytoken",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestClient_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "admin" || pass != "secret" {
			t.Errorf("expected admin:secret, got %s:%s", user, pass)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New()
	_, err := client.Execute(context.Background(), &protocol.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Auth: &protocol.AuthConfig{
			Type:     "basic",
			Username: "admin",
			Password: "secret",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestClient_Validate(t *testing.T) {
	client := New()

	if err := client.Validate(&protocol.Request{URL: "", Method: "GET"}); err == nil {
		t.Error("should fail with empty URL")
	}
	if err := client.Validate(&protocol.Request{URL: "http://example.com", Method: ""}); err == nil {
		t.Error("should fail with empty method")
	}
	if err := client.Validate(&protocol.Request{URL: "http://example.com", Method: "GET"}); err != nil {
		t.Errorf("should pass: %v", err)
	}
}
