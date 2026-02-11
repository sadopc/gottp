package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
)

func TestGenerateBashCompletion(t *testing.T) {
	output := generateBashCompletion()

	if !strings.Contains(output, "_gottp") {
		t.Error("bash completion should contain _gottp function name")
	}
	if !strings.Contains(output, "complete -F _gottp gottp") {
		t.Error("bash completion should register the completion function")
	}
	if !strings.Contains(output, "commands=") {
		t.Error("bash completion should define commands list")
	}

	// Verify all subcommands are listed
	subcommands := []string{"run", "init", "validate", "fmt", "import", "export", "mock", "completion", "version", "help"}
	for _, cmd := range subcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("bash completion should contain subcommand %q", cmd)
		}
	}

	// Verify run flags are included
	runFlags := []string{"--env", "--request", "--folder", "--workflow", "--output", "--verbose", "--timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("bash completion should contain run flag %q", flag)
		}
	}

	// Verify output format values
	outputFormats := []string{"text", "json", "junit"}
	for _, fmt := range outputFormats {
		if !strings.Contains(output, fmt) {
			t.Errorf("bash completion should contain output format %q", fmt)
		}
	}

	// Verify export format values
	exportFormats := []string{"curl", "har", "postman", "insomnia"}
	for _, fmt := range exportFormats {
		if !strings.Contains(output, fmt) {
			t.Errorf("bash completion should contain export format %q", fmt)
		}
	}
}

func TestGenerateZshCompletion(t *testing.T) {
	output := generateZshCompletion()

	if !strings.Contains(output, "#compdef gottp") {
		t.Error("zsh completion should contain #compdef gottp directive")
	}
	if !strings.Contains(output, "_gottp") {
		t.Error("zsh completion should contain _gottp function")
	}
	if !strings.Contains(output, "_arguments") {
		t.Error("zsh completion should use _arguments for flag completion")
	}
	if !strings.Contains(output, "_describe") {
		t.Error("zsh completion should use _describe for command completion")
	}
	if !strings.Contains(output, `_files -g "*.gottp.yaml"`) {
		t.Error("zsh completion should complete .gottp.yaml files")
	}

	// Verify subcommands with descriptions
	subcommands := []string{"run:", "init:", "validate:", "fmt:", "import:", "export:", "completion:", "version:", "help:"}
	for _, cmd := range subcommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("zsh completion should contain subcommand description for %q", cmd)
		}
	}

	// Verify run flags
	runFlags := []string{"--env", "--request", "--folder", "--workflow", "--output", "--verbose", "--timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("zsh completion should contain run flag %q", flag)
		}
	}

	// Verify format value completions
	if !strings.Contains(output, "(text json junit)") {
		t.Error("zsh completion should provide output format values")
	}
	if !strings.Contains(output, "(curl postman insomnia openapi har)") {
		t.Error("zsh completion should provide import format values")
	}
	if !strings.Contains(output, "(curl har postman insomnia)") {
		t.Error("zsh completion should provide export format values")
	}
}

func TestGenerateFishCompletion(t *testing.T) {
	output := generateFishCompletion()

	if !strings.Contains(output, "complete -c gottp") {
		t.Error("fish completion should contain complete -c gottp commands")
	}
	if !strings.Contains(output, "__fish_use_subcommand") {
		t.Error("fish completion should use __fish_use_subcommand for top-level completions")
	}
	if !strings.Contains(output, "__fish_seen_subcommand_from") {
		t.Error("fish completion should use __fish_seen_subcommand_from for subcommand flags")
	}

	// Verify all subcommands are registered with descriptions
	subcommands := map[string]string{
		"run":        "Run API requests",
		"init":       "Create a new",
		"validate":   "Validate collection",
		"fmt":        "Format and normalize",
		"import":     "Import collection",
		"export":     "Export collection",
		"mock":       "Start a mock server",
		"completion": "Generate shell completion",
		"version":    "Print version",
		"help":       "Show help",
	}
	for cmd, desc := range subcommands {
		if !strings.Contains(output, "-a "+cmd) {
			t.Errorf("fish completion should register subcommand %q", cmd)
		}
		if !strings.Contains(output, desc) {
			t.Errorf("fish completion should have description containing %q for subcommand %q", desc, cmd)
		}
	}

	// Verify run flags
	runFlags := []string{"env", "request", "folder", "workflow", "output", "verbose", "timeout"}
	for _, flag := range runFlags {
		if !strings.Contains(output, "-l "+flag) {
			t.Errorf("fish completion should contain run long flag %q", flag)
		}
	}

	// Verify format completions
	if !strings.Contains(output, "'text json junit'") {
		t.Error("fish completion should provide output format values for run")
	}
	if !strings.Contains(output, "'curl har postman insomnia'") {
		t.Error("fish completion should provide export format values")
	}
	if !strings.Contains(output, "'curl postman insomnia openapi har'") {
		t.Error("fish completion should provide import format values")
	}
}

func TestGenerateBashCompletionShellFormat(t *testing.T) {
	output := generateBashCompletion()

	// Should be a valid shell script starting with a comment
	if !strings.HasPrefix(output, "#") {
		t.Error("bash completion should start with a comment")
	}

	// Should end with the complete command
	trimmed := strings.TrimSpace(output)
	if !strings.HasSuffix(trimmed, "complete -F _gottp gottp") {
		t.Error("bash completion should end with complete registration")
	}
}

func TestGenerateZshCompletionShellFormat(t *testing.T) {
	output := generateZshCompletion()

	// Must start with #compdef
	if !strings.HasPrefix(output, "#compdef gottp") {
		t.Error("zsh completion must start with #compdef gottp")
	}

	// Should end with calling the function
	trimmed := strings.TrimSpace(output)
	if !strings.HasSuffix(trimmed, `_gottp "$@"`) {
		t.Error("zsh completion should end with _gottp \"$@\" call")
	}
}

func TestGenerateFishCompletionShellFormat(t *testing.T) {
	output := generateFishCompletion()

	// Should start with a comment
	if !strings.HasPrefix(output, "#") {
		t.Error("fish completion should start with a comment")
	}

	// Every non-comment, non-empty line should start with "complete"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "complete ") {
			t.Errorf("fish completion non-comment line should start with 'complete': %q", line)
		}
	}
}

func TestCollectAllRequests_Recursive(t *testing.T) {
	req1 := collection.NewRequest("One", "GET", "https://example.com/one")
	req2 := collection.NewRequest("Two", "POST", "https://example.com/two")
	req3 := collection.NewRequest("Three", "DELETE", "https://example.com/three")

	items := []collection.Item{
		{Request: req1},
		{Folder: &collection.Folder{
			Name: "Nested",
			Items: []collection.Item{
				{Request: req2},
				{Folder: &collection.Folder{Name: "Deeper", Items: []collection.Item{{Request: req3}}}},
			},
		}},
	}

	got := collectAllRequests(items)
	if len(got) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(got))
	}
	if got[0].Name != "One" || got[1].Name != "Two" || got[2].Name != "Three" {
		t.Fatalf("unexpected request order/content: %q, %q, %q", got[0].Name, got[1].Name, got[2].Name)
	}
}

func TestCollectionRequestToProtocol(t *testing.T) {
	colReq := &collection.Request{
		Name:   "Create User",
		Method: "POST",
		URL:    "https://example.com/users",
		Headers: []collection.KVPair{
			{Key: "Content-Type", Value: "application/json", Enabled: true},
			{Key: "X-Disabled", Value: "ignored", Enabled: false},
		},
		Params: []collection.KVPair{
			{Key: "page", Value: "1", Enabled: true},
			{Key: "disabled", Value: "x", Enabled: false},
		},
		Body: &collection.Body{Type: "json", Content: `{"name":"alice"}`},
	}

	req := collectionRequestToProtocol(colReq)
	if req.Protocol != "http" {
		t.Fatalf("expected default protocol http, got %q", req.Protocol)
	}
	if req.Method != "POST" || req.URL != "https://example.com/users" {
		t.Fatalf("unexpected method/url: %s %s", req.Method, req.URL)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected enabled header in output, got headers=%v", req.Headers)
	}
	if _, ok := req.Headers["X-Disabled"]; ok {
		t.Fatalf("disabled header should be omitted, got headers=%v", req.Headers)
	}
	if req.Params["page"] != "1" {
		t.Fatalf("expected enabled query param in output, got params=%v", req.Params)
	}
	if _, ok := req.Params["disabled"]; ok {
		t.Fatalf("disabled query param should be omitted, got params=%v", req.Params)
	}
	if string(req.Body) != `{"name":"alice"}` {
		t.Fatalf("unexpected body: %q", string(req.Body))
	}
}

func TestExportFunctions_WriteOutput(t *testing.T) {
	colReq := collection.NewRequest("Health", "GET", "https://example.com/health")
	colReq.Headers = []collection.KVPair{{Key: "Accept", Value: "application/json", Enabled: true}}
	requests := []*collection.Request{colReq}

	col := &collection.Collection{
		Name:    "Export Test",
		Version: "1",
		Items:   []collection.Item{{Request: colReq}},
	}

	assertOutput := func(name string, writer func(*os.File), wantContains string) {
		t.Helper()
		path := filepath.Join(t.TempDir(), name)
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("os.Create failed: %v", err)
		}

		writer(f)
		if err := f.Close(); err != nil {
			t.Fatalf("close failed: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile failed: %v", err)
		}
		if len(data) == 0 {
			t.Fatalf("expected non-empty output for %s", name)
		}
		if wantContains != "" && !strings.Contains(string(data), wantContains) {
			t.Fatalf("expected output for %s to contain %q, got:\n%s", name, wantContains, string(data))
		}
	}

	assertOutput("curl.out", func(f *os.File) { exportAsCurl(f, requests) }, "curl")
	assertOutput("har.out", func(f *os.File) { exportAsHAR(f, requests) }, "\"log\"")
	assertOutput("postman.out", func(f *os.File) { exportAsPostman(f, col) }, "\"info\"")
	assertOutput("insomnia.out", func(f *os.File) { exportAsInsomnia(f, col) }, "\"_type\": \"export\"")
}

func TestCurlRequestToCollection(t *testing.T) {
	input := &protocol.Request{
		Method: "POST",
		URL:    "https://example.com/tokens",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Params: map[string]string{
			"verbose": "1",
		},
		Body: []byte(`{"grant_type":"client_credentials"}`),
	}

	col := curlRequestToCollection(input)
	if col == nil {
		t.Fatal("curlRequestToCollection returned nil")
	}
	if col.Name != "cURL Import" {
		t.Fatalf("unexpected collection name: %q", col.Name)
	}
	if len(col.Items) != 1 || col.Items[0].Request == nil {
		t.Fatalf("expected one request item, got %+v", col.Items)
	}
	req := col.Items[0].Request
	if req.Method != "POST" || req.URL != "https://example.com/tokens" {
		t.Fatalf("unexpected request method/url: %s %s", req.Method, req.URL)
	}
	if req.Body == nil || req.Body.Content == "" {
		t.Fatal("expected body content to be copied")
	}
}

func TestReadStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	defer r.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	input := "line-1\nline-2\n"
	if _, err := io.WriteString(w, input); err != nil {
		t.Fatalf("pipe write failed: %v", err)
	}
	_ = w.Close()

	data, err := readStdin()
	if err != nil {
		t.Fatalf("readStdin failed: %v", err)
	}
	if string(data) != input {
		t.Fatalf("readStdin returned %q, want %q", string(data), input)
	}
}

func TestValidateHelpers(t *testing.T) {
	items := []collection.Item{
		{Request: &collection.Request{ID: "id-1", Name: "A", URL: "https://example.com/a"}},
		{Folder: &collection.Folder{Items: []collection.Item{
			{Request: &collection.Request{ID: "id-1", Name: "B", URL: ""}},
		}}},
	}

	if got := countRequests(items); got != 2 {
		t.Fatalf("countRequests = %d, want 2", got)
	}

	dups := checkDuplicateIDs(items, map[string]string{})
	if len(dups) != 1 || !strings.Contains(dups[0], "id-1") {
		t.Fatalf("unexpected duplicate IDs output: %v", dups)
	}

	empty := checkEmptyURLs(items)
	if len(empty) != 1 || empty[0] != "B" {
		t.Fatalf("unexpected empty URL list: %v", empty)
	}
}

func TestValidateEnvironmentAndFile(t *testing.T) {
	dir := t.TempDir()

	envPath := filepath.Join(dir, "environments.yaml")
	validEnv := `environments:
  - name: Development
    variables:
      base_url:
        value: "http://localhost:8080"
  - name: Production
    variables:
      base_url:
        value: "https://api.example.com"
`
	if err := os.WriteFile(envPath, []byte(validEnv), 0644); err != nil {
		t.Fatalf("failed to write environments.yaml: %v", err)
	}
	if err := validateEnvironment(envPath); err != nil {
		t.Fatalf("validateEnvironment should pass for valid file: %v", err)
	}

	dupEnvPath := filepath.Join(dir, "dup-environments.yaml")
	dupEnv := `environments:
  - name: Duplicate
    variables: {}
  - name: Duplicate
    variables: {}
`
	if err := os.WriteFile(dupEnvPath, []byte(dupEnv), 0644); err != nil {
		t.Fatalf("failed to write duplicate env file: %v", err)
	}
	if err := validateEnvironment(dupEnvPath); err == nil {
		t.Fatal("expected duplicate env validation error")
	}

	colPath := filepath.Join(dir, "api.gottp.yaml")
	col := &collection.Collection{
		Name:    "API",
		Version: "1",
		Items: []collection.Item{
			{Request: collection.NewRequest("Health", "GET", "https://example.com/health")},
		},
	}
	if err := collection.SaveToFile(col, colPath); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}
	if err := validateFile(colPath); err != nil {
		t.Fatalf("validateFile should pass for valid collection: %v", err)
	}

	badColPath := filepath.Join(dir, "bad.gottp.yaml")
	bad := &collection.Collection{
		Name:    "Bad",
		Version: "1",
		Items: []collection.Item{
			{Request: collection.NewRequest("Empty URL", "GET", "")},
		},
	}
	if err := collection.SaveToFile(bad, badColPath); err != nil {
		t.Fatalf("failed to save bad collection: %v", err)
	}
	if err := validateFile(badColPath); err == nil {
		t.Fatal("expected validateFile warning/error for empty URL")
	}
}

func TestFormatFile_CheckAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.gottp.yaml")

	unformatted := `name: Minimal
items:
  - request:
      name: Health
      method: GET
      url: https://example.com/health
`
	if err := os.WriteFile(path, []byte(unformatted), 0644); err != nil {
		t.Fatalf("failed to write unformatted file: %v", err)
	}

	hasUnformatted := false
	if err := formatFile(path, false, true, &hasUnformatted); err != nil {
		t.Fatalf("formatFile(check) failed: %v", err)
	}
	if !hasUnformatted {
		t.Fatal("expected check mode to detect unformatted file")
	}

	hasUnformatted = false
	if err := formatFile(path, true, false, &hasUnformatted); err != nil {
		t.Fatalf("formatFile(write) failed: %v", err)
	}

	if err := formatFile(path, false, true, &hasUnformatted); err != nil {
		t.Fatalf("formatFile(check after write) failed: %v", err)
	}
	if hasUnformatted {
		t.Fatal("expected file to be formatted after write mode")
	}
}

func TestPrintHelp_WritesExpectedSections(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	printHelp()
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stderr: %v", err)
	}
	text := string(out)

	if !strings.Contains(text, "Usage:") || !strings.Contains(text, "Commands:") {
		t.Fatalf("help output missing expected sections:\n%s", text)
	}
	if !strings.Contains(text, "run       Run API requests") || !strings.Contains(text, "completion  Generate shell completion") {
		t.Fatalf("help output missing expected command descriptions:\n%s", text)
	}
}
