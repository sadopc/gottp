package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	oauth2auth "github.com/serdar/gottp/internal/auth/oauth2"
	"github.com/serdar/gottp/internal/config"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/core/environment"
	"github.com/serdar/gottp/internal/core/history"
	"github.com/serdar/gottp/internal/core/state"
	"github.com/serdar/gottp/internal/export"
	curlimport "github.com/serdar/gottp/internal/import/curl"
	importutil "github.com/serdar/gottp/internal/import"
	"github.com/serdar/gottp/internal/import/insomnia"
	"github.com/serdar/gottp/internal/import/openapi"
	"github.com/serdar/gottp/internal/import/postman"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/protocol/graphql"
	grpcclient "github.com/serdar/gottp/internal/protocol/grpc"
	httpclient "github.com/serdar/gottp/internal/protocol/http"
	wsclient "github.com/serdar/gottp/internal/protocol/websocket"
	"github.com/serdar/gottp/internal/scripting"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/layout"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/panels/editor"
	"github.com/serdar/gottp/internal/ui/panels/response"
	"github.com/serdar/gottp/internal/ui/panels/sidebar"
	"github.com/serdar/gottp/internal/ui/theme"
)

// App is the root Bubble Tea model.
type App struct {
	sidebar  sidebar.Model
	editor   editor.Model
	response response.Model

	tabBar         components.TabBar
	statusBar      components.StatusBar
	commandPalette components.CommandPalette
	help           components.Help
	toast          components.Toast
	modal          components.Modal
	jump           components.JumpOverlay

	store        *state.Store
	protocols    *protocol.Registry
	scriptEngine *scripting.Engine
	envFile      *environment.EnvironmentFile
	cfg          config.Config
	history      *history.Store

	mode           msgs.AppMode
	focus          msgs.PanelFocus
	sidebarVisible bool
	layout         layout.PanelLayout
	keys           KeyMap

	theme  theme.Theme
	styles theme.Styles

	width  int
	height int
	ready  bool
}

// New creates a new App model.
func New(col *collection.Collection, colPath string, cfg config.Config) App {
	t := theme.Resolve(cfg.Theme)
	s := theme.NewStyles(t)

	store := state.NewStore()
	store.Collection = col
	store.CollectionPath = colPath
	store.NewTab()

	// Set up protocol registry
	registry := protocol.NewRegistry()
	httpClient := httpclient.New()
	if cfg.DefaultTimeout > 0 {
		httpClient.SetTimeout(cfg.DefaultTimeout)
	}
	registry.Register(httpClient)
	registry.Register(graphql.New())
	registry.Register(wsclient.New())
	registry.Register(grpcclient.New())

	// Init scripting engine
	scriptTimeout := cfg.ScriptTimeout
	if scriptTimeout == 0 {
		scriptTimeout = 5 * time.Second
	}
	scriptEngine := scripting.NewEngine(scriptTimeout)

	// Load environments from environments.yaml next to the collection file
	var envFile *environment.EnvironmentFile
	if colPath != "" {
		dir := filepath.Dir(colPath)
		ef, err := environment.LoadEnvironments(filepath.Join(dir, "environments.yaml"))
		if err == nil && len(ef.Environments) > 0 {
			envFile = ef
			// Auto-select first environment
			store.ActiveEnv = ef.Environments[0].Name
			store.EnvVars = ef.GetVariables(store.ActiveEnv)
		}
	}

	// Init history store
	var histStore *history.Store
	dataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "gottp")
	os.MkdirAll(dataDir, 0755)
	if hs, err := history.NewStore(filepath.Join(dataDir, "history.db")); err == nil {
		histStore = hs
	}

	a := App{
		sidebar:  sidebar.New(t, s),
		editor:   editor.New(t, s),
		response: response.New(t, s),

		tabBar:         components.NewTabBar(t, s),
		statusBar:      components.NewStatusBar(t, s),
		commandPalette: components.NewCommandPalette(t, s),
		help:           components.NewHelp(t, s),
		toast:          components.NewToast(t, s),
		modal:          components.NewModal(t, s),
		jump:           components.NewJumpOverlay(t, s),

		store:        store,
		protocols:    registry,
		scriptEngine: scriptEngine,
		envFile:      envFile,
		cfg:          cfg,
		history:      histStore,

		mode:           msgs.ModeNormal,
		focus:          msgs.FocusEditor,
		sidebarVisible: true,
		keys:           DefaultKeyMap(),

		theme:  t,
		styles: s,
	}

	if col != nil {
		items := collection.FlattenItems(col.Items, 0, "")
		a.sidebar.SetItems(items)
	}

	if store.ActiveEnv != "" {
		a.statusBar.SetEnv(store.ActiveEnv)
	}

	// Load recent history into sidebar
	a.loadHistory()

	a.syncTabs()
	return a
}

func (a App) Init() tea.Cmd {
	return a.response.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layout = layout.HandleResize(msg, a.sidebarVisible)
		a.resizePanels()
		a.ready = true
		return a, nil

	case tea.KeyMsg:
		if a.commandPalette.Visible {
			var cmd tea.Cmd
			a.commandPalette, cmd = a.commandPalette.Update(msg)
			return a, cmd
		}
		if a.help.Visible {
			var cmd tea.Cmd
			a.help, cmd = a.help.Update(msg)
			return a, cmd
		}
		if a.modal.Visible {
			var cmd tea.Cmd
			a.modal, cmd = a.modal.Update(msg)
			return a, cmd
		}
		if a.jump.Visible {
			var cmd tea.Cmd
			a.jump, cmd = a.jump.Update(msg)
			return a, cmd
		}

		if a.focus == msgs.FocusEditor && a.editor.Editing() {
			return a.updateEditorInsert(msg)
		}

		cmd := a.handleGlobalKey(msg)
		if cmd != nil {
			return a, cmd
		}
		return a.handlePanelKey(msg)

	case msgs.SendRequestMsg:
		return a.sendRequest()

	case msgs.RequestSentMsg:
		return a.handleRequestSent(msg)

	case msgs.NewRequestMsg:
		a.store.NewTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.CloseTabMsg:
		a.store.CloseTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.NextTabMsg:
		a.store.NextTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.PrevTabMsg:
		a.store.PrevTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.SwitchTabMsg:
		if msg.Index >= 0 && msg.Index < len(a.store.Tabs) {
			a.store.ActiveTab = msg.Index
			a.syncTabs()
			a.loadActiveRequest()
		}
		return a, nil

	case msgs.SwitchEnvMsg:
		if msg.Name != "" && a.envFile != nil {
			a.store.ActiveEnv = msg.Name
			a.store.EnvVars = a.envFile.GetVariables(msg.Name)
			a.statusBar.SetEnv(msg.Name)
			cmd := a.toast.Show("Environment: "+msg.Name, false, 2*time.Second)
			return a, cmd
		}
		// If Name is empty, open env picker via command palette
		if a.envFile != nil && len(a.envFile.Environments) > 0 {
			names := a.envFile.Names()
			a.commandPalette.OpenEnvPicker(names)
			a.mode = msgs.ModeCommandPalette
			return a, nil
		}
		cmd := a.toast.Show("No environments found", true, 2*time.Second)
		return a, cmd

	case msgs.SwitchThemeMsg:
		return a.handleSwitchTheme(msg)

	case msgs.ToggleSidebarMsg:
		a.sidebarVisible = !a.sidebarVisible
		a.layout = layout.Calculate(a.width, a.height, a.sidebarVisible)
		a.resizePanels()
		return a, nil

	case msgs.OpenCommandPaletteMsg:
		a.mode = msgs.ModeCommandPalette
		a.commandPalette.Open()
		return a, nil

	case msgs.ShowHelpMsg:
		a.mode = msgs.ModeModal
		a.help.Toggle()
		return a, nil

	case msgs.SetModeMsg:
		a.mode = msg.Mode
		a.statusBar.SetMode(msg.Mode)
		return a, nil

	case msgs.SaveRequestMsg:
		return a.saveCollection()

	case msgs.RequestSelectedMsg:
		return a.handleRequestSelected(msg)

	case msgs.StatusMsg:
		a.statusBar.SetMessage(msg.Text)
		if msg.Duration > 0 {
			cmds = append(cmds, tea.Tick(msg.Duration, func(time.Time) tea.Msg {
				return msgs.StatusMsg{Text: ""}
			}))
		}
		return a, tea.Batch(cmds...)

	case msgs.ToastMsg:
		cmd := a.toast.Show(msg.Text, msg.IsError, msg.Duration)
		return a, cmd

	case msgs.CopyAsCurlMsg:
		return a.copyAsCurl()

	case msgs.ImportCurlMsg:
		return a.importCurl()

	case msgs.ImportFileMsg:
		return a.handleImportFile(msg)

	case msgs.ImportCompleteMsg:
		return a.handleImportComplete(msg)

	case msgs.SetBaselineMsg:
		return a.handleSetBaseline()

	case msgs.ClearBaselineMsg:
		a.response.ClearBaseline()
		cmd := a.toast.Show("Baseline cleared", false, 2*time.Second)
		return a, cmd

	case msgs.OAuth2TokenMsg:
		return a.handleOAuth2Token(msg)

	case msgs.HistorySelectedMsg:
		return a.handleHistorySelected(msg)

	case msgs.FocusPanelMsg:
		a.focus = msg.Panel
		a.updateFocus()
		return a, nil

	case msgs.OpenEditorMsg:
		return a.openExternalEditor()

	case msgs.EditorDoneMsg:
		if msg.Content != "" {
			a.editor.SetBody(msg.Content)
			cmd := a.toast.Show("Body updated from editor", false, 2*time.Second)
			return a, cmd
		}
		return a, nil

	case msgs.SwitchProtocolMsg:
		a.editor.SetProtocol(msg.Protocol)
		a.response.SetMode(msg.Protocol)
		return a, nil

	case msgs.IntrospectMsg:
		return a.handleIntrospect()

	case msgs.IntrospectionResultMsg:
		return a.handleIntrospectionResult(msg)

	case msgs.ScriptResultMsg:
		return a.handleScriptResult(msg)

	case msgs.WSConnectedMsg:
		if msg.Err != nil {
			cmd := a.toast.Show("WebSocket error: "+msg.Err.Error(), true, 5*time.Second)
			return a, cmd
		}
		cmd := a.toast.Show("WebSocket connected", false, 2*time.Second)
		return a, cmd

	case msgs.WSDisconnectedMsg:
		if msg.Err != nil {
			cmd := a.toast.Show("WebSocket closed: "+msg.Err.Error(), true, 3*time.Second)
			return a, cmd
		}
		cmd := a.toast.Show("WebSocket disconnected", false, 2*time.Second)
		return a, cmd

	case msgs.WSMessageReceivedMsg:
		a.response.AddWSMessage(response.WSMessage{
			Direction: "received",
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			IsJSON:    msg.IsJSON,
		})
		return a, nil

	case msgs.GRPCReflectMsg:
		return a.handleGRPCReflect()

	case msgs.GRPCReflectionResultMsg:
		return a.handleGRPCReflectionResult(msg)
	}

	var cmd tea.Cmd
	a.toast, cmd = a.toast.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	a.statusBar, cmd = a.statusBar.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	a.response, cmd = a.response.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) handleGlobalKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, a.keys.Quit):
		return tea.Quit
	case key.Matches(msg, a.keys.SendRequest):
		return func() tea.Msg { return msgs.SendRequestMsg{} }
	case msg.String() == "ctrl+enter":
		return func() tea.Msg { return msgs.SendRequestMsg{} }
	case key.Matches(msg, a.keys.CommandPalette):
		return func() tea.Msg { return msgs.OpenCommandPaletteMsg{} }
	case key.Matches(msg, a.keys.NewRequest):
		return func() tea.Msg { return msgs.NewRequestMsg{} }
	case key.Matches(msg, a.keys.CloseTab):
		return func() tea.Msg { return msgs.CloseTabMsg{} }
	case key.Matches(msg, a.keys.SaveRequest):
		return func() tea.Msg { return msgs.SaveRequestMsg{} }
	case key.Matches(msg, a.keys.PrevTab):
		return func() tea.Msg { return msgs.PrevTabMsg{} }
	case key.Matches(msg, a.keys.NextTab):
		return func() tea.Msg { return msgs.NextTabMsg{} }
	}
	return nil
}

func (a App) handlePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		a.cycleFocus(false)
		return a, nil
	case "shift+tab":
		a.cycleFocus(true)
		return a, nil
	case "b":
		a.sidebarVisible = !a.sidebarVisible
		a.layout = layout.Calculate(a.width, a.height, a.sidebarVisible)
		a.resizePanels()
		return a, nil
	case "?":
		a.mode = msgs.ModeModal
		a.help.SetSize(a.width, a.height)
		a.help.Toggle()
		return a, nil
	case "i":
		// Enter insert mode: focus URL input in editor
		if a.focus == msgs.FocusEditor {
			a.mode = msgs.ModeInsert
			a.statusBar.SetMode(msgs.ModeInsert)
			a.editor.FocusURL()
			return a, nil
		}
	case "enter":
		// Send request when editor is focused in normal mode
		if a.focus == msgs.FocusEditor {
			return a.sendRequest()
		}
	case "S":
		// Capital S as alternative send shortcut (always works)
		return a.sendRequest()
	case "f":
		// Activate jump mode
		a.activateJumpMode()
		return a, nil
	case "E":
		// Open body in $EDITOR
		return a.openExternalEditor()
	}

	var cmd tea.Cmd
	switch a.focus {
	case msgs.FocusSidebar:
		a.sidebar, cmd = a.sidebar.Update(msg)
	case msgs.FocusEditor:
		a.editor, cmd = a.editor.Update(msg)
	case msgs.FocusResponse:
		a.response, cmd = a.response.Update(msg)
	}

	return a, cmd
}

func (a App) updateEditorInsert(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		a.mode = msgs.ModeNormal
		a.statusBar.SetMode(msgs.ModeNormal)
	}
	if msg.String() == "ctrl+enter" {
		return a.sendRequest()
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)

	if a.editor.Editing() {
		a.mode = msgs.ModeInsert
		a.statusBar.SetMode(msgs.ModeInsert)
	} else {
		a.mode = msgs.ModeNormal
		a.statusBar.SetMode(msgs.ModeNormal)
	}

	return a, cmd
}

func (a *App) cycleFocus(reverse bool) {
	panels := []msgs.PanelFocus{msgs.FocusSidebar, msgs.FocusEditor, msgs.FocusResponse}
	if !a.sidebarVisible {
		panels = []msgs.PanelFocus{msgs.FocusEditor, msgs.FocusResponse}
	}

	idx := 0
	for i, p := range panels {
		if p == a.focus {
			idx = i
			break
		}
	}

	if reverse {
		idx = (idx - 1 + len(panels)) % len(panels)
	} else {
		idx = (idx + 1) % len(panels)
	}

	a.focus = panels[idx]
	a.updateFocus()
}

func (a *App) updateFocus() {
	a.sidebar.SetFocused(a.focus == msgs.FocusSidebar)
	a.editor.SetFocused(a.focus == msgs.FocusEditor)
	a.response.SetFocused(a.focus == msgs.FocusResponse)
}

func (a *App) resizePanels() {
	l := a.layout
	a.sidebar.SetSize(l.SidebarWidth, l.ContentHeight)
	a.editor.SetSize(l.EditorWidth, l.ContentHeight)
	a.response.SetSize(l.ResponseWidth, l.ContentHeight)
	a.tabBar.SetWidth(a.width)
	a.statusBar.SetWidth(a.width)
	a.help.SetSize(a.width, a.height)
	a.updateFocus()
}

func (a *App) syncTabs() {
	tabs := make([]components.TabItem, len(a.store.Tabs))
	for i, t := range a.store.Tabs {
		tabs[i] = components.TabItem{
			Name:   t.Request.Name,
			Method: t.Request.Method,
		}
	}
	a.tabBar.SetTabs(tabs)
	a.tabBar.SetActive(a.store.ActiveTab)
}

func (a *App) loadActiveRequest() {
	req := a.store.ActiveRequest()
	if req != nil {
		a.editor.LoadRequest(req)
	}
}

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

func (a *App) loadHistory() {
	if a.history == nil {
		return
	}
	entries, err := a.history.List(20, 0)
	if err != nil {
		return
	}
	items := make([]sidebar.HistoryItem, len(entries))
	for i, e := range entries {
		items[i] = sidebar.HistoryItem{
			ID:         e.ID,
			Method:     e.Method,
			URL:        e.URL,
			StatusCode: e.StatusCode,
			Duration:   e.Duration,
			Timestamp:  e.Timestamp,
		}
	}
	a.sidebar.SetHistory(items)
}

func (a App) handleRequestSelected(msg msgs.RequestSelectedMsg) (tea.Model, tea.Cmd) {
	if a.store.Collection == nil {
		return a, nil
	}
	req := findRequest(a.store.Collection.Items, msg.RequestID)
	if req != nil {
		a.store.OpenRequest(req)
		a.syncTabs()
		a.editor.LoadRequest(req)
		a.focus = msgs.FocusEditor
		a.updateFocus()
	}
	return a, nil
}

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
			req.Auth = a.authConfigToCollection(authConfig)
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

func (a App) authConfigToCollection(auth *protocol.AuthConfig) *collection.Auth {
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
				AccessKeyID:    auth.AWSAuth.AccessKeyID,
				SecretAccessKey: auth.AWSAuth.SecretAccessKey,
				SessionToken:   auth.AWSAuth.SessionToken,
				Region:         auth.AWSAuth.Region,
				Service:        auth.AWSAuth.Service,
			}
		}
	}
	return ca
}

func (a App) handleSwitchTheme(msg msgs.SwitchThemeMsg) (tea.Model, tea.Cmd) {
	if msg.Name == "" {
		// Open theme picker
		names := theme.Names()
		a.commandPalette.OpenThemePicker(names)
		a.mode = msgs.ModeCommandPalette
		return a, nil
	}

	t := theme.Resolve(msg.Name)
	s := theme.NewStyles(t)
	a.theme = t
	a.styles = s

	// Rebuild all components with new styles
	a.sidebar = sidebar.New(t, s)
	a.response = response.New(t, s)
	a.tabBar = components.NewTabBar(t, s)
	a.statusBar = components.NewStatusBar(t, s)
	a.commandPalette = components.NewCommandPalette(t, s)
	a.help = components.NewHelp(t, s)
	a.toast = components.NewToast(t, s)
	a.modal = components.NewModal(t, s)
	a.jump = components.NewJumpOverlay(t, s)

	// Re-set state
	if a.store.Collection != nil {
		items := collection.FlattenItems(a.store.Collection.Items, 0, "")
		a.sidebar.SetItems(items)
	}
	a.loadHistory()
	if a.store.ActiveEnv != "" {
		a.statusBar.SetEnv(a.store.ActiveEnv)
	}
	a.statusBar.SetMode(a.mode)
	a.syncTabs()
	a.resizePanels()

	cmd := a.toast.Show("Theme: "+t.Name, false, 2*time.Second)
	return a, cmd
}

func (a App) handleImportFile(msg msgs.ImportFileMsg) (tea.Model, tea.Cmd) {
	// For file-based import, we'd need a file picker. For now, use clipboard content.
	text, err := clipboard.ReadAll()
	if err != nil {
		cmd := a.toast.Show("Clipboard error: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}
	text = strings.TrimSpace(text)
	if text == "" {
		cmd := a.toast.Show("Clipboard is empty. Copy file content first.", true, 2*time.Second)
		return a, cmd
	}

	data := []byte(text)
	cmd := func() tea.Msg {
		format := msg.Path // hint from command
		if format == "" {
			format = importutil.DetectFormat(data)
		}

		var col *collection.Collection
		var parseErr error

		switch format {
		case "postman":
			col, parseErr = postman.ParsePostman(data)
		case "insomnia":
			col, parseErr = insomnia.ParseInsomnia(data)
		case "openapi":
			col, parseErr = openapi.ParseOpenAPI(data)
		default:
			// Try auto-detection
			detected := importutil.DetectFormat(data)
			switch detected {
			case "postman":
				col, parseErr = postman.ParsePostman(data)
			case "insomnia":
				col, parseErr = insomnia.ParseInsomnia(data)
			case "openapi":
				col, parseErr = openapi.ParseOpenAPI(data)
			default:
				return msgs.ImportCompleteMsg{Err: os.ErrInvalid}
			}
		}

		return msgs.ImportCompleteMsg{Collection: col, Err: parseErr}
	}

	return a, cmd
}

func (a App) handleImportComplete(msg msgs.ImportCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toast.Show("Import failed: "+msg.Err.Error(), true, 3*time.Second)
		return a, cmd
	}

	if msg.Collection == nil {
		cmd := a.toast.Show("No data imported", true, 2*time.Second)
		return a, cmd
	}

	// Merge into current collection or set as new
	if a.store.Collection == nil {
		a.store.Collection = msg.Collection
	} else {
		a.store.Collection.Items = append(a.store.Collection.Items, msg.Collection.Items...)
	}

	items := collection.FlattenItems(a.store.Collection.Items, 0, "")
	a.sidebar.SetItems(items)

	cmd := a.toast.Show("Imported "+msg.Collection.Name, false, 2*time.Second)
	return a, cmd
}

func (a App) handleSetBaseline() (tea.Model, tea.Cmd) {
	body := a.response.ResponseBody()
	if len(body) == 0 {
		cmd := a.toast.Show("No response to use as baseline", true, 2*time.Second)
		return a, cmd
	}
	a.response.SetBaseline(body)
	cmd := a.toast.Show("Baseline set", false, 2*time.Second)
	return a, cmd
}

func (a App) openExternalEditor() (tea.Model, tea.Cmd) {
	editorCmd := a.cfg.Editor
	if editorCmd == "" {
		editorCmd = os.Getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vi"
	}

	// Write body to temp file
	tmpFile, err := os.CreateTemp("", "gottp-body-*.txt")
	if err != nil {
		cmd := a.toast.Show("Failed to create temp file: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}

	bodyContent := a.editor.GetBodyContent()
	if bodyContent != "" {
		tmpFile.WriteString(bodyContent)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	c := exec.Command(editorCmd, tmpPath)
	return a, tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return msgs.EditorDoneMsg{}
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return msgs.EditorDoneMsg{}
		}
		return msgs.EditorDoneMsg{Content: string(data)}
	})
}

func (a *App) activateJumpMode() {
	targets := []components.JumpTarget{
		{Name: "Sidebar", Panel: msgs.FocusSidebar, Action: msgs.FocusPanelMsg{Panel: msgs.FocusSidebar}},
		{Name: "Editor", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
		{Name: "Response", Panel: msgs.FocusResponse, Action: msgs.FocusPanelMsg{Panel: msgs.FocusResponse}},
		{Name: "Send Request", Action: msgs.SendRequestMsg{}},
		{Name: "New Request", Action: msgs.NewRequestMsg{}},
		{Name: "Command Palette", Action: msgs.OpenCommandPaletteMsg{}},
		{Name: "Save", Action: msgs.SaveRequestMsg{}},
		{Name: "Toggle Sidebar", Action: msgs.ToggleSidebarMsg{}},
	}
	a.jump.Open(targets)
	a.mode = msgs.ModeJump
	a.statusBar.SetMode(msgs.ModeJump)
}

func (a App) handleHistorySelected(msg msgs.HistorySelectedMsg) (tea.Model, tea.Cmd) {
	if a.history == nil {
		return a, nil
	}
	entries, err := a.history.List(100, 0)
	if err != nil {
		return a, nil
	}
	for _, e := range entries {
		if e.ID == msg.ID {
			// Create a new tab with the history entry
			colReq := collection.NewRequest("History", e.Method, e.URL)
			if e.RequestBody != "" {
				colReq.Body = &collection.Body{Type: "json", Content: e.RequestBody}
			}
			if e.Headers != "" {
				var headers map[string]string
				if json.Unmarshal([]byte(e.Headers), &headers) == nil {
					for k, v := range headers {
						colReq.Headers = append(colReq.Headers, collection.KVPair{Key: k, Value: v, Enabled: true})
					}
				}
			}
			a.store.OpenRequest(colReq)
			a.syncTabs()
			a.editor.LoadRequest(colReq)
			a.focus = msgs.FocusEditor
			a.updateFocus()
			break
		}
	}
	return a, nil
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
		colReq.Auth = a.authConfigToCollection(req.Auth)
	}

	a.store.OpenRequest(colReq)
	a.syncTabs()
	a.editor.LoadRequest(colReq)
	a.focus = msgs.FocusEditor
	a.updateFocus()

	cmd := a.toast.Show("Imported from cURL", false, 2*time.Second)
	return a, cmd
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	tabBar := a.tabBar.View()

	var panels string
	if a.layout.SinglePanel {
		switch a.focus {
		case msgs.FocusSidebar:
			panels = a.sidebar.View()
		case msgs.FocusEditor:
			panels = a.editor.View()
		case msgs.FocusResponse:
			panels = a.response.View()
		}
	} else {
		var panelViews []string
		if a.sidebarVisible && !a.layout.TwoPanelMode {
			panelViews = append(panelViews, a.sidebar.View())
		}
		panelViews = append(panelViews, a.editor.View())
		panelViews = append(panelViews, a.response.View())
		panels = lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
	}

	statusBar := a.statusBar.View()
	main := lipgloss.JoinVertical(lipgloss.Left, tabBar, panels, statusBar)

	if a.commandPalette.Visible {
		main = overlayCenter(main, a.commandPalette.View(), a.width, a.height)
	}
	if a.help.Visible {
		main = overlayCenter(main, a.help.View(), a.width, a.height)
	}
	if a.modal.Visible {
		main = overlayCenter(main, a.modal.View(), a.width, a.height)
	}
	if a.jump.Visible {
		main = overlayCenter(main, a.jump.View(), a.width, a.height)
	}
	if a.toast.Visible {
		toastView := a.toast.View()
		main = overlayTopRight(main, toastView, a.width)
	}

	return main
}

func overlayCenter(_, overlay string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1e1e2e")),
	)
}

func overlayTopRight(bg, overlay string, width int) string {
	overlayWidth := lipgloss.Width(overlay)
	gap := width - overlayWidth - 2
	if gap < 0 {
		gap = 0
	}
	positioned := lipgloss.NewStyle().MarginLeft(gap).Render(overlay)
	return positioned + "\n" + bg
}

func findRequest(items []collection.Item, id string) *collection.Request {
	for i := range items {
		if items[i].Request != nil && items[i].Request.ID == id {
			return items[i].Request
		}
		if items[i].Folder != nil {
			if req := findRequest(items[i].Folder.Items, id); req != nil {
				return req
			}
		}
	}
	return nil
}
