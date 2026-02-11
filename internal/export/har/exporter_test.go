package har

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sadopc/gottp/internal/protocol"
)

func TestExport(t *testing.T) {
	req := &protocol.Request{
		Method:  "POST",
		URL:     "https://api.example.com/users",
		Headers: map[string]string{"Content-Type": "application/json"},
		Params:  map[string]string{"version": "2"},
		Body:    []byte(`{"name":"John"}`),
	}
	resp := &protocol.Response{
		StatusCode:  201,
		Status:      "201 Created",
		Headers:     http.Header{"Content-Type": {"application/json"}},
		Body:        []byte(`{"id":1}`),
		ContentType: "application/json",
		Duration:    150 * time.Millisecond,
		Size:        8,
		Proto:       "HTTP/1.1",
		TLS:         true,
		Timing: &protocol.TimingDetail{
			DNSLookup:    10 * time.Millisecond,
			TCPConnect:   15 * time.Millisecond,
			TLSHandshake: 30 * time.Millisecond,
			TTFB:         80 * time.Millisecond,
			Transfer:     5 * time.Millisecond,
			Total:        150 * time.Millisecond,
		},
	}

	data, err := Export(req, resp)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify it's valid JSON
	var har HAR
	if err := json.Unmarshal(data, &har); err != nil {
		t.Fatalf("exported HAR is not valid JSON: %v", err)
	}

	if har.Log.Version != "1.2" {
		t.Errorf("expected version 1.2, got %s", har.Log.Version)
	}
	if har.Log.Creator.Name != "gottp" {
		t.Errorf("expected creator gottp, got %s", har.Log.Creator.Name)
	}
	if len(har.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(har.Log.Entries))
	}

	entry := har.Log.Entries[0]

	// Request
	if entry.Request.Method != "POST" {
		t.Errorf("expected POST, got %s", entry.Request.Method)
	}
	if entry.Request.URL != "https://api.example.com/users" {
		t.Errorf("unexpected URL: %s", entry.Request.URL)
	}
	if entry.Request.PostData == nil {
		t.Fatal("expected PostData")
	}
	if entry.Request.PostData.Text != `{"name":"John"}` {
		t.Errorf("unexpected body: %s", entry.Request.PostData.Text)
	}

	// Response
	if entry.Response.Status != 201 {
		t.Errorf("expected 201, got %d", entry.Response.Status)
	}
	if entry.Response.Content.Text != `{"id":1}` {
		t.Errorf("unexpected response body: %s", entry.Response.Content.Text)
	}

	// Timings
	if entry.Timings.DNS != 10 {
		t.Errorf("expected DNS=10, got %f", entry.Timings.DNS)
	}
	if entry.Timings.Connect != 15 {
		t.Errorf("expected Connect=15, got %f", entry.Timings.Connect)
	}
	if entry.Timings.SSL != 30 {
		t.Errorf("expected SSL=30, got %f", entry.Timings.SSL)
	}
	if entry.Timings.Wait != 80 {
		t.Errorf("expected Wait=80, got %f", entry.Timings.Wait)
	}
}

func TestExportWithoutTiming(t *testing.T) {
	req := &protocol.Request{
		Method: "GET",
		URL:    "https://example.com",
	}
	resp := &protocol.Response{
		StatusCode:  200,
		Status:      "200 OK",
		Headers:     http.Header{},
		Body:        []byte("OK"),
		ContentType: "text/plain",
		Duration:    50 * time.Millisecond,
		Size:        2,
		Proto:       "HTTP/1.1",
	}

	data, err := Export(req, resp)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var har HAR
	if err := json.Unmarshal(data, &har); err != nil {
		t.Fatalf("exported HAR is not valid JSON: %v", err)
	}

	entry := har.Log.Entries[0]
	if entry.Timings.DNS != -1 {
		t.Errorf("expected DNS=-1 when no timing detail, got %f", entry.Timings.DNS)
	}
	if entry.Timings.Wait != 50 {
		t.Errorf("expected Wait=50 (total duration), got %f", entry.Timings.Wait)
	}
}

func TestExportRoundTrip(t *testing.T) {
	// Export then verify the JSON can be parsed by the import parser
	req := &protocol.Request{
		Method:  "GET",
		URL:     "https://api.example.com/health",
		Headers: map[string]string{"Accept": "application/json"},
	}
	resp := &protocol.Response{
		StatusCode:  200,
		Status:      "200 OK",
		Headers:     http.Header{"Content-Type": {"application/json"}},
		Body:        []byte(`{"status":"ok"}`),
		ContentType: "application/json",
		Duration:    25 * time.Millisecond,
		Size:        15,
		Proto:       "HTTP/1.1",
	}

	data, err := Export(req, resp)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify the structure matches HAR 1.2 schema requirements
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal("not valid JSON object")
	}
	if _, ok := raw["log"]; !ok {
		t.Error("missing 'log' key")
	}

	var har HAR
	json.Unmarshal(data, &har)

	if har.Log.Version != "1.2" {
		t.Errorf("version should be 1.2, got %s", har.Log.Version)
	}
	if len(har.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(har.Log.Entries))
	}
	if har.Log.Entries[0].Request.Method != "GET" {
		t.Error("method should be GET")
	}
}
