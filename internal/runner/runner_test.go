package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/core/environment"
	"github.com/sadopc/gottp/internal/protocol"
	httpclient "github.com/sadopc/gottp/internal/protocol/http"
	"github.com/sadopc/gottp/internal/scripting"
)

func TestBuildProtocolRequest(t *testing.T) {
	colReq := &collection.Request{
		Name:     "Test Request",
		Protocol: "http",
		Method:   "POST",
		URL:      "https://example.com/api",
		Params: []collection.KVPair{
			{Key: "page", Value: "1", Enabled: true},
			{Key: "disabled", Value: "x", Enabled: false},
		},
		Headers: []collection.KVPair{
			{Key: "Content-Type", Value: "application/json", Enabled: true},
		},
		Body: &collection.Body{Type: "json", Content: `{"key":"value"}`},
		Auth: &collection.Auth{
			Type:   "bearer",
			Bearer: &collection.BearerAuth{Token: "my-token"},
		},
	}

	req := buildProtocolRequest(colReq)

	if req.Protocol != "http" {
		t.Errorf("expected protocol http, got %s", req.Protocol)
	}
	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}
	if req.URL != "https://example.com/api" {
		t.Errorf("unexpected URL: %s", req.URL)
	}
	if req.Params["page"] != "1" {
		t.Errorf("expected param page=1, got %s", req.Params["page"])
	}
	if _, ok := req.Params["disabled"]; ok {
		t.Error("disabled param should not be included")
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %s", req.Headers["Content-Type"])
	}
	if string(req.Body) != `{"key":"value"}` {
		t.Errorf("unexpected body: %s", string(req.Body))
	}
	if req.Auth == nil || req.Auth.Type != "bearer" || req.Auth.Token != "my-token" {
		t.Error("unexpected auth config")
	}
}

func TestBuildAuthConfig(t *testing.T) {
	tests := []struct {
		name string
		auth *collection.Auth
		want *protocol.AuthConfig
	}{
		{"nil auth", nil, nil},
		{"none auth", &collection.Auth{Type: "none"}, nil},
		{"basic auth", &collection.Auth{
			Type:  "basic",
			Basic: &collection.BasicAuth{Username: "user", Password: "pass"},
		}, &protocol.AuthConfig{Type: "basic", Username: "user", Password: "pass"}},
		{"bearer auth", &collection.Auth{
			Type:   "bearer",
			Bearer: &collection.BearerAuth{Token: "tok"},
		}, &protocol.AuthConfig{Type: "bearer", Token: "tok"}},
		{"apikey auth", &collection.Auth{
			Type:   "apikey",
			APIKey: &collection.APIKeyAuth{Key: "X-API-Key", Value: "secret", In: "header"},
		}, &protocol.AuthConfig{Type: "apikey", APIKey: "X-API-Key", APIValue: "secret", APIIn: "header"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAuthConfig(tt.auth)
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil auth config")
			}
			if got.Type != tt.want.Type {
				t.Errorf("type: got %s, want %s", got.Type, tt.want.Type)
			}
			if got.Username != tt.want.Username {
				t.Errorf("username: got %s, want %s", got.Username, tt.want.Username)
			}
			if got.Password != tt.want.Password {
				t.Errorf("password: got %s, want %s", got.Password, tt.want.Password)
			}
			if got.Token != tt.want.Token {
				t.Errorf("token: got %s, want %s", got.Token, tt.want.Token)
			}
			if got.APIKey != tt.want.APIKey {
				t.Errorf("apikey: got %s, want %s", got.APIKey, tt.want.APIKey)
			}
		})
	}
}

func TestResolveVars(t *testing.T) {
	r := &Runner{
		envVars: map[string]string{"host": "example.com", "token": "abc123"},
		colVars: map[string]string{"version": "v1"},
	}

	req := &protocol.Request{
		URL:     "https://{{host}}/api/{{version}}/users",
		Headers: map[string]string{"Authorization": "Bearer {{token}}"},
		Params:  map[string]string{"q": "{{host}}"},
		Body:    []byte(`{"host":"{{host}}"}`),
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "{{token}}",
		},
	}

	r.resolveVars(req)

	if req.URL != "https://example.com/api/v1/users" {
		t.Errorf("URL not resolved: %s", req.URL)
	}
	if req.Headers["Authorization"] != "Bearer abc123" {
		t.Errorf("header not resolved: %s", req.Headers["Authorization"])
	}
	if req.Params["q"] != "example.com" {
		t.Errorf("param not resolved: %s", req.Params["q"])
	}
	if string(req.Body) != `{"host":"example.com"}` {
		t.Errorf("body not resolved: %s", string(req.Body))
	}
	if req.Auth.Token != "abc123" {
		t.Errorf("auth token not resolved: %s", req.Auth.Token)
	}
}

func TestCollectRequests(t *testing.T) {
	r := &Runner{
		collection: &collection.Collection{
			Items: []collection.Item{
				{Request: &collection.Request{Name: "Get Users", Method: "GET", URL: "/users"}},
				{Folder: &collection.Folder{
					Name: "Auth",
					Items: []collection.Item{
						{Request: &collection.Request{Name: "Login", Method: "POST", URL: "/login"}},
						{Request: &collection.Request{Name: "Logout", Method: "POST", URL: "/logout"}},
					},
				}},
				{Request: &collection.Request{Name: "Health", Method: "GET", URL: "/health"}},
			},
		},
	}

	// All requests
	all := r.collectRequests(Config{})
	if len(all) != 4 {
		t.Errorf("expected 4 requests, got %d", len(all))
	}

	// By name
	byName := r.collectRequests(Config{RequestName: "Login"})
	if len(byName) != 1 {
		t.Fatalf("expected 1 request, got %d", len(byName))
	}
	if byName[0].Name != "Login" {
		t.Errorf("expected Login, got %s", byName[0].Name)
	}

	// Case-insensitive name
	byNameCI := r.collectRequests(Config{RequestName: "login"})
	if len(byNameCI) != 1 {
		t.Fatalf("expected 1 request (case-insensitive), got %d", len(byNameCI))
	}

	// By folder
	byFolder := r.collectRequests(Config{FolderName: "Auth"})
	if len(byFolder) != 2 {
		t.Errorf("expected 2 requests in Auth folder, got %d", len(byFolder))
	}

	// Non-existent
	none := r.collectRequests(Config{RequestName: "Nonexistent"})
	if len(none) != 0 {
		t.Errorf("expected 0 requests, got %d", len(none))
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    int
	}{
		{"all success", []Result{{TestsPassed: true}}, 0},
		{"test failure", []Result{{TestsPassed: false}}, 1},
		{"request error", []Result{{Error: http.ErrAbortHandler, TestsPassed: true}}, 2},
		{"error takes priority", []Result{
			{Error: http.ErrAbortHandler, TestsPassed: false},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCode(tt.results)
			if got != tt.want {
				t.Errorf("ExitCode = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunWithTestServer(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`[{"id":1,"name":"Alice"}]`))
		case "/error":
			w.WriteHeader(500)
			w.Write([]byte("internal server error"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	// Build runner manually (bypasses file loading)
	registry := protocol.NewRegistry()
	registry.Register(httpclient.New())

	r := &Runner{
		collection: &collection.Collection{
			Items: []collection.Item{
				{Request: &collection.Request{
					Name:     "Get Users",
					Protocol: "http",
					Method:   "GET",
					URL:      server.URL + "/users",
				}},
				{Request: &collection.Request{
					Name:     "Server Error",
					Protocol: "http",
					Method:   "GET",
					URL:      server.URL + "/error",
				}},
			},
		},
		registry:     registry,
		scriptEngine: scripting.NewEngine(5 * time.Second),
		envVars:      map[string]string{},
		colVars:      map[string]string{},
		timeout:      10 * time.Second,
	}

	results, err := r.Run(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First request should succeed
	if results[0].StatusCode != 200 {
		t.Errorf("expected status 200, got %d", results[0].StatusCode)
	}
	if results[0].Error != nil {
		t.Errorf("unexpected error: %v", results[0].Error)
	}

	// Second request should return 500
	if results[1].StatusCode != 500 {
		t.Errorf("expected status 500, got %d", results[1].StatusCode)
	}
}

func TestRunWithScripts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	registry := protocol.NewRegistry()
	registry.Register(httpclient.New())

	r := &Runner{
		collection: &collection.Collection{
			Items: []collection.Item{
				{Request: &collection.Request{
					Name:     "Scripted Request",
					Protocol: "http",
					Method:   "GET",
					URL:      server.URL,
					PostScript: `
						gottp.log("status code: " + gottp.response.StatusCode);
						gottp.test("status is 200", function() {
							gottp.assert(gottp.response.StatusCode === 200, "expected 200");
						});
						gottp.test("body contains ok", function() {
							gottp.assert(gottp.response.Body.indexOf("ok") > -1, "expected ok in body");
						});
					`,
				}},
			},
		},
		registry:     registry,
		scriptEngine: scripting.NewEngine(5 * time.Second),
		envVars:      map[string]string{},
		colVars:      map[string]string{},
		timeout:      10 * time.Second,
	}

	results, err := r.Run(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if !res.TestsPassed {
		t.Errorf("expected tests to pass, testResults=%+v", res.TestResults)
	}
	if len(res.TestResults) != 2 {
		t.Fatalf("expected 2 test results, got %d", len(res.TestResults))
	}
	if !res.TestResults[0].Passed || !res.TestResults[1].Passed {
		t.Error("expected both tests to pass")
	}
	if len(res.ScriptLogs) != 1 {
		t.Errorf("expected 1 log, got %d", len(res.ScriptLogs))
	}
}

func TestRunWithEnvResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	registry := protocol.NewRegistry()
	registry.Register(httpclient.New())

	r := &Runner{
		collection: &collection.Collection{
			Items: []collection.Item{
				{Request: &collection.Request{
					Name:     "Auth Request",
					Protocol: "http",
					Method:   "GET",
					URL:      "{{base_url}}/api",
					Headers: []collection.KVPair{
						{Key: "Authorization", Value: "Bearer {{api_token}}", Enabled: true},
					},
				}},
			},
		},
		registry:     registry,
		scriptEngine: scripting.NewEngine(5 * time.Second),
		envVars:      map[string]string{"base_url": server.URL, "api_token": "test-token"},
		colVars:      map[string]string{},
		timeout:      10 * time.Second,
	}

	results, err := r.Run(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if results[0].StatusCode != 200 {
		t.Errorf("expected 200, got %d (env vars may not be resolving)", results[0].StatusCode)
	}
}

func TestNewFromFile(t *testing.T) {
	// Create temp collection file
	dir := t.TempDir()
	colPath := filepath.Join(dir, "test.gottp.yaml")
	colContent := `name: Test
version: "1"
items:
  - request:
      name: Hello
      method: GET
      url: https://example.com
`
	if err := os.WriteFile(colPath, []byte(colContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create environments file
	envPath := filepath.Join(dir, "environments.yaml")
	envContent := `environments:
  - name: dev
    variables:
      host:
        value: localhost
  - name: prod
    variables:
      host:
        value: example.com
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading with specific environment
	runner, err := New(Config{CollectionPath: colPath, Environment: "prod"})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if runner.envVars["host"] != "example.com" {
		t.Errorf("expected host=example.com, got %s", runner.envVars["host"])
	}

	// Test loading with non-existent environment
	_, err = New(Config{CollectionPath: colPath, Environment: "staging"})
	if err == nil {
		t.Error("expected error for non-existent environment")
	}

	// Test auto-select first environment
	runner2, err := New(Config{CollectionPath: colPath})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if runner2.envVars["host"] != "localhost" {
		t.Errorf("expected host=localhost (auto-selected dev), got %s", runner2.envVars["host"])
	}
}

func TestPrintText(t *testing.T) {
	var buf bytes.Buffer
	results := []Result{
		{
			Name:        "Get Users",
			Method:      "GET",
			URL:         "https://api.example.com/users",
			StatusCode:  200,
			Status:      "200 OK",
			Duration:    145 * time.Millisecond,
			Size:        2300,
			TestsPassed: true,
			TestResults: []TestResult{
				{Name: "status is 200", Passed: true},
			},
		},
		{
			Name:        "Create User",
			Method:      "POST",
			URL:         "https://api.example.com/users",
			StatusCode:  500,
			Status:      "500 Internal Server Error",
			Duration:    89 * time.Millisecond,
			Size:        100,
			TestsPassed: false,
			TestResults: []TestResult{
				{Name: "status is 201", Passed: false, Error: "expected 201"},
			},
		},
	}

	PrintText(&buf, results, false)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("Get Users")) {
		t.Error("expected Get Users in output")
	}
	if !bytes.Contains([]byte(output), []byte("Create User")) {
		t.Error("expected Create User in output")
	}
	if !bytes.Contains([]byte(output), []byte("1 passed")) {
		t.Error("expected '1 passed' in output")
	}
	if !bytes.Contains([]byte(output), []byte("1 failed")) {
		t.Error("expected '1 failed' in output")
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	results := []Result{
		{
			Name:        "Test",
			Method:      "GET",
			URL:         "https://example.com",
			StatusCode:  200,
			Duration:    100 * time.Millisecond,
			TestsPassed: true,
		},
	}

	if err := PrintJSON(&buf, results); err != nil {
		t.Fatal(err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 result in JSON, got %d", len(parsed))
	}
}

func TestPrintJUnit(t *testing.T) {
	var buf bytes.Buffer
	results := []Result{
		{
			Name:        "Scripted",
			Method:      "GET",
			URL:         "https://example.com",
			StatusCode:  200,
			Duration:    150 * time.Millisecond,
			TestsPassed: false,
			TestResults: []TestResult{
				{Name: "passes", Passed: true},
				{Name: "fails", Passed: false, Error: "bad value"},
			},
		},
	}

	if err := PrintJUnit(&buf, results); err != nil {
		t.Fatal(err)
	}

	// Verify XML is valid
	var suites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &suites); err != nil {
		t.Fatalf("invalid JUnit XML: %v", err)
	}
	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.Suites))
	}
	suite := suites.Suites[0]
	if suite.Tests != 2 {
		t.Errorf("expected 2 tests, got %d", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
	}
}

// Ensure environment.KVPair is distinct (resolver has its own copy to avoid circular imports).
func TestResolverKVPairReuse(t *testing.T) {
	envVars := map[string]string{"host": "api.example.com"}
	colVars := map[string]string{}

	result := environment.Resolve("https://{{host}}/v1", envVars, colVars)
	if result != "https://api.example.com/v1" {
		t.Errorf("unexpected resolution: %s", result)
	}
}
