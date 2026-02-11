package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/serdar/gottp/internal/core/collection"
)

func testCollection() *collection.Collection {
	return &collection.Collection{
		Name:    "Test API",
		Version: "1",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID:       "1",
					Name:     "Get Users",
					Protocol: "http",
					Method:   "GET",
					URL:      "https://api.example.com/users",
					Body: &collection.Body{
						Type:    "json",
						Content: `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`,
					},
				},
			},
			{
				Request: &collection.Request{
					ID:       "2",
					Name:     "Create User",
					Protocol: "http",
					Method:   "POST",
					URL:      "https://api.example.com/users",
					Body: &collection.Body{
						Type:    "json",
						Content: `{"id":3,"name":"Charlie"}`,
					},
				},
			},
			{
				Request: &collection.Request{
					ID:       "3",
					Name:     "Get Health",
					Protocol: "http",
					Method:   "GET",
					URL:      "https://api.example.com/health",
					Body: &collection.Body{
						Type:    "text",
						Content: "OK",
					},
				},
			},
			{
				Folder: &collection.Folder{
					Name: "Admin",
					Items: []collection.Item{
						{
							Request: &collection.Request{
								ID:       "4",
								Name:     "Delete User",
								Protocol: "http",
								Method:   "DELETE",
								URL:      "https://api.example.com/users/1",
							},
						},
					},
				},
			},
			{
				Request: &collection.Request{
					ID:       "5",
					Name:     "XML Response",
					Protocol: "http",
					Method:   "GET",
					URL:      "https://api.example.com/data.xml",
					Body: &collection.Body{
						Type:    "xml",
						Content: `<data><item>hello</item></data>`,
					},
				},
			},
		},
	}
}

func TestRouteMatching(t *testing.T) {
	srv := New(testCollection())

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"GET /users", "GET", "/users", http.StatusOK},
		{"POST /users", "POST", "/users", http.StatusOK},
		{"DELETE /users/1", "DELETE", "/users/1", http.StatusOK},
		{"GET /health", "GET", "/health", http.StatusOK},
		{"GET /data.xml", "GET", "/data.xml", http.StatusOK},
		{"unmatched path", "GET", "/nonexistent", http.StatusNotFound},
		{"wrong method", "PUT", "/users", http.StatusNotFound},
	}

	handler := srv.Handler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRouteMatchingWithQueryParams(t *testing.T) {
	srv := New(testCollection())
	handler := srv.Handler()

	// Query params should not affect path matching
	req := httptest.NewRequest("GET", "/users?page=1&limit=10", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestResponseBodyServing(t *testing.T) {
	srv := New(testCollection())
	handler := srv.Handler()

	tests := []struct {
		name     string
		method   string
		path     string
		wantBody string
	}{
		{"GET /users returns JSON array", "GET", "/users", `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`},
		{"POST /users returns JSON object", "POST", "/users", `{"id":3,"name":"Charlie"}`},
		{"GET /health returns text", "GET", "/health", "OK"},
		{"DELETE /users/1 returns empty", "DELETE", "/users/1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			body := rec.Body.String()
			if body != tt.wantBody {
				t.Errorf("got body %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestContentTypeHeaders(t *testing.T) {
	srv := New(testCollection())
	handler := srv.Handler()

	tests := []struct {
		name   string
		method string
		path   string
		wantCT string
	}{
		{"JSON response", "GET", "/users", "application/json"},
		{"text response", "GET", "/health", "text/plain"},
		{"XML response", "GET", "/data.xml", "application/xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			ct := rec.Header().Get("Content-Type")
			if ct != tt.wantCT {
				t.Errorf("got Content-Type %q, want %q", ct, tt.wantCT)
			}
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	srv := New(testCollection())
	handler := srv.Handler()

	// Test normal request has CORS
	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("got ACAO %q, want %q", origin, "*")
	}
	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
	if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}

	// Test OPTIONS preflight
	req = httptest.NewRequest("OPTIONS", "/users", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS got status %d, want %d", rec.Code, http.StatusNoContent)
	}
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("OPTIONS got ACAO %q, want %q", origin, "*")
	}
}

func TestCustomCORSOrigin(t *testing.T) {
	srv := New(testCollection(), WithCORSOrigin("https://myapp.example.com"))
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "https://myapp.example.com" {
		t.Errorf("got ACAO %q, want %q", origin, "https://myapp.example.com")
	}
}

func TestLatencySimulation(t *testing.T) {
	latency := 50 * time.Millisecond
	srv := New(testCollection(), WithLatency(latency))
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if elapsed < latency {
		t.Errorf("request took %v, expected at least %v", elapsed, latency)
	}
}

func TestErrorRateSimulation(t *testing.T) {
	// With error rate 1.0, every request should return 500
	srv := New(testCollection(), WithErrorRate(1.0))
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d with error rate 1.0", rec.Code, http.StatusInternalServerError)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] != "Simulated server error" {
		t.Errorf("got error %q, want %q", resp["error"], "Simulated server error")
	}
}

func TestErrorRateZero(t *testing.T) {
	// With error rate 0.0, no requests should return 500
	srv := New(testCollection(), WithErrorRate(0.0))
	handler := srv.Handler()

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/users", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusInternalServerError {
			t.Fatal("got 500 with error rate 0.0")
		}
	}
}

func TestNotFoundWithRouteListing(t *testing.T) {
	srv := New(testCollection())
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}

	var resp map[string]interface{}
	body, _ := io.ReadAll(rec.Body)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode 404 response: %v (body: %s)", err, string(body))
	}

	if resp["error"] != "Route not found" {
		t.Errorf("got error %q, want %q", resp["error"], "Route not found")
	}

	routes, ok := resp["available_routes"].([]interface{})
	if !ok {
		t.Fatal("expected available_routes to be an array")
	}
	if len(routes) == 0 {
		t.Error("expected available_routes to be non-empty")
	}

	// Verify at least one route is listed
	firstRoute := routes[0].(map[string]interface{})
	if _, ok := firstRoute["method"]; !ok {
		t.Error("expected route to have 'method' field")
	}
	if _, ok := firstRoute["path"]; !ok {
		t.Error("expected route to have 'path' field")
	}
}

func TestDynamicTemplateVariables(t *testing.T) {
	col := &collection.Collection{
		Name:    "Template Test",
		Version: "1",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID:       "1",
					Name:     "Dynamic",
					Protocol: "http",
					Method:   "GET",
					URL:      "https://api.example.com/dynamic",
					Body: &collection.Body{
						Type:    "json",
						Content: `{"ts":"{{$timestamp}}","id":"{{$uuid}}","num":{{$randomInt}}}`,
					},
				},
			},
		},
	}

	srv := New(col)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/dynamic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should not contain template variables
	if strings.Contains(body, "{{$timestamp}}") {
		t.Error("body still contains {{$timestamp}}")
	}
	if strings.Contains(body, "{{$uuid}}") {
		t.Error("body still contains {{$uuid}}")
	}
	if strings.Contains(body, "{{$randomInt}}") {
		t.Error("body still contains {{$randomInt}}")
	}

	// Parse as JSON to verify it's valid
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Errorf("expanded body is not valid JSON: %v (body: %s)", err, body)
	}

	if _, ok := result["ts"]; !ok {
		t.Error("expected 'ts' field in response")
	}
	if _, ok := result["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
	if _, ok := result["num"]; !ok {
		t.Error("expected 'num' field in response")
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://api.example.com/users", "/users"},
		{"https://api.example.com/users/1", "/users/1"},
		{"https://api.example.com/", "/"},
		{"https://api.example.com", "/"},
		{"http://localhost:3000/api/v1/items", "/api/v1/items"},
		{"/users", "/users"},
		{"{{baseUrl}}/users", "/users"},
		{"", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractPath(tt.url)
			if got != tt.want {
				t.Errorf("extractPath(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/users/", "/users"},
		{"/users", "/users"},
		{"users", "/users"},
		{"/", "/"},
		{"", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{`{"key": "value"}`, "application/json"},
		{`[1, 2, 3]`, "application/json"},
		{`<root><item/></root>`, "application/xml"},
		{`hello world`, "text/plain"},
		{``, "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			got := detectContentType(tt.body)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestRoutesFromNestedFolders(t *testing.T) {
	col := &collection.Collection{
		Name: "Nested",
		Items: []collection.Item{
			{
				Folder: &collection.Folder{
					Name: "Level 1",
					Items: []collection.Item{
						{
							Folder: &collection.Folder{
								Name: "Level 2",
								Items: []collection.Item{
									{
										Request: &collection.Request{
											ID:       "deep",
											Name:     "Deep Request",
											Protocol: "http",
											Method:   "GET",
											URL:      "https://api.example.com/deep/path",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	srv := New(col)
	routes := srv.Routes()

	if len(routes) != 1 {
		t.Fatalf("got %d routes, want 1", len(routes))
	}
	if routes[0].path != "/deep/path" {
		t.Errorf("got path %q, want %q", routes[0].path, "/deep/path")
	}
}

func TestWebSocketAndGRPCSkipped(t *testing.T) {
	col := &collection.Collection{
		Name: "Multi-Protocol",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID:       "ws",
					Name:     "WebSocket",
					Protocol: "websocket",
					Method:   "GET",
					URL:      "ws://localhost/ws",
				},
			},
			{
				Request: &collection.Request{
					ID:       "grpc",
					Name:     "gRPC",
					Protocol: "grpc",
					Method:   "POST",
					URL:      "localhost:50051",
				},
			},
			{
				Request: &collection.Request{
					ID:       "http",
					Name:     "HTTP",
					Protocol: "http",
					Method:   "GET",
					URL:      "https://api.example.com/ok",
				},
			},
		},
	}

	srv := New(col)
	routes := srv.Routes()

	if len(routes) != 1 {
		t.Fatalf("got %d routes, want 1 (only HTTP)", len(routes))
	}
	if routes[0].path != "/ok" {
		t.Errorf("got path %q, want /ok", routes[0].path)
	}
}

func TestWithPortOption(t *testing.T) {
	srv := New(testCollection(), WithPort(9090))
	if srv.Port() != 9090 {
		t.Errorf("got port %d, want 9090", srv.Port())
	}
}

func TestWithErrorRateClamping(t *testing.T) {
	srv := New(testCollection(), WithErrorRate(2.0))
	if srv.errorRate != 1.0 {
		t.Errorf("got error rate %f, want 1.0 (clamped)", srv.errorRate)
	}

	srv = New(testCollection(), WithErrorRate(-0.5))
	if srv.errorRate != 0.0 {
		t.Errorf("got error rate %f, want 0.0 (clamped)", srv.errorRate)
	}
}

func TestEmptyCollection(t *testing.T) {
	col := &collection.Collection{
		Name: "Empty",
	}

	srv := New(col)
	if len(srv.Routes()) != 0 {
		t.Errorf("got %d routes, want 0", len(srv.Routes()))
	}

	handler := srv.Handler()
	req := httptest.NewRequest("GET", "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("got status %d, want 404", rec.Code)
	}
}

func TestDefaultMethodIsGET(t *testing.T) {
	col := &collection.Collection{
		Name: "No Method",
		Items: []collection.Item{
			{
				Request: &collection.Request{
					ID:       "1",
					Name:     "No Method",
					Protocol: "http",
					URL:      "https://api.example.com/default",
				},
			},
		},
	}

	srv := New(col)
	routes := srv.Routes()

	if len(routes) != 1 {
		t.Fatalf("got %d routes, want 1", len(routes))
	}
	if routes[0].method != "GET" {
		t.Errorf("got method %q, want GET", routes[0].method)
	}
}
