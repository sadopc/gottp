package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/protocol"
	httpclient "github.com/sadopc/gottp/internal/protocol/http"
	"github.com/sadopc/gottp/internal/scripting"
)

func newWorkflowRunner(col *collection.Collection) *Runner {
	registry := protocol.NewRegistry()
	registry.Register(httpclient.New())

	return &Runner{
		collection:   col,
		registry:     registry,
		scriptEngine: scripting.NewEngine(2 * time.Second),
		envVars:      map[string]string{},
		colVars:      map[string]string{},
		timeout:      2 * time.Second,
	}
}

func TestRunWorkflow_NoCollection(t *testing.T) {
	r := &Runner{}
	_, err := r.RunWorkflow(context.Background(), "Any", false)
	if err == nil || !strings.Contains(err.Error(), "no collection loaded") {
		t.Fatalf("expected no collection error, got %v", err)
	}
}

func TestRunWorkflow_NotFoundErrors(t *testing.T) {
	rNoWorkflows := newWorkflowRunner(&collection.Collection{Name: "No workflows", Items: nil})
	_, err := rNoWorkflows.RunWorkflow(context.Background(), "Missing", false)
	if err == nil || !strings.Contains(err.Error(), "no workflows defined") {
		t.Fatalf("expected no workflows error, got %v", err)
	}

	rWithWorkflows := newWorkflowRunner(&collection.Collection{
		Name: "Has workflows",
		Workflows: []collection.Workflow{
			{Name: "Smoke"},
			{Name: "Regression"},
		},
	})
	_, err = rWithWorkflows.RunWorkflow(context.Background(), "Missing", false)
	if err == nil || !strings.Contains(err.Error(), "workflow \"Missing\" not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Smoke") || !strings.Contains(err.Error(), "Regression") {
		t.Fatalf("expected available workflow names in error, got %v", err)
	}
}

func TestRunWorkflow_SuccessWithExtractAndCondition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"abc123"}`))
		case "/second":
			if r.URL.Query().Get("token") != "abc123" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"missing token"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	first := collection.NewRequest("First", "GET", server.URL+"/first")
	second := collection.NewRequest("Second", "GET", server.URL+"/second?token={{token}}")

	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: first},
			{Request: second},
		},
		Workflows: []collection.Workflow{
			{
				Name: "Happy Path",
				Steps: []collection.WorkflowStep{
					{
						Request:   "First",
						Extracts:  map[string]string{"token": "$.token"},
						Condition: "status == 200",
					},
					{
						Request:   "Second",
						Condition: "success",
					},
				},
			},
		},
	}

	r := newWorkflowRunner(col)
	res, err := r.RunWorkflow(context.Background(), "happy path", true)
	if err != nil {
		t.Fatalf("RunWorkflow failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected workflow success, got failure: %s", res.Error)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(res.Steps))
	}
	if got := r.envVars["token"]; got != "abc123" {
		t.Fatalf("expected extracted token abc123, got %q", got)
	}
}

func TestRunWorkflow_RequestNotFoundInStep(t *testing.T) {
	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: collection.NewRequest("Existing", "GET", "https://example.com")},
		},
		Workflows: []collection.Workflow{
			{
				Name: "Broken",
				Steps: []collection.WorkflowStep{
					{Request: "MissingRequest"},
				},
			},
		},
	}

	r := newWorkflowRunner(col)
	res, err := r.RunWorkflow(context.Background(), "Broken", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected workflow to fail when step request is missing")
	}
	if !strings.Contains(res.Error, "request \"MissingRequest\" not found") {
		t.Fatalf("unexpected error message: %q", res.Error)
	}
}

func TestRunWorkflow_RequestExecutionFailure(t *testing.T) {
	bad := collection.NewRequest("Bad", "GET", "://bad-url")
	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: bad},
		},
		Workflows: []collection.Workflow{{
			Name:  "BrokenRequest",
			Steps: []collection.WorkflowStep{{Request: "Bad"}},
		}},
	}

	r := newWorkflowRunner(col)
	res, err := r.RunWorkflow(context.Background(), "BrokenRequest", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected workflow to fail")
	}
	if !strings.Contains(res.Error, "failed") {
		t.Fatalf("expected failed error message, got %q", res.Error)
	}
}

func TestRunWorkflow_ConditionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	first := collection.NewRequest("First", "GET", server.URL)
	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: first},
		},
		Workflows: []collection.Workflow{{
			Name: "ConditionFail",
			Steps: []collection.WorkflowStep{{
				Request:   "First",
				Condition: "status == 404",
			}},
		}},
	}

	r := newWorkflowRunner(col)
	res, err := r.RunWorkflow(context.Background(), "ConditionFail", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected condition failure")
	}
	if !strings.Contains(res.Error, "condition failed") {
		t.Fatalf("unexpected condition failure message: %q", res.Error)
	}
}

func TestRunWorkflow_TestAssertionFailureMarksWorkflowUnsuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	req := collection.NewRequest("Scripted", "GET", server.URL)
	req.PostScript = `gottp.test("failing test", function() { gottp.assert(false, "boom") })`

	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: req},
		},
		Workflows: []collection.Workflow{{
			Name:  "ScriptFailure",
			Steps: []collection.WorkflowStep{{Request: "Scripted"}},
		}},
	}

	r := newWorkflowRunner(col)
	res, err := r.RunWorkflow(context.Background(), "ScriptFailure", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected workflow to be unsuccessful due to failing script test")
	}
	if res.Error != "" {
		t.Fatalf("expected no fatal workflow error for script assertion failure, got %q", res.Error)
	}
}

func TestBuildRequestMapAndListWorkflows(t *testing.T) {
	col := &collection.Collection{
		Name: "Workflow Test",
		Items: []collection.Item{
			{Request: collection.NewRequest("Get Users", "GET", "https://example.com/users")},
			{Request: collection.NewRequest("Create User", "POST", "https://example.com/users")},
		},
		Workflows: []collection.Workflow{
			{Name: "Smoke"},
			{Name: "Regression"},
		},
	}
	r := newWorkflowRunner(col)

	m := r.buildRequestMap()
	if len(m) != 2 {
		t.Fatalf("expected 2 requests in map, got %d", len(m))
	}
	if _, ok := m["get users"]; !ok {
		t.Fatal("expected lowercase key for 'Get Users'")
	}

	names := r.ListWorkflows()
	if len(names) != 2 || names[0] != "Smoke" || names[1] != "Regression" {
		t.Fatalf("unexpected workflow names: %v", names)
	}

	var emptyRunner Runner
	if got := emptyRunner.ListWorkflows(); got != nil {
		t.Fatalf("expected nil workflow list when collection is nil, got %v", got)
	}
}
