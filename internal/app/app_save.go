package app

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/core/environment"
	"github.com/serdar/gottp/internal/export"
	"github.com/serdar/gottp/internal/export/codegen"
	curlimport "github.com/serdar/gottp/internal/import/curl"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/templates"
	"github.com/serdar/gottp/internal/ui/msgs"
)

func (a App) saveCollection() (tea.Model, tea.Cmd) {
	if a.store.Collection == nil || a.store.CollectionPath == "" {
		a.statusBar.SetMessage("No collection to save")
		return a, nil
	}

	// Sync form state back to the active request before saving
	req := a.store.ActiveRequest()
	if req != nil {
		built := a.editor.BuildRequest()
		req.Method = built.Method
		req.URL = built.URL

		// Sync params
		formParams := a.editor.GetParams()
		req.Params = make([]collection.KVPair, len(formParams))
		for i, p := range formParams {
			req.Params[i] = collection.KVPair{Key: p.Key, Value: p.Value, Enabled: p.Enabled}
		}

		// Sync headers
		formHeaders := a.editor.GetHeaders()
		req.Headers = make([]collection.KVPair, len(formHeaders))
		for i, h := range formHeaders {
			req.Headers[i] = collection.KVPair{Key: h.Key, Value: h.Value, Enabled: h.Enabled}
		}

		// Sync body
		bodyContent := a.editor.GetBodyContent()
		if bodyContent != "" {
			if req.Body == nil {
				req.Body = &collection.Body{Type: "json"}
			}
			req.Body.Content = bodyContent
		} else {
			req.Body = nil
		}

		// Sync auth
		authConfig := a.editor.BuildAuth()
		if authConfig != nil && authConfig.Type != "none" {
			req.Auth = authConfigToCollection(authConfig)
		} else {
			req.Auth = nil
		}
	}

	err := collection.SaveToFile(a.store.Collection, a.store.CollectionPath)
	if err != nil {
		cmd := a.toast.Show("Save failed: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}
	cmd := a.toast.Show("Collection saved", false, 2*time.Second)
	return a, cmd
}

func authConfigToCollection(auth *protocol.AuthConfig) *collection.Auth {
	if auth == nil {
		return nil
	}
	ca := &collection.Auth{Type: auth.Type}
	switch auth.Type {
	case "basic":
		ca.Basic = &collection.BasicAuth{Username: auth.Username, Password: auth.Password}
	case "bearer":
		ca.Bearer = &collection.BearerAuth{Token: auth.Token}
	case "apikey":
		ca.APIKey = &collection.APIKeyAuth{Key: auth.APIKey, Value: auth.APIValue, In: auth.APIIn}
	case "oauth2":
		if auth.OAuth2 != nil {
			ca.OAuth2 = &collection.OAuth2Auth{
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
			ca.AWSAuth = &collection.AWSAuth{
				AccessKeyID:     auth.AWSAuth.AccessKeyID,
				SecretAccessKey: auth.AWSAuth.SecretAccessKey,
				SessionToken:    auth.AWSAuth.SessionToken,
				Region:          auth.AWSAuth.Region,
				Service:         auth.AWSAuth.Service,
			}
		}
	case "digest":
		ca.Digest = &collection.DigestAuth{
			Username: auth.DigestUsername,
			Password: auth.DigestPassword,
		}
	}
	return ca
}

func (a App) copyAsCurl() (tea.Model, tea.Cmd) {
	req := a.editor.BuildRequest()
	if req.URL == "" {
		cmd := a.toast.Show("No URL to copy", true, 2*time.Second)
		return a, cmd
	}

	// Resolve env vars before export
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

	curlCmd := export.AsCurl(req)
	if err := clipboard.WriteAll(curlCmd); err != nil {
		cmd := a.toast.Show("Clipboard error: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}
	cmd := a.toast.Show("Copied as cURL", false, 2*time.Second)
	return a, cmd
}

func (a App) handleGenerateCode(msg msgs.GenerateCodeMsg) (tea.Model, tea.Cmd) {
	req := a.editor.BuildRequest()
	if req.URL == "" {
		cmd := a.toast.Show("No URL to generate code for", true, 2*time.Second)
		return a, cmd
	}

	// Resolve env vars before generating
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

	code, err := codegen.Generate(req, codegen.Language(msg.Language))
	if err != nil {
		cmd := a.toast.Show("Code generation failed: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}

	if err := clipboard.WriteAll(code); err != nil {
		cmd := a.toast.Show("Clipboard error: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}

	cmd := a.toast.Show("Copied "+msg.Language+" code", false, 2*time.Second)
	return a, cmd
}

func (a App) handleInsertTemplate(msg msgs.InsertTemplateMsg) (tea.Model, tea.Cmd) {
	tmpl := templates.ByName(msg.TemplateName)
	if tmpl == nil {
		cmd := a.toast.Show("Template not found: "+msg.TemplateName, true, 2*time.Second)
		return a, cmd
	}

	req := tmpl.Request
	a.store.OpenRequest(req)
	a.syncTabs()
	a.editor.LoadRequest(req)
	a.focus = msgs.FocusEditor
	a.updateFocus()

	cmd := a.toast.Show("Template: "+tmpl.Name, false, 2*time.Second)
	return a, cmd
}

func (a App) importCurl() (tea.Model, tea.Cmd) {
	text, err := clipboard.ReadAll()
	if err != nil {
		cmd := a.toast.Show("Clipboard error: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}
	text = strings.TrimSpace(text)
	if text == "" {
		cmd := a.toast.Show("Clipboard is empty", true, 2*time.Second)
		return a, cmd
	}

	req, err := curlimport.ParseCurl(text)
	if err != nil {
		cmd := a.toast.Show("Invalid cURL: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}

	// Create a new tab with the imported request
	colReq := collection.NewRequest("Imported", req.Method, req.URL)
	for k, v := range req.Headers {
		colReq.Headers = append(colReq.Headers, collection.KVPair{Key: k, Value: v, Enabled: true})
	}
	for k, v := range req.Params {
		colReq.Params = append(colReq.Params, collection.KVPair{Key: k, Value: v, Enabled: true})
	}
	if len(req.Body) > 0 {
		colReq.Body = &collection.Body{Type: "json", Content: string(req.Body)}
	}
	if req.Auth != nil {
		colReq.Auth = authConfigToCollection(req.Auth)
	}

	a.store.OpenRequest(colReq)
	a.syncTabs()
	a.editor.LoadRequest(colReq)
	a.focus = msgs.FocusEditor
	a.updateFocus()

	cmd := a.toast.Show("Imported from cURL", false, 2*time.Second)
	return a, cmd
}
