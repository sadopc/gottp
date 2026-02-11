package scripting

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPreScriptMutation(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	req := &ScriptRequest{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: map[string]string{},
	}

	result := engine.RunPreScript(`
		gottp.request.SetHeader("X-Custom", "test-value");
		gottp.request.SetURL("https://modified.com");
		gottp.log("pre-script ran");
	`, req, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if req.Headers["X-Custom"] != "test-value" {
		t.Errorf("expected X-Custom header, got %v", req.Headers)
	}
	if req.URL != "https://modified.com" {
		t.Errorf("expected modified URL, got %s", req.URL)
	}
	if len(result.Logs) != 1 || result.Logs[0] != "pre-script ran" {
		t.Errorf("expected log entry, got %v", result.Logs)
	}
}

func TestPostScriptAssertions(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	req := &ScriptRequest{}
	resp := &ScriptResponse{
		StatusCode: 200,
		Body:       `{"ok":true}`,
	}

	result := engine.RunPostScript(`
		gottp.test("status 200", function() {
			gottp.assert(gottp.response.StatusCode === 200);
		});
		gottp.test("has body", function() {
			gottp.assert(gottp.response.Body.length > 0);
		});
	`, req, resp, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.TestResults) != 2 {
		t.Fatalf("expected 2 test results, got %d", len(result.TestResults))
	}
	for _, tr := range result.TestResults {
		if !tr.Passed {
			t.Errorf("test %q failed: %s", tr.Name, tr.Error)
		}
	}
}

func TestPostScriptFailedAssertion(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	resp := &ScriptResponse{StatusCode: 404}

	result := engine.RunPostScript(`
		gottp.test("should fail", function() {
			gottp.assert(gottp.response.StatusCode === 200, "expected 200");
		});
	`, &ScriptRequest{}, resp, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.TestResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.TestResults))
	}
	if result.TestResults[0].Passed {
		t.Error("expected test to fail")
	}
}

func TestScriptTimeout(t *testing.T) {
	engine := NewEngine(500 * time.Millisecond)

	result := engine.RunPreScript(`while(true){}`, &ScriptRequest{}, nil)
	if result.Err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestEnvVarRoundTrip(t *testing.T) {
	engine := NewEngine(5 * time.Second)

	result := engine.RunPreScript(`
		gottp.setEnvVar("token", "abc123");
		var val = gottp.getEnvVar("token");
		gottp.log(val);
	`, &ScriptRequest{}, map[string]string{"existing": "value"})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.EnvChanges["token"] != "abc123" {
		t.Errorf("expected token=abc123, got %v", result.EnvChanges)
	}
	if len(result.Logs) != 1 || result.Logs[0] != "abc123" {
		t.Errorf("expected log abc123, got %v", result.Logs)
	}
}

func TestUtilityFunctions(t *testing.T) {
	engine := NewEngine(5 * time.Second)

	result := engine.RunPreScript(`
		var encoded = gottp.base64encode("hello");
		var decoded = gottp.base64decode(encoded);
		gottp.log(decoded);
		var hash = gottp.sha256("test");
		gottp.log(hash);
		var id = gottp.uuid();
		gottp.log(id);
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(result.Logs))
	}
	if result.Logs[0] != "hello" {
		t.Errorf("expected hello, got %s", result.Logs[0])
	}
	// SHA-256 of "test"
	if len(result.Logs[1]) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(result.Logs[1]))
	}
	// UUID should be 36 chars
	if len(result.Logs[2]) != 36 {
		t.Errorf("expected 36 char UUID, got %d", len(result.Logs[2]))
	}
}

func TestMD5(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var hash = gottp.md5("hello");
		gottp.log(hash);
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	// MD5 of "hello" is 5d41402abc4b2a76b9719d911017c592
	if result.Logs[0] != "5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("unexpected MD5: %s", result.Logs[0])
	}
}

func TestHMACSha256(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var mac = gottp.hmacSha256("message", "secret");
		gottp.log(mac);
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Logs[0]) != 64 {
		t.Errorf("expected 64 char HMAC, got %d", len(result.Logs[0]))
	}
}

func TestTimestamp(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var ts = gottp.timestamp();
		gottp.log(String(ts));
		var tsMs = gottp.timestampMs();
		gottp.log(String(tsMs));
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(result.Logs))
	}
	// Timestamps should be numeric strings
	if len(result.Logs[0]) < 10 {
		t.Errorf("timestamp too short: %s", result.Logs[0])
	}
	if len(result.Logs[1]) < 13 {
		t.Errorf("timestamp ms too short: %s", result.Logs[1])
	}
}

func TestRandomInt(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var r = gottp.randomInt(0, 10);
		gottp.log(String(r));
		var r2 = gottp.randomInt(100);
		gottp.log(String(r2));
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(result.Logs))
	}
}

func TestSleep(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	start := time.Now()
	result := engine.RunPreScript(`
		gottp.sleep(100);
	`, &ScriptRequest{}, nil)
	elapsed := time.Since(start)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if elapsed < 80*time.Millisecond {
		t.Errorf("sleep too short: %v", elapsed)
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	os.WriteFile(path, []byte(`{"key":"value"}`), 0644)

	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var content = gottp.readFile("`+path+`");
		gottp.log(content);
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Logs[0] != `{"key":"value"}` {
		t.Errorf("unexpected file content: %s", result.Logs[0])
	}
}

func TestReadFileNotFound(t *testing.T) {
	engine := NewEngine(5 * time.Second)
	result := engine.RunPreScript(`
		var content = gottp.readFile("/nonexistent/file.txt");
		gottp.log(String(content));
	`, &ScriptRequest{}, nil)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Logs[0] != "undefined" {
		t.Errorf("expected undefined for missing file, got %s", result.Logs[0])
	}
}
