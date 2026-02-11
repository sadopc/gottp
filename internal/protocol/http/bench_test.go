package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/serdar/gottp/internal/protocol"
)

// newTestServer creates a simple echo server that reads the request body and
// responds with a JSON payload of the specified size.
func newTestServer(respSize int) *httptest.Server {
	body := strings.Repeat("x", respSize)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))
}

func BenchmarkClientExecute(b *testing.B) {
	b.Run("GET/NoBody", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method:  "GET",
			URL:     server.URL + "/test",
			Headers: map[string]string{"Accept": "application/json"},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GET/WithParams", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method: "GET",
			URL:    server.URL + "/search",
			Params: map[string]string{
				"q":     "benchmark",
				"page":  "1",
				"limit": "50",
				"sort":  "date",
			},
			Headers: map[string]string{
				"Accept":       "application/json",
				"X-Request-ID": "bench-123",
			},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("POST/SmallBody", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		payload, _ := json.Marshal(map[string]string{"name": "test"})
		req := &protocol.Request{
			Method:  "POST",
			URL:     server.URL + "/users",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    payload,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("POST/MediumBody_1KB", func(b *testing.B) {
		server := newTestServer(256)
		defer server.Close()
		client := New()
		body := []byte(strings.Repeat(`{"key":"value"},`, 64)) // ~1KB
		req := &protocol.Request{
			Method:  "POST",
			URL:     server.URL + "/data",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    body,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("POST/LargeBody_100KB", func(b *testing.B) {
		server := newTestServer(1024)
		defer server.Close()
		client := New()
		body := []byte(strings.Repeat("a", 100*1024)) // 100KB
		req := &protocol.Request{
			Method:  "POST",
			URL:     server.URL + "/upload",
			Headers: map[string]string{"Content-Type": "application/octet-stream"},
			Body:    body,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("POST/LargeBody_1MB", func(b *testing.B) {
		server := newTestServer(1024)
		defer server.Close()
		client := New()
		body := []byte(strings.Repeat("a", 1024*1024)) // 1MB
		req := &protocol.Request{
			Method:  "POST",
			URL:     server.URL + "/upload",
			Headers: map[string]string{"Content-Type": "application/octet-stream"},
			Body:    body,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkClientExecuteResponseSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"64B", 64},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		b.Run(fmt.Sprintf("ResponseSize/%s", s.name), func(b *testing.B) {
			server := newTestServer(s.size)
			defer server.Close()
			client := New()
			req := &protocol.Request{
				Method:  "GET",
				URL:     server.URL,
				Headers: map[string]string{},
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := client.Execute(context.Background(), req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkClientExecuteWithAuth(b *testing.B) {
	b.Run("BasicAuth", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method:  "GET",
			URL:     server.URL,
			Headers: map[string]string{},
			Auth: &protocol.AuthConfig{
				Type:     "basic",
				Username: "admin",
				Password: "secret-password-123",
			},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("BearerAuth", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method:  "GET",
			URL:     server.URL,
			Headers: map[string]string{},
			Auth: &protocol.AuthConfig{
				Type:  "bearer",
				Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("APIKeyHeader", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method:  "GET",
			URL:     server.URL,
			Headers: map[string]string{},
			Auth: &protocol.AuthConfig{
				Type:     "apikey",
				APIKey:   "X-API-Key",
				APIValue: "sk-1234567890abcdef",
				APIIn:    "header",
			},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("APIKeyQuery", func(b *testing.B) {
		server := newTestServer(64)
		defer server.Close()
		client := New()
		req := &protocol.Request{
			Method:  "GET",
			URL:     server.URL,
			Headers: map[string]string{},
			Auth: &protocol.AuthConfig{
				Type:     "apikey",
				APIKey:   "api_key",
				APIValue: "sk-1234567890abcdef",
				APIIn:    "query",
			},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Execute(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkValidate(b *testing.B) {
	client := New()

	b.Run("Valid", func(b *testing.B) {
		req := &protocol.Request{
			Method: "GET",
			URL:    "https://api.example.com/users?page=1&limit=50",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.Validate(req)
		}
	})

	b.Run("InvalidEmptyURL", func(b *testing.B) {
		req := &protocol.Request{
			Method: "GET",
			URL:    "",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.Validate(req)
		}
	})
}

func BenchmarkBuildTransport(b *testing.B) {
	b.Run("NoProxy", func(b *testing.B) {
		client := New()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.buildTransport("")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HTTPProxy", func(b *testing.B) {
		client := New()
		client.SetProxy("http://proxy.example.com:8080", "")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.buildTransport("")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HTTPProxyWithNoProxy", func(b *testing.B) {
		client := New()
		client.SetProxy("http://proxy.example.com:8080", "localhost,127.0.0.1,.internal.corp")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.buildTransport("")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PerRequestProxy", func(b *testing.B) {
		client := New()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.buildTransport("http://override-proxy.example.com:3128")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkParseNoProxy(b *testing.B) {
	b.Run("Short", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = parseNoProxy("localhost,127.0.0.1")
		}
	})

	b.Run("Long", func(b *testing.B) {
		hosts := strings.Join([]string{
			"localhost", "127.0.0.1", "::1",
			".internal.corp", ".dev.local", ".staging.local",
			"api.example.com", "db.example.com", "cache.example.com",
			"10.0.0.1", "10.0.0.2", "10.0.0.3",
		}, ",")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = parseNoProxy(hosts)
		}
	})
}

func BenchmarkShouldBypassProxy(b *testing.B) {
	hosts := parseNoProxy("localhost,127.0.0.1,.internal.corp,.dev.local,api.example.com")

	b.Run("Match", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = shouldBypassProxy("api.example.com", hosts)
		}
	})

	b.Run("WildcardMatch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = shouldBypassProxy("service.internal.corp", hosts)
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = shouldBypassProxy("external.service.io", hosts)
		}
	})
}
