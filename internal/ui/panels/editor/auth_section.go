package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/theme"
)

var authTypes = []string{"none", "basic", "bearer", "apikey"}

// AuthSection manages auth configuration with type selector and field inputs.
type AuthSection struct {
	authType    string // none, basic, bearer, apikey
	typeIndex   int
	cursor      int // 0=type, 1+=fields
	editing     bool
	activeInput int // which input is active for editing

	// Basic auth
	username textinput.Model
	password textinput.Model

	// Bearer
	token textinput.Model

	// API Key
	apiKeyName  textinput.Model
	apiKeyValue textinput.Model
	apiKeyIn    string // header, query
	apiKeyInIdx int    // 0=header, 1=query

	width  int
	styles theme.Styles
}

// NewAuthSection creates a new auth section.
func NewAuthSection(styles theme.Styles) AuthSection {
	mkInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 512
		ti.Width = 40
		return ti
	}

	return AuthSection{
		authType: "none",
		username: mkInput("Username"),
		password: mkInput("Password"),
		token:    mkInput("Bearer token"),
		apiKeyName:  mkInput("Key name (e.g. X-API-Key)"),
		apiKeyValue: mkInput("Key value"),
		apiKeyIn:    "header",
		styles:      styles,
	}
}

// SetSize updates the section width.
func (m *AuthSection) SetSize(w int) {
	m.width = w
	inputW := w - 16
	if inputW < 10 {
		inputW = 10
	}
	m.username.Width = inputW
	m.password.Width = inputW
	m.token.Width = inputW
	m.apiKeyName.Width = inputW
	m.apiKeyValue.Width = inputW
}

// Editing returns whether any field is being edited.
func (m AuthSection) Editing() bool {
	return m.editing
}

// BuildAuth returns a protocol.AuthConfig from the current state.
func (m AuthSection) BuildAuth() *protocol.AuthConfig {
	switch m.authType {
	case "basic":
		return &protocol.AuthConfig{
			Type:     "basic",
			Username: m.username.Value(),
			Password: m.password.Value(),
		}
	case "bearer":
		return &protocol.AuthConfig{
			Type:  "bearer",
			Token: m.token.Value(),
		}
	case "apikey":
		return &protocol.AuthConfig{
			Type:     "apikey",
			APIKey:   m.apiKeyName.Value(),
			APIValue: m.apiKeyValue.Value(),
			APIIn:    m.apiKeyIn,
		}
	default:
		return nil
	}
}

// LoadAuth loads auth configuration from a collection auth.
func (m *AuthSection) LoadAuth(auth *collection.Auth) {
	if auth == nil {
		m.authType = "none"
		m.typeIndex = 0
		return
	}
	m.authType = auth.Type
	for i, t := range authTypes {
		if t == auth.Type {
			m.typeIndex = i
			break
		}
	}
	switch auth.Type {
	case "basic":
		if auth.Basic != nil {
			m.username.SetValue(auth.Basic.Username)
			m.password.SetValue(auth.Basic.Password)
		}
	case "bearer":
		if auth.Bearer != nil {
			m.token.SetValue(auth.Bearer.Token)
		}
	case "apikey":
		if auth.APIKey != nil {
			m.apiKeyName.SetValue(auth.APIKey.Key)
			m.apiKeyValue.SetValue(auth.APIKey.Value)
			m.apiKeyIn = auth.APIKey.In
			if m.apiKeyIn == "query" {
				m.apiKeyInIdx = 1
			} else {
				m.apiKeyInIdx = 0
				m.apiKeyIn = "header"
			}
		}
	}
}

// Update handles input messages.
func (m AuthSection) Update(msg tea.Msg) (AuthSection, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNormal(msg)
}

func (m AuthSection) updateNormal(msg tea.Msg) (AuthSection, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			maxCursor := m.maxCursor()
			if m.cursor < maxCursor {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", " ":
			if m.cursor == 0 {
				// Cycle auth type
				m.typeIndex = (m.typeIndex + 1) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
				m.cursor = 0
			} else {
				// Start editing the focused field
				m.startEditing()
				return m, textinput.Blink
			}
		case "h", "left":
			if m.cursor == 0 {
				m.typeIndex = (m.typeIndex - 1 + len(authTypes)) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
			}
			// For apikey "in" field
			if m.authType == "apikey" && m.cursor == 3 {
				m.apiKeyInIdx = (m.apiKeyInIdx + 1) % 2
				if m.apiKeyInIdx == 0 {
					m.apiKeyIn = "header"
				} else {
					m.apiKeyIn = "query"
				}
			}
		case "l", "right":
			if m.cursor == 0 {
				m.typeIndex = (m.typeIndex + 1) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
			}
			if m.authType == "apikey" && m.cursor == 3 {
				m.apiKeyInIdx = (m.apiKeyInIdx + 1) % 2
				if m.apiKeyInIdx == 0 {
					m.apiKeyIn = "header"
				} else {
					m.apiKeyIn = "query"
				}
			}
		}
	}
	return m, nil
}

func (m AuthSection) updateEditing(msg tea.Msg) (AuthSection, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			m.blurAll()
			m.editing = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.authType {
	case "basic":
		if m.cursor == 1 {
			m.username, cmd = m.username.Update(msg)
		} else if m.cursor == 2 {
			m.password, cmd = m.password.Update(msg)
		}
	case "bearer":
		if m.cursor == 1 {
			m.token, cmd = m.token.Update(msg)
		}
	case "apikey":
		if m.cursor == 1 {
			m.apiKeyName, cmd = m.apiKeyName.Update(msg)
		} else if m.cursor == 2 {
			m.apiKeyValue, cmd = m.apiKeyValue.Update(msg)
		}
	}
	return m, cmd
}

func (m *AuthSection) startEditing() {
	m.editing = true
	switch m.authType {
	case "basic":
		if m.cursor == 1 {
			m.username.Focus()
			m.username.CursorEnd()
		} else if m.cursor == 2 {
			m.password.Focus()
			m.password.CursorEnd()
		}
	case "bearer":
		if m.cursor == 1 {
			m.token.Focus()
			m.token.CursorEnd()
		}
	case "apikey":
		if m.cursor == 1 {
			m.apiKeyName.Focus()
			m.apiKeyName.CursorEnd()
		} else if m.cursor == 2 {
			m.apiKeyValue.Focus()
			m.apiKeyValue.CursorEnd()
		}
	}
}

func (m *AuthSection) blurAll() {
	m.username.Blur()
	m.password.Blur()
	m.token.Blur()
	m.apiKeyName.Blur()
	m.apiKeyValue.Blur()
}

func (m AuthSection) maxCursor() int {
	switch m.authType {
	case "basic":
		return 2 // type, username, password
	case "bearer":
		return 1 // type, token
	case "apikey":
		return 3 // type, key, value, in
	default:
		return 0 // none: just type
	}
}

// View renders the auth section.
func (m AuthSection) View() string {
	var lines []string

	// Type selector row
	typeLabel := "  Type: "
	if m.cursor == 0 {
		typeLabel = "> Type: "
	}

	var typeParts []string
	for i, t := range authTypes {
		if i == m.typeIndex {
			typeParts = append(typeParts, m.styles.TabActive.Render(t))
		} else {
			typeParts = append(typeParts, m.styles.TabInactive.Render(t))
		}
	}
	lines = append(lines, typeLabel+strings.Join(typeParts, " "))

	switch m.authType {
	case "none":
		lines = append(lines, "")
		lines = append(lines, m.styles.Muted.Render("  No authentication"))

	case "basic":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Username", m.username, 1))
		lines = append(lines, m.renderField("Password", m.password, 2))

	case "bearer":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Token", m.token, 1))

	case "apikey":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Key", m.apiKeyName, 1))
		lines = append(lines, m.renderField("Value", m.apiKeyValue, 2))
		// In selector
		prefix := "  "
		if m.cursor == 3 {
			prefix = "> "
		}
		inLabel := prefix + m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render("Send In")) + " "
		var inParts []string
		inOptions := []string{"header", "query"}
		for i, opt := range inOptions {
			if i == m.apiKeyInIdx {
				inParts = append(inParts, m.styles.TabActive.Render(opt))
			} else {
				inParts = append(inParts, m.styles.TabInactive.Render(opt))
			}
		}
		lines = append(lines, inLabel+strings.Join(inParts, " "))
	}

	return strings.Join(lines, "\n")
}

func (m AuthSection) renderField(label string, input textinput.Model, fieldIdx int) string {
	prefix := "  "
	if m.cursor == fieldIdx {
		prefix = "> "
	}

	labelStr := m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render(label))

	if m.cursor == fieldIdx && m.editing {
		return prefix + labelStr + " " + input.View()
	}

	val := input.Value()
	if val == "" {
		val = input.Placeholder
		return prefix + labelStr + " " + m.styles.Muted.Render(val)
	}
	return prefix + labelStr + " " + m.styles.Normal.Render(val)
}
