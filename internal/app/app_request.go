package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	oauth2auth "github.com/serdar/gottp/internal/auth/oauth2"
	"github.com/serdar/gottp/internal/core/environment"
	"github.com/serdar/gottp/internal/core/history"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/protocol/graphql"
	"github.com/serdar/gottp/internal/scripting"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/panels/response"
)

func (a App) sendRequest() (tea.Model, tea.Cmd) {
	req := a.editor.BuildRequest()
	if req.URL == "" {
		a.statusBar.SetMessage("URL is required")
		return a, nil
	}

	// Set response mode based on protocol
	a.response.SetMode(a.editor.Protocol())

	// Resolve environment variables
	envVars := a.store.EnvVars
	var colVars map[string]string
	if a.store.Collection != nil {
		colVars = a.store.Collection.Variables
	}
	if envVars == nil {
		envVars = map[string]string{}
	}
	if colVars == nil {
		colVars = map[string]string{}
	}
	if len(envVars) > 0 || len(colVars) > 0 {
		req.URL = environment.Resolve(req.URL, envVars, colVars)
		for k, v := range req.Headers {
			req.Headers[k] = environment.Resolve(v, envVars, colVars)
		}
		for k, v := range req.Params {
			req.Params[k] = environment.Resolve(v, envVars, colVars)
		}
		if len(req.Body) > 0 {
			req.Body = []byte(environment.Resolve(string(req.Body), envVars, colVars))
		}
		if req.Auth != nil {
			req.Auth.Username = environment.Resolve(req.Auth.Username, envVars, colVars)
			req.Auth.Password = environment.Resolve(req.Auth.Password, envVars, colVars)
			req.Auth.Token = environment.Resolve(req.Auth.Token, envVars, colVars)
			req.Auth.APIKey = environment.Resolve(req.Auth.APIKey, envVars, colVars)
			req.Auth.APIValue = environment.Resolve(req.Auth.APIValue, envVars, colVars)
		}
	}

	// Run pre-request script
	if req.PreScript != "" && a.scriptEngine != nil {
		scriptReq := &scripting.ScriptRequest{
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Params:  req.Params,
			Body:    string(req.Body),
		}
		result := a.scriptEngine.RunPreScript(req.PreScript, scriptReq, envVars)
		if result.Err != nil {
			a.response.SetScriptResults(result.Logs, convertTestResults(result.TestResults), result.Err.Error())
			cmd := a.toast.Show("Pre-script error: "+result.Err.Error(), true, 3*time.Second)
			return a, cmd
		}
		// Apply mutations from pre-script
		req.Method = scriptReq.Method
		req.URL = scriptReq.URL
		req.Headers = scriptReq.Headers
		req.Params = scriptReq.Params
		req.Body = []byte(scriptReq.Body)
		// Apply env changes
		for k, v := range result.EnvChanges {
			a.store.EnvVars[k] = v
		}
	}

	// Handle OAuth2: check for valid token or initiate flow
	if req.Auth != nil && req.Auth.Type == "oauth2" && req.Auth.OAuth2 != nil {
		oauth := req.Auth.OAuth2
		if oauth.AccessToken == "" || (oauth.TokenExpiry != (time.Time{}) && time.Now().After(oauth.TokenExpiry)) {
			return a.initiateOAuth2(req)
		}
	}

	a.response.SetLoading(true)

	timeout := a.cfg.DefaultTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	registry := a.protocols
	postScript := req.PostScript
	scriptEngine := a.scriptEngine
	cmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, err := registry.Execute(ctx, req)
		if err != nil {
			return msgs.RequestSentMsg{Err: err}
		}

		sentMsg := msgs.RequestSentMsg{
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			Headers:     resp.Headers,
			Body:        resp.Body,
			ContentType: resp.ContentType,
			Duration:    resp.Duration,
			Size:        resp.Size,
		}

		// Run post-request script if present
		if postScript != "" && scriptEngine != nil {
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
			result := scriptEngine.RunPostScript(postScript, scriptReq, scriptResp, envVars)
			sentMsg.ScriptResult = &msgs.ScriptResultMsg{
				Logs:        result.Logs,
				TestResults: convertScriptTestResults(result.TestResults),
				EnvChanges:  result.EnvChanges,
			}
			if result.Err != nil {
				errStr := result.Err.Error()
				sentMsg.ScriptErr = &errStr
			}
		}

		return sentMsg
	}

	return a, tea.Batch(cmd, a.response.Init())
}

func (a App) initiateOAuth2(req *protocol.Request) (tea.Model, tea.Cmd) {
	oauth := req.Auth.OAuth2
	a.response.SetLoading(true)

	switch oauth.GrantType {
	case "client_credentials":
		cfg := oauth2auth.OAuth2Config{
			TokenURL:     oauth.TokenURL,
			ClientID:     oauth.ClientID,
			ClientSecret: oauth.ClientSecret,
			Scope:        oauth.Scope,
		}
		cmd := func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			token, err := oauth2auth.ClientCredentials(ctx, cfg)
			if err != nil {
				return msgs.OAuth2TokenMsg{Err: err}
			}
			return msgs.OAuth2TokenMsg{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				ExpiresIn:    token.ExpiresIn,
			}
		}
		return a, cmd

	case "password":
		cfg := oauth2auth.OAuth2Config{
			TokenURL:     oauth.TokenURL,
			ClientID:     oauth.ClientID,
			ClientSecret: oauth.ClientSecret,
			Username:     oauth.Username,
			Password:     oauth.Password,
			Scope:        oauth.Scope,
		}
		cmd := func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			token, err := oauth2auth.PasswordGrant(ctx, cfg)
			if err != nil {
				return msgs.OAuth2TokenMsg{Err: err}
			}
			return msgs.OAuth2TokenMsg{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				ExpiresIn:    token.ExpiresIn,
			}
		}
		return a, cmd

	case "authorization_code":
		a.response.SetLoading(false)
		cmd := a.toast.Show("Auth code flow: use browser to authorize", false, 5*time.Second)
		return a, cmd
	}

	a.response.SetLoading(false)
	cmd := a.toast.Show("Unknown OAuth2 grant type", true, 3*time.Second)
	return a, cmd
}

func (a App) handleOAuth2Token(msg msgs.OAuth2TokenMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.response.SetLoading(false)
		cmd := a.toast.Show("OAuth2 error: "+msg.Err.Error(), true, 5*time.Second)
		return a, cmd
	}

	// Update the auth form with the acquired token and retry
	authCfg := a.editor.BuildAuth()
	if authCfg != nil && authCfg.OAuth2 != nil {
		authCfg.OAuth2.AccessToken = msg.AccessToken
		authCfg.OAuth2.RefreshToken = msg.RefreshToken
		if msg.ExpiresIn > 0 {
			authCfg.OAuth2.TokenExpiry = time.Now().Add(time.Duration(msg.ExpiresIn) * time.Second)
		}
	}

	cmd := a.toast.Show("OAuth2 token acquired", false, 2*time.Second)
	return a, cmd
}

func convertTestResults(results []scripting.TestResult) []response.ScriptTestResult {
	out := make([]response.ScriptTestResult, len(results))
	for i, r := range results {
		out[i] = response.ScriptTestResult{Name: r.Name, Passed: r.Passed, Error: r.Error}
	}
	return out
}

func convertScriptTestResults(results []scripting.TestResult) []msgs.ScriptTestResult {
	out := make([]msgs.ScriptTestResult, len(results))
	for i, r := range results {
		out[i] = msgs.ScriptTestResult{Name: r.Name, Passed: r.Passed, Error: r.Error}
	}
	return out
}

func (a App) handleRequestSent(msg msgs.RequestSentMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.response.SetLoading(false)
		a.statusBar.SetMessage("Error: " + msg.Err.Error())
		cmd := a.toast.Show("Request failed: "+msg.Err.Error(), true, 5*time.Second)
		return a, cmd
	}

	resp := &protocol.Response{
		StatusCode:  msg.StatusCode,
		Status:      msg.Status,
		Headers:     msg.Headers,
		Body:        msg.Body,
		ContentType: msg.ContentType,
		Duration:    msg.Duration,
		Size:        msg.Size,
	}

	a.response.SetResponse(resp)
	a.statusBar.SetStatus(msg.StatusCode, msg.Duration, msg.Size, msg.ContentType)

	// Process post-script results if present
	if msg.ScriptResult != nil {
		var testResults []response.ScriptTestResult
		for _, tr := range msg.ScriptResult.TestResults {
			testResults = append(testResults, response.ScriptTestResult{
				Name: tr.Name, Passed: tr.Passed, Error: tr.Error,
			})
		}
		errMsg := ""
		if msg.ScriptErr != nil {
			errMsg = *msg.ScriptErr
		}
		a.response.SetScriptResults(msg.ScriptResult.Logs, testResults, errMsg)
		// Apply env changes from post-script
		for k, v := range msg.ScriptResult.EnvChanges {
			a.store.EnvVars[k] = v
		}
	}

	// Save to history
	if a.history != nil {
		req := a.editor.BuildRequest()
		headersJSON, _ := json.Marshal(req.Headers)
		a.history.Add(history.Entry{
			Method:       req.Method,
			URL:          req.URL,
			StatusCode:   msg.StatusCode,
			Duration:     msg.Duration,
			Size:         msg.Size,
			RequestBody:  string(req.Body),
			ResponseBody: string(msg.Body),
			Headers:      string(headersJSON),
			Timestamp:    time.Now(),
		})
		a.loadHistory()
	}

	return a, nil
}

func (a App) handleIntrospect() (tea.Model, tea.Cmd) {
	req := a.editor.BuildRequest()
	if req.URL == "" {
		cmd := a.toast.Show("URL is required for introspection", true, 2*time.Second)
		return a, cmd
	}

	url := req.URL
	headers := req.Headers
	cmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		schema, err := graphql.RunIntrospection(ctx, url, headers)
		if err != nil {
			return msgs.IntrospectionResultMsg{Err: err}
		}
		types := make([]msgs.SchemaType, len(schema.Types))
		for i, t := range schema.Types {
			fields := make([]msgs.SchemaField, len(t.Fields))
			for j, f := range t.Fields {
				fields[j] = msgs.SchemaField{Name: f.Name, Type: f.Type}
			}
			types[i] = msgs.SchemaType{Name: t.Name, Fields: fields}
		}
		return msgs.IntrospectionResultMsg{Types: types}
	}

	toastCmd := a.toast.Show("Running introspection...", false, 2*time.Second)
	return a, tea.Batch(cmd, toastCmd)
}

func (a App) handleIntrospectionResult(msg msgs.IntrospectionResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toast.Show("Introspection failed: "+msg.Err.Error(), true, 5*time.Second)
		return a, cmd
	}
	cmd := a.toast.Show("Introspection complete: "+fmt.Sprintf("%d types", len(msg.Types)), false, 2*time.Second)
	return a, cmd
}

func (a App) handleScriptResult(msg msgs.ScriptResultMsg) (tea.Model, tea.Cmd) {
	var testResults []response.ScriptTestResult
	for _, tr := range msg.TestResults {
		testResults = append(testResults, response.ScriptTestResult{
			Name:   tr.Name,
			Passed: tr.Passed,
			Error:  tr.Error,
		})
	}
	errMsg := ""
	if msg.Err != nil {
		errMsg = msg.Err.Error()
	}
	a.response.SetScriptResults(msg.Logs, testResults, errMsg)

	// Apply env changes
	for k, v := range msg.EnvChanges {
		a.store.EnvVars[k] = v
	}

	return a, nil
}

func (a App) handleGRPCReflect() (tea.Model, tea.Cmd) {
	// gRPC reflection is a placeholder until the gRPC client is implemented
	cmd := a.toast.Show("gRPC reflection not yet implemented", true, 2*time.Second)
	return a, cmd
}

func (a App) handleGRPCReflectionResult(msg msgs.GRPCReflectionResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toast.Show("gRPC reflection failed: "+msg.Err.Error(), true, 5*time.Second)
		return a, cmd
	}
	// Pass services to gRPC form
	a.editor.GRPCFormRef().SetServices(msg.Services)
	cmd := a.toast.Show("gRPC reflection complete", false, 2*time.Second)
	return a, cmd
}
