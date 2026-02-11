package runner

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPrintWorkflowText_PassedWorkflow(t *testing.T) {
	wf := &WorkflowResult{
		Name:    "Smoke",
		Success: true,
		Steps: []Result{
			{
				Name:        "Get Health",
				Method:      "GET",
				StatusCode:  200,
				Duration:    25 * time.Millisecond,
				Size:        12,
				TestsPassed: true,
				TestResults: []TestResult{{Name: "status is 200", Passed: true}},
				ScriptLogs:  []string{"health check complete"},
			},
		},
	}

	var buf bytes.Buffer
	PrintWorkflowText(&buf, wf, true)
	out := buf.String()

	if !strings.Contains(out, "Workflow: Smoke") {
		t.Fatalf("expected workflow header in output, got:\n%s", out)
	}
	if !strings.Contains(out, "status is 200") {
		t.Fatalf("expected test result in output, got:\n%s", out)
	}
	if !strings.Contains(out, "[log] health check complete") {
		t.Fatalf("expected script log in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Workflow passed") {
		t.Fatalf("expected passed summary in output, got:\n%s", out)
	}
}

func TestPrintWorkflowText_FailedWorkflow(t *testing.T) {
	wf := &WorkflowResult{
		Name:    "Broken",
		Success: false,
		Error:   "step failed",
		Steps: []Result{
			{
				Name:     "Create User",
				Method:   "POST",
				Duration: 10 * time.Millisecond,
				Size:     0,
				Error:    contextDeadlineExceededForTests{},
			},
		},
	}

	var buf bytes.Buffer
	PrintWorkflowText(&buf, wf, false)
	out := buf.String()

	if !strings.Contains(out, "Workflow failed: step failed") {
		t.Fatalf("expected failed summary in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Error:") {
		t.Fatalf("expected step error details in output, got:\n%s", out)
	}
}

func TestPrintWorkflowJSON(t *testing.T) {
	wf := &WorkflowResult{Name: "JSON Flow", Success: true}

	var buf bytes.Buffer
	if err := PrintWorkflowJSON(&buf, wf); err != nil {
		t.Fatalf("PrintWorkflowJSON failed: %v", err)
	}

	var decoded WorkflowResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}
	if decoded.Name != "JSON Flow" || !decoded.Success {
		t.Fatalf("unexpected decoded JSON: %+v", decoded)
	}
}

func TestPrintWorkflowJUnit(t *testing.T) {
	wf := &WorkflowResult{
		Name: "JUnit Flow",
		Steps: []Result{
			{
				Name:        "Step One",
				Duration:    20 * time.Millisecond,
				TestsPassed: false,
				TestResults: []TestResult{{Name: "assert", Passed: false, Error: "failed assertion"}},
			},
		},
		Success: false,
	}

	var buf bytes.Buffer
	if err := PrintWorkflowJUnit(&buf, wf); err != nil {
		t.Fatalf("PrintWorkflowJUnit failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "<testsuites") || !strings.Contains(out, "<testsuite") {
		t.Fatalf("expected junit XML structure, got:\n%s", out)
	}
	if !strings.Contains(out, "AssertionFailure") {
		t.Fatalf("expected assertion failure marker, got:\n%s", out)
	}
}

func TestPrintWorkflowJUnit_EmptyFailedWorkflow(t *testing.T) {
	wf := &WorkflowResult{Name: "Empty Failure", Success: false, Error: "workflow failed before steps"}

	var buf bytes.Buffer
	if err := PrintWorkflowJUnit(&buf, wf); err != nil {
		t.Fatalf("PrintWorkflowJUnit failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "WorkflowError") {
		t.Fatalf("expected workflow-level error in junit output, got:\n%s", out)
	}
}

func TestOutputHelpers(t *testing.T) {
	if got := formatSize(0); got != "0 B" {
		t.Fatalf("formatSize(0) = %q", got)
	}
	if got := formatSize(1500); !strings.Contains(got, "KB") {
		t.Fatalf("expected KB format, got %q", got)
	}
	if got := formatSize(3 * 1024 * 1024); !strings.Contains(got, "MB") {
		t.Fatalf("expected MB format, got %q", got)
	}

	if got := formatDuration(500 * time.Microsecond); !strings.Contains(got, "µs") {
		t.Fatalf("expected microsecond formatting, got %q", got)
	}
	if got := formatDuration(250 * time.Millisecond); !strings.Contains(got, "ms") {
		t.Fatalf("expected millisecond formatting, got %q", got)
	}
	if got := formatDuration(2 * time.Second); !strings.Contains(got, "s") {
		t.Fatalf("expected second formatting, got %q", got)
	}

	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate should keep short string, got %q", got)
	}
	if got := truncate("very-long-string", 8); got != "very-..." {
		t.Fatalf("unexpected truncate result: %q", got)
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved"},
		{302, "Found"},
		{304, "Not Modified"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{409, "Conflict"},
		{422, "Unprocessable"},
		{429, "Too Many Requests"},
		{500, "Server Error"},
		{502, "Bad Gateway"},
		{503, "Unavailable"},
		{504, "Gateway Timeout"},
		{999, ""},
	}

	for _, tt := range tests {
		if got := statusText(tt.code); got != tt.want {
			t.Fatalf("statusText(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

type contextDeadlineExceededForTests struct{}

func (contextDeadlineExceededForTests) Error() string {
	return "context deadline exceeded"
}
