package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/theme"
)

// GQLSubTab identifies the active sub-tab in the GraphQL form.
type GQLSubTab int

const (
	GQLTabQuery GQLSubTab = iota
	GQLTabVariables
	GQLTabHeaders
	GQLTabAuth
)

var gqlSubTabNames = []string{"Query", "Variables", "Headers", "Auth"}

// GraphQLForm is the GraphQL request form component.
type GraphQLForm struct {
	url       textinput.Model
	query     textarea.Model
	variables textarea.Model
	headers   components.KVTable
	auth      AuthSection

	activeTab  GQLSubTab
	focusField int // 0=url, 1=sub-tab content

	width  int
	height int
	styles theme.Styles
}

// NewGraphQLForm creates a new GraphQL form.
func NewGraphQLForm(styles theme.Styles) GraphQLForm {
	urlInput := textinput.New()
	urlInput.Placeholder = "GraphQL endpoint URL..."
	urlInput.CharLimit = 2048
	urlInput.Width = 40

	queryArea := textarea.New()
	queryArea.Placeholder = "query {\n  ...\n}"
	queryArea.ShowLineNumbers = true
	queryArea.CharLimit = 0
	queryArea.SetWidth(40)
	queryArea.SetHeight(8)

	varsArea := textarea.New()
	varsArea.Placeholder = `{"key": "value"}`
	varsArea.ShowLineNumbers = false
	varsArea.CharLimit = 0
	varsArea.SetWidth(40)
	varsArea.SetHeight(6)

	headers := components.NewKVTable(styles)
	headers.SetPairs([]components.KVPair{
		{Key: "Content-Type", Value: "application/json", Enabled: true},
	})

	return GraphQLForm{
		url:       urlInput,
		query:     queryArea,
		variables: varsArea,
		headers:   headers,
		auth:      NewAuthSection(styles),
		activeTab: GQLTabQuery,
		styles:    styles,
		width:     60,
		height:    20,
	}
}

// SetSize updates the form dimensions.
func (m *GraphQLForm) SetSize(w, h int) {
	m.width = w
	m.height = h

	urlW := w - 4
	if urlW < 10 {
		urlW = 10
	}
	m.url.Width = urlW

	contentW := w - 2
	if contentW < 10 {
		contentW = 10
	}
	m.headers.SetSize(contentW)
	m.auth.SetSize(contentW)

	bodyH := h - 6
	if bodyH < 3 {
		bodyH = 3
	}
	m.query.SetWidth(contentW)
	m.query.SetHeight(bodyH)
	m.variables.SetWidth(contentW)
	m.variables.SetHeight(bodyH)
}

// FocusURL focuses the URL input.
func (m *GraphQLForm) FocusURL() {
	m.focusField = 0
	m.url.Focus()
	m.url.CursorEnd()
}

// Editing returns whether any input is in text editing mode.
func (m GraphQLForm) Editing() bool {
	if m.focusField == 0 && m.url.Focused() {
		return true
	}
	if m.focusField == 1 {
		switch m.activeTab {
		case GQLTabQuery:
			return m.query.Focused()
		case GQLTabVariables:
			return m.variables.Focused()
		case GQLTabHeaders:
			return m.headers.Editing()
		case GQLTabAuth:
			return m.auth.Editing()
		}
	}
	return false
}

// BuildRequest constructs a protocol.Request from the GraphQL form.
func (m GraphQLForm) BuildRequest() *protocol.Request {
	req := &protocol.Request{
		Protocol:         "graphql",
		Method:           "POST",
		URL:              m.url.Value(),
		Headers:          make(map[string]string),
		GraphQLQuery:     strings.TrimSpace(m.query.Value()),
		GraphQLVariables: strings.TrimSpace(m.variables.Value()),
	}

	for _, h := range m.headers.GetPairs() {
		if h.Enabled && h.Key != "" {
			req.Headers[h.Key] = h.Value
		}
	}

	req.Auth = m.auth.BuildAuth()
	return req
}

// BuildAuth returns the auth config.
func (m GraphQLForm) BuildAuth() *protocol.AuthConfig {
	return m.auth.BuildAuth()
}

// GetHeaders returns header pairs.
func (m GraphQLForm) GetHeaders() []components.KVPair {
	return m.headers.GetPairs()
}

// GetBodyContent returns the query text.
func (m GraphQLForm) GetBodyContent() string {
	return strings.TrimSpace(m.query.Value())
}

// SetBody sets the query text.
func (m *GraphQLForm) SetBody(content string) {
	m.query.SetValue(content)
}

// GetParams returns empty params (GraphQL doesn't use params).
func (m GraphQLForm) GetParams() []components.KVPair {
	return nil
}

// LoadRequest populates the form from a collection request.
func (m *GraphQLForm) LoadRequest(req *collection.Request) {
	m.url.SetValue(req.URL)

	if req.GraphQL != nil {
		m.query.SetValue(req.GraphQL.Query)
		m.variables.SetValue(req.GraphQL.Variables)
	}

	if len(req.Headers) > 0 {
		kvPairs := make([]components.KVPair, len(req.Headers))
		for i, h := range req.Headers {
			kvPairs[i] = components.KVPair{Key: h.Key, Value: h.Value, Enabled: h.Enabled}
		}
		m.headers.SetPairs(kvPairs)
	}

	m.auth.LoadAuth(req.Auth)
	m.focusField = 0
}

// Init implements tea.Model.
func (m GraphQLForm) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m GraphQLForm) Update(msg tea.Msg) (GraphQLForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Editing() {
			return m.updateEditing(msg)
		}
		return m.updateNormal(msg)
	}

	// Non-key messages
	if m.focusField == 0 {
		var cmd tea.Cmd
		m.url, cmd = m.url.Update(msg)
		return m, cmd
	}
	if m.focusField == 1 {
		return m.updateTabContent(msg)
	}
	return m, nil
}

func (m GraphQLForm) updateNormal(msg tea.KeyMsg) (GraphQLForm, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "shift+tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "enter":
		if m.focusField == 0 {
			m.url.Focus()
			return m, textinput.Blink
		}
		return m.enterTabContent()
	case "h", "left":
		if m.focusField == 1 && m.activeTab > GQLTabQuery {
			m.activeTab--
		}
	case "l", "right":
		if m.focusField == 1 && m.activeTab < GQLTabAuth {
			m.activeTab++
		}
	case "1":
		m.activeTab = GQLTabQuery
	case "2":
		m.activeTab = GQLTabVariables
	case "3":
		m.activeTab = GQLTabHeaders
	case "4":
		m.activeTab = GQLTabAuth
	default:
		if m.focusField == 1 {
			return m.updateTabContent(msg)
		}
	}
	return m, nil
}

func (m GraphQLForm) updateEditing(msg tea.KeyMsg) (GraphQLForm, tea.Cmd) {
	if m.focusField == 0 {
		if msg.String() == "esc" {
			m.url.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.url, cmd = m.url.Update(msg)
		return m, cmd
	}

	if m.focusField == 1 {
		switch m.activeTab {
		case GQLTabQuery:
			if msg.String() == "esc" {
				m.query.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.query, cmd = m.query.Update(msg)
			return m, cmd
		case GQLTabVariables:
			if msg.String() == "esc" {
				m.variables.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.variables, cmd = m.variables.Update(msg)
			return m, cmd
		case GQLTabHeaders:
			if msg.String() == "esc" && !m.headers.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.headers, cmd = m.headers.Update(msg)
			return m, cmd
		case GQLTabAuth:
			if msg.String() == "esc" && !m.auth.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.auth, cmd = m.auth.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *GraphQLForm) enterTabContent() (GraphQLForm, tea.Cmd) {
	switch m.activeTab {
	case GQLTabQuery:
		cmd := m.query.Focus()
		return *m, cmd
	case GQLTabVariables:
		cmd := m.variables.Focus()
		return *m, cmd
	case GQLTabHeaders:
		var cmd tea.Cmd
		m.headers, cmd = m.headers.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case GQLTabAuth:
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	}
	return *m, nil
}

func (m GraphQLForm) updateTabContent(msg tea.Msg) (GraphQLForm, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case GQLTabQuery:
		m.query, cmd = m.query.Update(msg)
	case GQLTabVariables:
		m.variables, cmd = m.variables.Update(msg)
	case GQLTabHeaders:
		m.headers, cmd = m.headers.Update(msg)
	case GQLTabAuth:
		m.auth, cmd = m.auth.Update(msg)
	}
	return m, cmd
}

func (m *GraphQLForm) syncFocus() {
	m.url.Blur()
	m.query.Blur()
	m.variables.Blur()
}

// View renders the GraphQL form.
func (m GraphQLForm) View() string {
	var b strings.Builder

	// URL bar
	urlLabel := m.styles.Hint.Render("POST ")
	if m.focusField == 0 {
		urlLabel = m.styles.Cursor.Render(" POST ")
	}
	b.WriteString(urlLabel + " " + m.url.View())
	b.WriteString("\n\n")

	// Sub-tab bar
	var tabs []string
	for i, name := range gqlSubTabNames {
		if GQLSubTab(i) == m.activeTab {
			tabs = append(tabs, m.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.TabInactive.Render(name))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n\n")

	// Tab content
	switch m.activeTab {
	case GQLTabQuery:
		b.WriteString(m.query.View())
	case GQLTabVariables:
		b.WriteString(m.variables.View())
	case GQLTabHeaders:
		b.WriteString(m.headers.View())
	case GQLTabAuth:
		b.WriteString(m.auth.View())
	}

	return b.String()
}
