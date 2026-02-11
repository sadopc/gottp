package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/protocol"
	"github.com/sadopc/gottp/internal/ui/components"
	"github.com/sadopc/gottp/internal/ui/theme"
)

var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// SubTab identifies the active sub-tab in the HTTP form.
type SubTab int

const (
	TabParams SubTab = iota
	TabHeaders
	TabAuth
	TabBody
)

var subTabNames = []string{"Params", "Headers", "Auth", "Body"}

// HTTPForm is the HTTP request form component.
type HTTPForm struct {
	Method      string
	methodIndex int

	url textinput.Model

	activeTab SubTab
	params    components.KVTable
	headers   components.KVTable
	auth      AuthSection
	body      textarea.Model

	// Focus tracking: 0=method, 1=url, 2=sub-tab content
	focusField int

	width  int
	height int
	styles theme.Styles
}

// NewHTTPForm creates a new HTTPForm.
func NewHTTPForm(styles theme.Styles) HTTPForm {
	urlInput := textinput.New()
	urlInput.Placeholder = "Enter URL..."
	urlInput.CharLimit = 2048
	urlInput.Width = 40

	bodyArea := textarea.New()
	bodyArea.Placeholder = "Request body..."
	bodyArea.ShowLineNumbers = false
	bodyArea.CharLimit = 0
	bodyArea.SetWidth(40)
	bodyArea.SetHeight(6)

	params := components.NewKVTable(styles)
	headers := components.NewKVTable(styles)

	// Default headers
	headers.SetPairs([]components.KVPair{
		{Key: "Content-Type", Value: "application/json", Enabled: true},
		{Key: "Accept", Value: "*/*", Enabled: true},
	})

	return HTTPForm{
		Method:      "GET",
		methodIndex: 0,
		url:         urlInput,
		activeTab:   TabParams,
		params:      params,
		headers:     headers,
		auth:        NewAuthSection(styles),
		body:        bodyArea,
		styles:      styles,
		width:       60,
		height:      20,
	}
}

// SetSize updates the form dimensions.
func (m *HTTPForm) SetSize(w, h int) {
	m.width = w
	m.height = h

	urlW := w - 12 // method label + padding
	if urlW < 10 {
		urlW = 10
	}
	m.url.Width = urlW

	contentW := w - 2
	if contentW < 10 {
		contentW = 10
	}
	m.params.SetSize(contentW)
	m.headers.SetSize(contentW)
	m.auth.SetSize(contentW)

	bodyH := h - 6 // url bar + tab bar + padding
	if bodyH < 3 {
		bodyH = 3
	}
	m.body.SetWidth(contentW)
	m.body.SetHeight(bodyH)
}

// URLFocused returns whether the URL input is focused.
func (m HTTPForm) URLFocused() bool {
	return m.focusField == 1
}

// FocusURL focuses the URL input field for editing.
func (m *HTTPForm) FocusURL() {
	m.focusField = 1
	m.url.Focus()
	m.url.CursorEnd()
}

// Editing returns whether any child is in text editing mode.
func (m HTTPForm) Editing() bool {
	if m.focusField == 1 && m.url.Focused() {
		return true
	}
	if m.focusField == 2 {
		switch m.activeTab {
		case TabParams:
			return m.params.Editing()
		case TabHeaders:
			return m.headers.Editing()
		case TabAuth:
			return m.auth.Editing()
		case TabBody:
			return m.body.Focused()
		}
	}
	return false
}

// Init implements tea.Model.
func (m HTTPForm) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m HTTPForm) Update(msg tea.Msg) (HTTPForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If a child is in editing mode, delegate first
		if m.Editing() {
			return m.updateEditing(msg)
		}
		return m.updateNormal(msg)
	}

	// Pass non-key messages to active inputs
	if m.focusField == 1 {
		var cmd tea.Cmd
		m.url, cmd = m.url.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focusField == 2 {
		cmds = append(cmds, m.updateTabContent(msg)...)
	}

	return m, tea.Batch(cmds...)
}

func (m HTTPForm) updateNormal(msg tea.KeyMsg) (HTTPForm, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focusField = (m.focusField + 1) % 3
		m.syncFocus()
	case "shift+tab":
		m.focusField = (m.focusField + 2) % 3
		m.syncFocus()
	case "enter":
		if m.focusField == 0 {
			m.cycleMethod()
		} else if m.focusField == 1 {
			m.url.Focus()
			return m, textinput.Blink
		} else if m.focusField == 2 {
			return m.enterTabContent()
		}
	case " ":
		if m.focusField == 0 {
			m.cycleMethod()
		}
	case "h", "left":
		if m.focusField == 2 {
			if m.activeTab > TabParams {
				m.activeTab--
			}
		}
	case "l", "right":
		if m.focusField == 2 {
			if m.activeTab < TabBody {
				m.activeTab++
			}
		}
	case "1":
		m.activeTab = TabParams
	case "2":
		m.activeTab = TabHeaders
	case "3":
		m.activeTab = TabAuth
	case "4":
		m.activeTab = TabBody
	default:
		if m.focusField == 2 {
			cmds := m.updateTabContent(msg)
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

func (m HTTPForm) updateEditing(msg tea.KeyMsg) (HTTPForm, tea.Cmd) {
	if m.focusField == 1 {
		switch msg.String() {
		case "esc":
			m.url.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.url, cmd = m.url.Update(msg)
		return m, cmd
	}

	if m.focusField == 2 {
		switch m.activeTab {
		case TabParams:
			if msg.String() == "esc" && !m.params.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.params, cmd = m.params.Update(msg)
			return m, cmd
		case TabHeaders:
			if msg.String() == "esc" && !m.headers.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.headers, cmd = m.headers.Update(msg)
			return m, cmd
		case TabAuth:
			if msg.String() == "esc" && !m.auth.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.auth, cmd = m.auth.Update(msg)
			return m, cmd
		case TabBody:
			switch msg.String() {
			case "esc":
				m.body.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.body, cmd = m.body.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *HTTPForm) enterTabContent() (HTTPForm, tea.Cmd) {
	switch m.activeTab {
	case TabParams:
		var cmd tea.Cmd
		m.params, cmd = m.params.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case TabHeaders:
		var cmd tea.Cmd
		m.headers, cmd = m.headers.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case TabAuth:
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case TabBody:
		cmd := m.body.Focus()
		return *m, cmd
	}
	return *m, nil
}

func (m *HTTPForm) updateTabContent(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	switch m.activeTab {
	case TabParams:
		var cmd tea.Cmd
		m.params, cmd = m.params.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabHeaders:
		var cmd tea.Cmd
		m.headers, cmd = m.headers.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabAuth:
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabBody:
		var cmd tea.Cmd
		m.body, cmd = m.body.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

func (m *HTTPForm) syncFocus() {
	m.url.Blur()
	m.body.Blur()
}

func (m *HTTPForm) cycleMethod() {
	m.methodIndex = (m.methodIndex + 1) % len(httpMethods)
	m.Method = httpMethods[m.methodIndex]
}

// GetParams returns the current param pairs.
func (m HTTPForm) GetParams() []components.KVPair {
	return m.params.GetPairs()
}

// GetHeaders returns the current header pairs.
func (m HTTPForm) GetHeaders() []components.KVPair {
	return m.headers.GetPairs()
}

// GetBodyContent returns the current body text.
func (m HTTPForm) GetBodyContent() string {
	return strings.TrimSpace(m.body.Value())
}

// SetBody sets the body content.
func (m *HTTPForm) SetBody(content string) {
	m.body.SetValue(content)
}

// BuildAuth returns the auth configuration from the auth section.
func (m HTTPForm) BuildAuth() *protocol.AuthConfig {
	return m.auth.BuildAuth()
}

// BuildRequest constructs a protocol.Request from the form state.
func (m HTTPForm) BuildRequest() *protocol.Request {
	req := &protocol.Request{
		Protocol: "http",
		Method:   m.Method,
		URL:      m.url.Value(),
		Headers:  make(map[string]string),
		Params:   make(map[string]string),
	}

	for _, p := range m.params.GetPairs() {
		if p.Enabled && p.Key != "" {
			req.Params[p.Key] = p.Value
		}
	}

	for _, h := range m.headers.GetPairs() {
		if h.Enabled && h.Key != "" {
			req.Headers[h.Key] = h.Value
		}
	}

	body := strings.TrimSpace(m.body.Value())
	if body != "" {
		req.Body = []byte(body)
	}

	req.Auth = m.auth.BuildAuth()

	return req
}

// LoadRequest populates the form from a saved collection request.
func (m *HTTPForm) LoadRequest(req *collection.Request) {
	m.Method = req.Method
	for i, method := range httpMethods {
		if method == req.Method {
			m.methodIndex = i
			break
		}
	}

	m.url.SetValue(req.URL)

	// Load params
	if len(req.Params) > 0 {
		kvPairs := make([]components.KVPair, len(req.Params))
		for i, p := range req.Params {
			kvPairs[i] = components.KVPair{Key: p.Key, Value: p.Value, Enabled: p.Enabled}
		}
		m.params.SetPairs(kvPairs)
	}

	// Load headers
	if len(req.Headers) > 0 {
		kvPairs := make([]components.KVPair, len(req.Headers))
		for i, h := range req.Headers {
			kvPairs[i] = components.KVPair{Key: h.Key, Value: h.Value, Enabled: h.Enabled}
		}
		m.headers.SetPairs(kvPairs)
	}

	// Load body
	if req.Body != nil {
		m.body.SetValue(req.Body.Content)
	}

	// Load auth
	m.auth.LoadAuth(req.Auth)

	m.focusField = 1
}

// View renders the HTTP form.
func (m HTTPForm) View() string {
	var b strings.Builder

	// URL bar: [METHOD] url-input
	methodStyle := m.styles.MethodStyle(m.Method)
	methodLabel := methodStyle.Render(m.Method)
	if m.focusField == 0 {
		methodLabel = m.styles.Cursor.Render(" " + m.Method + " ")
	}
	b.WriteString(methodLabel + " " + m.url.View())
	b.WriteString("\n\n")

	// Sub-tab bar
	var tabs []string
	for i, name := range subTabNames {
		if SubTab(i) == m.activeTab {
			tabs = append(tabs, m.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.TabInactive.Render(name))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n\n")

	// Tab content
	switch m.activeTab {
	case TabParams:
		b.WriteString(m.params.View())
	case TabHeaders:
		b.WriteString(m.headers.View())
	case TabAuth:
		b.WriteString(m.auth.View())
	case TabBody:
		b.WriteString(m.body.View())
	}

	return b.String()
}
