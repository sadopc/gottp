package har

import (
	"testing"
)

func TestParseHAR(t *testing.T) {
	data := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Browser DevTools", "version": "1.0"},
			"entries": [
				{
					"startedDateTime": "2024-01-01T00:00:00.000Z",
					"time": 150,
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users?page=1",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Accept", "value": "application/json"},
							{"name": "Authorization", "value": "Bearer token123"}
						],
						"queryString": [
							{"name": "page", "value": "1"}
						],
						"headersSize": -1,
						"bodySize": -1
					},
					"response": {
						"status": 200,
						"statusText": "OK",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"content": {
							"size": 42,
							"mimeType": "application/json",
							"text": "{\"users\":[]}"
						},
						"headersSize": -1,
						"bodySize": 42
					}
				},
				{
					"startedDateTime": "2024-01-01T00:00:01.000Z",
					"time": 200,
					"request": {
						"method": "POST",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"queryString": [],
						"postData": {
							"mimeType": "application/json",
							"text": "{\"name\":\"John\"}"
						},
						"headersSize": -1,
						"bodySize": 15
					},
					"response": {
						"status": 201,
						"statusText": "Created",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"content": {
							"size": 10,
							"mimeType": "application/json",
							"text": "{\"id\":1}"
						},
						"headersSize": -1,
						"bodySize": 10
					}
				}
			]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "HAR Import (Browser DevTools)" {
		t.Errorf("expected 'HAR Import (Browser DevTools)', got %q", col.Name)
	}

	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}

	// First entry: GET with query params
	getReq := col.Items[0].Request
	if getReq == nil {
		t.Fatal("expected request for first item")
	}
	if getReq.Method != "GET" {
		t.Errorf("expected GET, got %s", getReq.Method)
	}
	if getReq.URL != "https://api.example.com/users" {
		t.Errorf("expected URL without query string, got %s", getReq.URL)
	}
	if len(getReq.Params) != 1 || getReq.Params[0].Key != "page" {
		t.Errorf("expected 1 query param 'page', got %v", getReq.Params)
	}
	if len(getReq.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(getReq.Headers))
	}

	// Second entry: POST with body
	postReq := col.Items[1].Request
	if postReq == nil {
		t.Fatal("expected request for second item")
	}
	if postReq.Method != "POST" {
		t.Errorf("expected POST, got %s", postReq.Method)
	}
	if postReq.Body == nil {
		t.Fatal("expected body for POST request")
	}
	if postReq.Body.Type != "json" {
		t.Errorf("expected json body type, got %s", postReq.Body.Type)
	}
	if postReq.Body.Content != `{"name":"John"}` {
		t.Errorf("unexpected body content: %s", postReq.Body.Content)
	}
}

func TestParseHAR_PseudoHeaders(t *testing.T) {
	data := []byte(`{
		"log": {
			"version": "1.2",
			"entries": [{
				"startedDateTime": "2024-01-01T00:00:00.000Z",
				"time": 100,
				"request": {
					"method": "GET",
					"url": "https://example.com/",
					"httpVersion": "HTTP/2",
					"headers": [
						{"name": ":method", "value": "GET"},
						{"name": ":path", "value": "/"},
						{"name": "Accept", "value": "*/*"}
					],
					"queryString": [],
					"headersSize": -1,
					"bodySize": -1
				},
				"response": {
					"status": 200,
					"statusText": "OK",
					"httpVersion": "HTTP/2",
					"headers": [],
					"content": {"size": 0, "mimeType": "text/html", "text": ""},
					"headersSize": -1,
					"bodySize": 0
				}
			}]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := col.Items[0].Request
	// Should only have "Accept", not :method or :path
	if len(req.Headers) != 1 {
		t.Errorf("expected 1 header (pseudo-headers filtered), got %d", len(req.Headers))
	}
	if req.Headers[0].Key != "Accept" {
		t.Errorf("expected Accept header, got %s", req.Headers[0].Key)
	}
}

func TestParseHAR_Empty(t *testing.T) {
	_, err := ParseHAR([]byte(`{"log":{"version":"1.2","entries":[]}}`))
	if err == nil {
		t.Error("expected error for empty entries")
	}
}

func TestParseHAR_Invalid(t *testing.T) {
	_, err := ParseHAR([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseHAR_BodyTypes(t *testing.T) {
	tests := []struct {
		mime     string
		expected string
	}{
		{"application/json", "json"},
		{"application/xml", "xml"},
		{"text/xml", "xml"},
		{"application/x-www-form-urlencoded", "form"},
		{"multipart/form-data", "multipart"},
		{"text/plain", "text"},
		{"", "text"},
	}

	for _, tt := range tests {
		got := detectBodyType(tt.mime)
		if got != tt.expected {
			t.Errorf("detectBodyType(%q) = %q, want %q", tt.mime, got, tt.expected)
		}
	}
}
