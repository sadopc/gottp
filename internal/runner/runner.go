package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/core/environment"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/protocol/graphql"
	grpcclient "github.com/serdar/gottp/internal/protocol/grpc"
	httpclient "github.com/serdar/gottp/internal/protocol/http"
	wsclient "github.com/serdar/gottp/internal/protocol/websocket"
	"github.com/serdar/gottp/internal/scripting"
)

// Runner executes requests headlessly (no TUI).
type Runner struct {
	collection   *collection.Collection
	envFile      *environment.EnvironmentFile
	registry     *protocol.Registry
	scriptEngine *scripting.Engine
	envVars      map[string]string
	colVars      map[string]string
	timeout      time.Duration
}

// Config holds runner configuration.
type Config struct {
	CollectionPath string
	Environment    string
	RequestName    string // run single request by name
	FolderName     string // run all requests in folder
	WorkflowName   string // run a named workflow
	OutputFormat   string // "text", "json", "junit"
	Verbose        bool
	Timeout        time.Duration
}

// Result holds execution results for a single request.
type Result struct {
	Name        string              `json:"name"`
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	StatusCode  int                 `json:"status_code"`
	Status      string              `json:"status"`
	Duration    time.Duration       `json:"duration"`
	Size        int64               `json:"size"`
	Error       error               `json:"-"`
	ErrorString string              `json:"error,omitempty"`
	ScriptLogs  []string            `json:"script_logs,omitempty"`
	TestResults []TestResult        `json:"test_results,omitempty"`
	TestsPassed bool                `json:"tests_passed"`
	Body        []byte              `json:"-"`
	BodyString  string              `json:"body,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
}

// TestResult holds the result of a script test assertion.
type TestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// New creates a runner from config.
func New(cfg Config) (*Runner, error) {
	if cfg.CollectionPath == "" {
		return nil, fmt.Errorf("collection path is required")
	}

	col, err := collection.LoadFromFile(cfg.CollectionPath)
	if err != nil {
		return nil, fmt.Errorf("loading collection: %w", err)
	}

	// Load environments
	dir := filepath.Dir(cfg.CollectionPath)
	envFile, err := environment.LoadEnvironments(filepath.Join(dir, "environments.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading environments: %w", err)
	}

	// Resolve active environment
	envVars := map[string]string{}
	if cfg.Environment != "" {
		envVars = envFile.GetVariables(cfg.Environment)
		if len(envVars) == 0 {
			// Check if the environment name exists at all
			found := false
			for _, env := range envFile.Environments {
				if env.Name == cfg.Environment {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("environment %q not found (available: %s)",
					cfg.Environment, strings.Join(envFile.Names(), ", "))
			}
		}
	} else if len(envFile.Environments) > 0 {
		// Auto-select first environment
		envVars = envFile.GetVariables(envFile.Environments[0].Name)
	}

	colVars := map[string]string{}
	if col.Variables != nil {
		colVars = col.Variables
	}

	// Set up protocol registry
	registry := protocol.NewRegistry()
	registry.Register(httpclient.New())
	registry.Register(graphql.New())
	registry.Register(wsclient.New())
	registry.Register(grpcclient.New())

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Runner{
		collection:   col,
		envFile:      envFile,
		registry:     registry,
		scriptEngine: scripting.NewEngine(5 * time.Second),
		envVars:      envVars,
		colVars:      colVars,
		timeout:      timeout,
	}, nil
}

// Run executes the configured requests and returns results.
func (r *Runner) Run(ctx context.Context, cfg Config) ([]Result, error) {
	requests := r.collectRequests(cfg)
	if len(requests) == 0 {
		if cfg.RequestName != "" {
			return nil, fmt.Errorf("request %q not found in collection", cfg.RequestName)
		}
		if cfg.FolderName != "" {
			return nil, fmt.Errorf("folder %q not found in collection", cfg.FolderName)
		}
		return nil, fmt.Errorf("no requests found in collection")
	}

	results := make([]Result, 0, len(requests))
	for _, req := range requests {
		result := r.executeRequest(ctx, req, cfg.Verbose)
		results = append(results, result)
	}
	return results, nil
}

// collectRequests gathers the requests to run based on config filters.
func (r *Runner) collectRequests(cfg Config) []*collection.Request {
	var requests []*collection.Request

	if cfg.RequestName != "" {
		// Find single request by name (case-insensitive)
		r.walkItems(r.collection.Items, "", func(req *collection.Request, folder string) {
			if strings.EqualFold(req.Name, cfg.RequestName) {
				requests = append(requests, req)
			}
		})
		return requests
	}

	if cfg.FolderName != "" {
		// Find all requests in a folder (case-insensitive)
		r.walkItems(r.collection.Items, "", func(req *collection.Request, folder string) {
			if strings.EqualFold(folder, cfg.FolderName) {
				requests = append(requests, req)
			}
		})
		return requests
	}

	// All requests
	r.walkItems(r.collection.Items, "", func(req *collection.Request, folder string) {
		requests = append(requests, req)
	})
	return requests
}

// walkItems walks through collection items, calling fn for each request with its parent folder name.
func (r *Runner) walkItems(items []collection.Item, parentFolder string, fn func(*collection.Request, string)) {
	for i := range items {
		if items[i].Folder != nil {
			r.walkItems(items[i].Folder.Items, items[i].Folder.Name, fn)
		}
		if items[i].Request != nil {
			fn(items[i].Request, parentFolder)
		}
	}
}

// executeRequest runs a single request through the full lifecycle.
func (r *Runner) executeRequest(ctx context.Context, colReq *collection.Request, verbose bool) Result {
	result := Result{
		Name:   colReq.Name,
		Method: colReq.Method,
		URL:    colReq.URL,
	}

	// Build protocol request from collection request
	req := buildProtocolRequest(colReq)

	// Resolve environment variables
	r.resolveVars(req)
	result.URL = req.URL // update with resolved URL

	// Run pre-request script
	if colReq.PreScript != "" {
		scriptReq := &scripting.ScriptRequest{
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Params:  req.Params,
			Body:    string(req.Body),
		}
		scriptResult := r.scriptEngine.RunPreScript(colReq.PreScript, scriptReq, r.envVars)
		result.ScriptLogs = append(result.ScriptLogs, scriptResult.Logs...)

		if scriptResult.Err != nil {
			result.Error = fmt.Errorf("pre-script error: %w", scriptResult.Err)
			result.ErrorString = result.Error.Error()
			return result
		}

		// Apply mutations from pre-script
		req.Method = scriptReq.Method
		req.URL = scriptReq.URL
		req.Headers = scriptReq.Headers
		req.Params = scriptReq.Params
		req.Body = []byte(scriptReq.Body)

		// Apply env changes
		for k, v := range scriptResult.EnvChanges {
			r.envVars[k] = v
		}
	}

	// Execute request
	reqCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	resp, err := r.registry.Execute(reqCtx, req)
	if err != nil {
		result.Error = err
		result.ErrorString = err.Error()
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.Duration = resp.Duration
	result.Size = resp.Size
	if verbose {
		result.Body = resp.Body
		result.BodyString = string(resp.Body)
		headers := make(map[string][]string)
		for k, v := range resp.Headers {
			headers[k] = v
		}
		result.Headers = headers
	}

	// Run post-request script
	if colReq.PostScript != "" {
		scriptReq := &scripting.ScriptRequest{
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Params:  req.Params,
			Body:    string(req.Body),
		}
		respHeaders := make(map[string]string)
		for k := range resp.Headers {
			respHeaders[k] = resp.Headers.Get(k)
		}
		scriptResp := &scripting.ScriptResponse{
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			Body:        string(resp.Body),
			Headers:     respHeaders,
			Duration:    float64(resp.Duration.Milliseconds()),
			Size:        resp.Size,
			ContentType: resp.ContentType,
		}
		scriptResult := r.scriptEngine.RunPostScript(colReq.PostScript, scriptReq, scriptResp, r.envVars)
		result.ScriptLogs = append(result.ScriptLogs, scriptResult.Logs...)

		if scriptResult.Err != nil {
			result.ScriptLogs = append(result.ScriptLogs, "Post-script error: "+scriptResult.Err.Error())
		}

		// Collect test results
		result.TestsPassed = true
		for _, tr := range scriptResult.TestResults {
			result.TestResults = append(result.TestResults, TestResult{
				Name:   tr.Name,
				Passed: tr.Passed,
				Error:  tr.Error,
			})
			if !tr.Passed {
				result.TestsPassed = false
			}
		}
		// If no tests were run, tests are considered passed
		if len(result.TestResults) == 0 {
			result.TestsPassed = true
		}

		// Apply env changes
		for k, v := range scriptResult.EnvChanges {
			r.envVars[k] = v
		}
	} else {
		result.TestsPassed = true
	}

	return result
}

// buildProtocolRequest converts a collection.Request to a protocol.Request.
func buildProtocolRequest(colReq *collection.Request) *protocol.Request {
	req := &protocol.Request{
		Protocol:   colReq.Protocol,
		Method:     colReq.Method,
		URL:        colReq.URL,
		Headers:    make(map[string]string),
		Params:     make(map[string]string),
		PreScript:  colReq.PreScript,
		PostScript: colReq.PostScript,
	}

	if req.Protocol == "" {
		req.Protocol = "http"
	}

	// Params
	for _, p := range colReq.Params {
		if p.Enabled && p.Key != "" {
			req.Params[p.Key] = p.Value
		}
	}

	// Headers
	for _, h := range colReq.Headers {
		if h.Enabled && h.Key != "" {
			req.Headers[h.Key] = h.Value
		}
	}

	// Body
	if colReq.Body != nil && colReq.Body.Content != "" {
		req.Body = []byte(colReq.Body.Content)
	}

	// Auth
	if colReq.Auth != nil {
		req.Auth = buildAuthConfig(colReq.Auth)
	}

	// GraphQL
	if colReq.GraphQL != nil {
		req.GraphQLQuery = colReq.GraphQL.Query
		req.GraphQLVariables = colReq.GraphQL.Variables
	}

	// gRPC
	if colReq.GRPC != nil {
		req.GRPCService = colReq.GRPC.Service
		req.GRPCMethod = colReq.GRPC.Method
		req.Metadata = make(map[string]string)
		for _, m := range colReq.GRPC.Metadata {
			if m.Enabled && m.Key != "" {
				req.Metadata[m.Key] = m.Value
			}
		}
	}

	return req
}

// buildAuthConfig converts collection auth to protocol auth config.
func buildAuthConfig(auth *collection.Auth) *protocol.AuthConfig {
	if auth == nil || auth.Type == "" || auth.Type == "none" {
		return nil
	}
	cfg := &protocol.AuthConfig{Type: auth.Type}
	switch auth.Type {
	case "basic":
		if auth.Basic != nil {
			cfg.Username = auth.Basic.Username
			cfg.Password = auth.Basic.Password
		}
	case "bearer":
		if auth.Bearer != nil {
			cfg.Token = auth.Bearer.Token
		}
	case "apikey":
		if auth.APIKey != nil {
			cfg.APIKey = auth.APIKey.Key
			cfg.APIValue = auth.APIKey.Value
			cfg.APIIn = auth.APIKey.In
		}
	case "oauth2":
		if auth.OAuth2 != nil {
			cfg.OAuth2 = &protocol.OAuth2AuthConfig{
				GrantType:    auth.OAuth2.GrantType,
				AuthURL:      auth.OAuth2.AuthURL,
				TokenURL:     auth.OAuth2.TokenURL,
				ClientID:     auth.OAuth2.ClientID,
				ClientSecret: auth.OAuth2.ClientSecret,
				Scope:        auth.OAuth2.Scope,
				Username:     auth.OAuth2.Username,
				Password:     auth.OAuth2.Password,
				UsePKCE:      auth.OAuth2.UsePKCE,
			}
		}
	case "awsv4":
		if auth.AWSAuth != nil {
			cfg.AWSAuth = &protocol.AWSAuthConfig{
				AccessKeyID:     auth.AWSAuth.AccessKeyID,
				SecretAccessKey: auth.AWSAuth.SecretAccessKey,
				SessionToken:    auth.AWSAuth.SessionToken,
				Region:          auth.AWSAuth.Region,
				Service:         auth.AWSAuth.Service,
			}
		}
	case "digest":
		if auth.Digest != nil {
			cfg.DigestUsername = auth.Digest.Username
			cfg.DigestPassword = auth.Digest.Password
		}
	}
	return cfg
}

// resolveVars replaces {{variable}} placeholders in all request fields.
func (r *Runner) resolveVars(req *protocol.Request) {
	if len(r.envVars) == 0 && len(r.colVars) == 0 {
		return
	}

	req.URL = environment.Resolve(req.URL, r.envVars, r.colVars)

	for k, v := range req.Headers {
		req.Headers[k] = environment.Resolve(v, r.envVars, r.colVars)
	}
	for k, v := range req.Params {
		req.Params[k] = environment.Resolve(v, r.envVars, r.colVars)
	}
	if len(req.Body) > 0 {
		req.Body = []byte(environment.Resolve(string(req.Body), r.envVars, r.colVars))
	}
	if req.Auth != nil {
		req.Auth.Username = environment.Resolve(req.Auth.Username, r.envVars, r.colVars)
		req.Auth.Password = environment.Resolve(req.Auth.Password, r.envVars, r.colVars)
		req.Auth.Token = environment.Resolve(req.Auth.Token, r.envVars, r.colVars)
		req.Auth.APIKey = environment.Resolve(req.Auth.APIKey, r.envVars, r.colVars)
		req.Auth.APIValue = environment.Resolve(req.Auth.APIValue, r.envVars, r.colVars)
	}

	// GraphQL
	if req.GraphQLQuery != "" {
		req.GraphQLQuery = environment.Resolve(req.GraphQLQuery, r.envVars, r.colVars)
	}
	if req.GraphQLVariables != "" {
		req.GraphQLVariables = environment.Resolve(req.GraphQLVariables, r.envVars, r.colVars)
	}
}

// ExitCode returns the appropriate exit code based on results.
// 0 = all succeeded, 1 = test failures, 2 = request errors.
func ExitCode(results []Result) int {
	hasErrors := false
	hasTestFailures := false
	for _, r := range results {
		if r.Error != nil {
			hasErrors = true
		}
		if !r.TestsPassed {
			hasTestFailures = true
		}
	}
	if hasErrors {
		return 2
	}
	if hasTestFailures {
		return 1
	}
	return 0
}
