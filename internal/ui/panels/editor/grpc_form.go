package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// GRPCSubTab identifies the active sub-tab in the gRPC form.
type GRPCSubTab int

const (
	GRPCTabService GRPCSubTab = iota
	GRPCTabRequest
	GRPCTabMetadata
	GRPCTabAuth
)

var grpcSubTabNames = []string{"Service", "Request", "Metadata", "Auth"}

// GRPCForm is the gRPC request form component.
type GRPCForm struct {
	server   textinput.Model
	service  string
	method   string
	body     textarea.Model
	metadata components.KVTable
	auth     AuthSection

	services []msgs.GRPCServiceInfo
	svcIdx   int
	mtdIdx   int

	activeTab  GRPCSubTab
	focusField int // 0=server, 1=sub-tab content

	width  int
	height int
	styles theme.Styles
}

// NewGRPCForm creates a new gRPC form.
func NewGRPCForm(styles theme.Styles) GRPCForm {
	serverInput := textinput.New()
	serverInput.Placeholder = "localhost:50051"
	serverInput.CharLimit = 512
	serverInput.Width = 40

	bodyArea := textarea.New()
	bodyArea.Placeholder = `{"field": "value"}`
	bodyArea.ShowLineNumbers = false
	bodyArea.CharLimit = 0
	bodyArea.SetWidth(40)
	bodyArea.SetHeight(6)

	return GRPCForm{
		server:    serverInput,
		body:      bodyArea,
		metadata:  components.NewKVTable(styles),
		auth:      NewAuthSection(styles),
		activeTab: GRPCTabService,
		styles:    styles,
		width:     60,
		height:    20,
	}
}

// SetSize updates the form dimensions.
func (m *GRPCForm) SetSize(w, h int) {
	m.width = w
	m.height = h
	svrW := w - 4
	if svrW < 10 {
		svrW = 10
	}
	m.server.Width = svrW
	contentW := w - 2
	if contentW < 10 {
		contentW = 10
	}
	m.metadata.SetSize(contentW)
	m.auth.SetSize(contentW)
	bodyH := h - 6
	if bodyH < 3 {
		bodyH = 3
	}
	m.body.SetWidth(contentW)
	m.body.SetHeight(bodyH)
}

// FocusURL focuses the server address input.
func (m *GRPCForm) FocusURL() {
	m.focusField = 0
	m.server.Focus()
	m.server.CursorEnd()
}

// SetServices populates discovered services.
func (m *GRPCForm) SetServices(services []msgs.GRPCServiceInfo) {
	m.services = services
	if len(services) > 0 {
		m.svcIdx = 0
		m.service = services[0].Name
		if len(services[0].Methods) > 0 {
			m.mtdIdx = 0
			m.method = services[0].Methods[0].FullName
		}
	}
}

// Editing returns whether any input is in editing mode.
func (m GRPCForm) Editing() bool {
	if m.focusField == 0 && m.server.Focused() {
		return true
	}
	if m.focusField == 1 {
		switch m.activeTab {
		case GRPCTabRequest:
			return m.body.Focused()
		case GRPCTabMetadata:
			return m.metadata.Editing()
		case GRPCTabAuth:
			return m.auth.Editing()
		}
	}
	return false
}

// BuildRequest constructs a protocol.Request from the gRPC form.
func (m GRPCForm) BuildRequest() *protocol.Request {
	req := &protocol.Request{
		Protocol:    "grpc",
		Method:      "POST",
		URL:         m.server.Value(),
		Headers:     make(map[string]string),
		GRPCService: m.service,
		GRPCMethod:  m.method,
		Metadata:    make(map[string]string),
	}

	body := strings.TrimSpace(m.body.Value())
	if body != "" {
		req.Body = []byte(body)
	}

	for _, h := range m.metadata.GetPairs() {
		if h.Enabled && h.Key != "" {
			req.Metadata[h.Key] = h.Value
		}
	}

	req.Auth = m.auth.BuildAuth()
	return req
}

// BuildAuth returns the auth config.
func (m GRPCForm) BuildAuth() *protocol.AuthConfig {
	return m.auth.BuildAuth()
}

// GetHeaders returns empty headers (gRPC uses metadata).
func (m GRPCForm) GetHeaders() []components.KVPair {
	return m.metadata.GetPairs()
}

// GetParams returns empty params.
func (m GRPCForm) GetParams() []components.KVPair {
	return nil
}

// GetBodyContent returns the request body JSON.
func (m GRPCForm) GetBodyContent() string {
	return strings.TrimSpace(m.body.Value())
}

// SetBody sets the request body.
func (m *GRPCForm) SetBody(content string) {
	m.body.SetValue(content)
}

// LoadRequest populates from a collection request.
func (m *GRPCForm) LoadRequest(req *collection.Request) {
	m.server.SetValue(req.URL)
	if req.GRPC != nil {
		m.service = req.GRPC.Service
		m.method = req.GRPC.Method
	}
	if req.Body != nil {
		m.body.SetValue(req.Body.Content)
	}
	m.auth.LoadAuth(req.Auth)
	m.focusField = 0
}

func (m GRPCForm) Init() tea.Cmd { return nil }

func (m GRPCForm) Update(msg tea.Msg) (GRPCForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Editing() {
			return m.updateEditing(msg)
		}
		return m.updateNormal(msg)
	}
	if m.focusField == 0 {
		var cmd tea.Cmd
		m.server, cmd = m.server.Update(msg)
		return m, cmd
	}
	if m.focusField == 1 {
		return m.updateTabContent(msg)
	}
	return m, nil
}

func (m GRPCForm) updateNormal(msg tea.KeyMsg) (GRPCForm, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "shift+tab":
		m.focusField = (m.focusField + 1) % 2
		m.syncFocus()
	case "ctrl+i":
		// Trigger reflection
		return m, func() tea.Msg { return msgs.GRPCReflectMsg{} }
	case "enter":
		if m.focusField == 0 {
			m.server.Focus()
			return m, textinput.Blink
		}
		return m.enterTabContent()
	case "h", "left":
		if m.focusField == 1 && m.activeTab > GRPCTabService {
			m.activeTab--
		}
	case "l", "right":
		if m.focusField == 1 && m.activeTab < GRPCTabAuth {
			m.activeTab++
		}
	case "j", "down":
		if m.focusField == 1 && m.activeTab == GRPCTabService {
			m.cycleService(1)
		}
	case "k", "up":
		if m.focusField == 1 && m.activeTab == GRPCTabService {
			m.cycleService(-1)
		}
	case "1":
		m.activeTab = GRPCTabService
	case "2":
		m.activeTab = GRPCTabRequest
	case "3":
		m.activeTab = GRPCTabMetadata
	case "4":
		m.activeTab = GRPCTabAuth
	default:
		if m.focusField == 1 {
			return m.updateTabContent(msg)
		}
	}
	return m, nil
}

func (m *GRPCForm) cycleService(dir int) {
	if len(m.services) == 0 {
		return
	}
	// Cycle through methods in current service, then move to next service
	svc := m.services[m.svcIdx]
	m.mtdIdx += dir
	if m.mtdIdx >= len(svc.Methods) {
		m.svcIdx = (m.svcIdx + 1) % len(m.services)
		m.mtdIdx = 0
	} else if m.mtdIdx < 0 {
		m.svcIdx = (m.svcIdx - 1 + len(m.services)) % len(m.services)
		m.mtdIdx = len(m.services[m.svcIdx].Methods) - 1
		if m.mtdIdx < 0 {
			m.mtdIdx = 0
		}
	}
	svc = m.services[m.svcIdx]
	m.service = svc.Name
	if len(svc.Methods) > 0 && m.mtdIdx < len(svc.Methods) {
		m.method = svc.Methods[m.mtdIdx].FullName
	}
}

func (m GRPCForm) updateEditing(msg tea.KeyMsg) (GRPCForm, tea.Cmd) {
	if m.focusField == 0 {
		if msg.String() == "esc" {
			m.server.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.server, cmd = m.server.Update(msg)
		return m, cmd
	}
	if m.focusField == 1 {
		switch m.activeTab {
		case GRPCTabRequest:
			if msg.String() == "esc" {
				m.body.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.body, cmd = m.body.Update(msg)
			return m, cmd
		case GRPCTabMetadata:
			if msg.String() == "esc" && !m.metadata.Editing() {
				return m, nil
			}
			var cmd tea.Cmd
			m.metadata, cmd = m.metadata.Update(msg)
			return m, cmd
		case GRPCTabAuth:
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

func (m *GRPCForm) enterTabContent() (GRPCForm, tea.Cmd) {
	switch m.activeTab {
	case GRPCTabRequest:
		cmd := m.body.Focus()
		return *m, cmd
	case GRPCTabMetadata:
		var cmd tea.Cmd
		m.metadata, cmd = m.metadata.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	case GRPCTabAuth:
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *m, cmd
	}
	return *m, nil
}

func (m GRPCForm) updateTabContent(msg tea.Msg) (GRPCForm, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case GRPCTabRequest:
		m.body, cmd = m.body.Update(msg)
	case GRPCTabMetadata:
		m.metadata, cmd = m.metadata.Update(msg)
	case GRPCTabAuth:
		m.auth, cmd = m.auth.Update(msg)
	}
	return m, cmd
}

func (m *GRPCForm) syncFocus() {
	m.server.Blur()
	m.body.Blur()
}

// View renders the gRPC form.
func (m GRPCForm) View() string {
	var b strings.Builder

	// Server address
	label := m.styles.Hint.Render("gRPC")
	if m.focusField == 0 {
		label = m.styles.Cursor.Render(" gRPC ")
	}
	b.WriteString(label + " " + m.server.View())
	b.WriteString("\n\n")

	// Sub-tab bar
	var tabs []string
	for i, name := range grpcSubTabNames {
		if GRPCSubTab(i) == m.activeTab {
			tabs = append(tabs, m.styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.TabInactive.Render(name))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n\n")

	switch m.activeTab {
	case GRPCTabService:
		if len(m.services) == 0 {
			b.WriteString(m.styles.Hint.Render("No services discovered. Press Ctrl+I to run reflection."))
		} else {
			for si, svc := range m.services {
				svcStyle := m.styles.Muted
				if si == m.svcIdx {
					svcStyle = m.styles.TabActive
				}
				b.WriteString(svcStyle.Render(svc.Name) + "\n")
				for mi, mtd := range svc.Methods {
					prefix := "  "
					mtdStyle := m.styles.Muted
					if si == m.svcIdx && mi == m.mtdIdx {
						prefix = "> "
						mtdStyle = m.styles.Cursor
					}
					label := mtd.Name
					if tag := streamingTag(mtd); tag != "" {
						label += " " + tag
					}
					b.WriteString(prefix + mtdStyle.Render(label) + "\n")
				}
			}
		}
	case GRPCTabRequest:
		if m.method != "" {
			methodLabel := "Method: " + m.method
			if tag := m.selectedStreamingTag(); tag != "" {
				methodLabel += " [" + tag + "]"
			}
			b.WriteString(m.styles.Hint.Render(methodLabel) + "\n\n")
		}
		b.WriteString(m.body.View())
	case GRPCTabMetadata:
		b.WriteString(m.metadata.View())
	case GRPCTabAuth:
		b.WriteString(m.auth.View())
	}

	return b.String()
}

// streamingTag returns a short label describing the streaming mode of a gRPC
// method, or an empty string for unary methods.
func streamingTag(mtd msgs.GRPCMethodInfo) string {
	if mtd.IsClientStream && mtd.IsServerStream {
		return "[Bidirectional]"
	}
	if mtd.IsServerStream {
		return "[Server Streaming]"
	}
	if mtd.IsClientStream {
		return "[Client Streaming]"
	}
	return ""
}

// selectedStreamingTag returns the streaming tag for the currently selected method.
func (m GRPCForm) selectedStreamingTag() string {
	if len(m.services) == 0 || m.svcIdx >= len(m.services) {
		return ""
	}
	svc := m.services[m.svcIdx]
	if m.mtdIdx >= len(svc.Methods) {
		return ""
	}
	mtd := svc.Methods[m.mtdIdx]
	if mtd.IsClientStream && mtd.IsServerStream {
		return "Bidirectional"
	}
	if mtd.IsServerStream {
		return "Server Streaming"
	}
	if mtd.IsClientStream {
		return "Client Streaming"
	}
	return ""
}
