package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/protocol"
	"github.com/sadopc/gottp/internal/ui/components"
	"github.com/sadopc/gottp/internal/ui/msgs"
	"github.com/sadopc/gottp/internal/ui/theme"
)

// WSSubTab identifies the active sub-tab in the WebSocket form.
type WSSubTab int

const (
	WSTabConnection WSSubTab = iota
	WSTabHeaders
	WSTabAuth
	WSTabMessages
)

var wsSubTabNames = []string{"Connection", "Headers", "Auth", "Messages"}

// WebSocketForm is the WebSocket request form component.
type WebSocketForm struct {
	url     textinput.Model
	message textarea.Model
	headers components.KVTable
	auth    AuthSection

	activeTab  WSSubTab
	focusField int // 0=url, 1=sub-tab content
	connected  bool

	width  int
	height int
	styles theme.Styles
}

// NewWebSocketForm creates a new WebSocket form.
func NewWebSocketForm(styles theme.Styles) WebSocketForm {
	urlInput := textinput.New()
	urlInput.Placeholder = "ws://localhost:8080/ws"
	urlInput.CharLimit = 2048
	urlInput.Width = 40

	msgArea := textarea.New()
	msgArea.Placeholder = "Message to send..."
	msgArea.ShowLineNumbers = false
	msgArea.CharLimit = 0
	msgArea.SetWidth(40)
	msgArea.SetHeight(6)

	return WebSocketForm{
		url:       urlInput,
		message:   msgArea,
		headers:   components.NewKVTable(styles),
		auth:      NewAuthSection(styles),
		activeTab: WSTabConnection,
		styles:    styles,
		width:     60,
		height:    20,
	}
}

// SetSize updates the form dimensions.
func (m *WebSocketForm) SetSize(w, h int) {
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
	m.message.SetWidth(contentW)
	m.message.SetHeight(bodyH)
}

// FocusURL focuses the URL input.
func (m *WebSocketForm) FocusURL() {
	m.focusField = 0
	m.url.Focus()
	m.url.CursorEnd()
}

// SetConnected sets the connection state.
func (m *WebSocketForm) SetConnected(connected bool) {
	m.connected = connected
}

// Editing returns whether any input is in editing mode.
func (m WebSocketForm) Editing() bool {
	if m.focusField == 0 && m.url.Focused() {
		return true
	}
	if m.focusField == 1 {
		switch m.activeTab {
		case WSTabMessages:
			return m.message.Focused()
		case WSTabHeaders:
			return m.headers.Editing()
		case WSTabAuth:
			return m.auth.Editing()
		}
	}
	return false
}

// BuildRequest constructs a protocol.Request for WebSocket connection.
func (m WebSocketForm) BuildRequest() *protocol.Request {
	req := &protocol.Request{
		Protocol: "websocket",
		Method:   "GET",
		URL:      m.url.Value(),
		Headers:  make(map[string]string),
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
func (m WebSocketForm) BuildAuth() *protocol.AuthConfig {
	return m.auth.BuildAuth()
}

// GetHeaders returns header pairs.
func (m WebSocketForm) GetHeaders() []components.KVPair {
	return m.headers.GetPairs()
}

// GetParams returns empty params.
func (m WebSocketForm) GetParams() []components.KVPair {
	return nil
}

// GetBodyContent returns the message text.
func (m WebSocketForm) GetBodyContent() string {
	return strings.TrimSpace(m.message.Value())
}

// SetBody sets the message text.
func (m *WebSocketForm) SetBody(content string) {
	m.message.SetValue(content)
}

// LoadRequest populates from a collection request.
func (m *WebSocketForm) LoadRequest(req *collection.Request) {
	m.url.SetValue(req.URL)
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

func (m WebSocketForm) Init() tea.Cmd { return nil }

func (m WebSocketForm) Update(msg tea.Msg) (WebSocketForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Editing() {
			return m.updateEditing(msg)
		}
		return m.updateNormal(msg)
	}
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

func (m WebSocketForm) updateNormal(msg tea.KeyMsg) (WebSocketForm, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "shift+tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "enter":
		if m.focusField == 0 {
			// Connect/Disconnect
			if m.connected {
				return m, func() tea.Msg { return msgs.WSDisconnectMsg{} }
			}
			m.url.Focus()
			return m, textinput.Blink
		}
		return m.enterTabContent()
	case "h", "left":
		if m.focusField == 1 && m.activeTab > WSTabConnection {
			m.activeTab--
		}
	case "l", "right":
		if m.focusField == 1 && m.activeTab < WSTabMessages {
			m.activeTab++
		}
	case "1":
		m.activeTab = WSTabConnection
	case "2":
		m.activeTab = WSTabHeaders
	case "3":
		m.activeTab = WSTabAuth
	case "4":
		m.activeTab = WSTabMessages
	default:
		if m.focusField == 1 {
			return m.updateTabContent(msg)
		}
	}
	return m, nil
}

func (m WebSocketForm) updateEditing(msg tea.KeyMsg) (WebSocketForm, tea.Cmd) {
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
		case WSTabMessages:
			if msg.String() == "esc" {
				m.message.Blur()
				return m, nil
			}
			// Ctrl+Enter sends the message
			if msg.String() == "ctrl+enter" && m.connected {
				content := strings.TrimSpace(m.message.Value())
				if content != "" {
					m.message.SetValue("")
					return m, func() tea.Msg { return msgs.WSSendMsg{Content: content} }
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.message, cmd = m.message.Update(msg)
			return m, cmd
		case WSTabHeaders:
			if msg.String() == "esc" && !m.headers.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.headers, cmd = m.headers.Update(msg)
			return m, cmd
		case WSTabAuth:
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

func (m *WebSocketForm) enterTabContent() (WebSocketForm, tea.Cmd) {
	switch m.activeTab {
	case WSTabConnection:
		m.url.Focus()
		return *m, textinput.Blink
	case WSTabHeaders:
		var cmd tea.Cmd
		m.headers, cmd = m.headers.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case WSTabAuth:
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case WSTabMessages:
		cmd := m.message.Focus()
		return *m, cmd
	}
	return *m, nil
}

func (m WebSocketForm) updateTabContent(msg tea.Msg) (WebSocketForm, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case WSTabMessages:
		m.message, cmd = m.message.Update(msg)
	case WSTabHeaders:
		m.headers, cmd = m.headers.Update(msg)
	case WSTabAuth:
		m.auth, cmd = m.auth.Update(msg)
	}
	return m, cmd
}

func (m *WebSocketForm) syncFocus() {
	m.url.Blur()
	m.message.Blur()
}

// View renders the WebSocket form.
func (m WebSocketForm) View() string {
	var b strings.Builder

	// Connection status + URL
	statusLabel := m.styles.Hint.Render("WS ")
	if m.connected {
		statusLabel = m.styles.TabActive.Render(" CONNECTED ")
	}
	if m.focusField == 0 {
		statusLabel = m.styles.Cursor.Render(" WS ")
	}
	b.WriteString(statusLabel + " " + m.url.View())
	b.WriteString("\n\n")

	// Sub-tab bar
	var tabs []string
	for i, name := range wsSubTabNames {
		if WSSubTab(i) == m.activeTab {
			tabs = append(tabs, m.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.TabInactive.Render(name))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n\n")

	switch m.activeTab {
	case WSTabConnection:
		if m.connected {
			b.WriteString(m.styles.TabActive.Render("Connected"))
			b.WriteString("\n")
			b.WriteString(m.styles.Hint.Render("Press Enter to disconnect"))
		} else {
			b.WriteString(m.styles.Hint.Render("Press Ctrl+Enter to connect"))
		}
	case WSTabHeaders:
		b.WriteString(m.headers.View())
	case WSTabAuth:
		b.WriteString(m.auth.View())
	case WSTabMessages:
		if m.connected {
			b.WriteString(m.message.View())
			b.WriteString("\n")
			b.WriteString(m.styles.Hint.Render("Ctrl+Enter to send"))
		} else {
			b.WriteString(m.styles.Hint.Render("Connect first to send messages"))
		}
	}

	return b.String()
}
