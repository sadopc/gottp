package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// Model is the editor panel container with multi-protocol support.
type Model struct {
	httpForm    HTTPForm
	graphqlForm GraphQLForm
	wsForm      WebSocketForm
	grpcForm    GRPCForm

	protocolSelector ProtocolSelector
	protocol         string // "http", "graphql", "websocket", "grpc"
	protoFocused     bool   // whether protocol selector has focus

	focused bool
	width   int
	height  int
	styles  theme.Styles
}

// New creates a new editor panel.
func New(t theme.Theme, styles theme.Styles) Model {
	return Model{
		httpForm:         NewHTTPForm(styles),
		graphqlForm:      NewGraphQLForm(styles),
		wsForm:           NewWebSocketForm(styles),
		grpcForm:         NewGRPCForm(styles),
		protocolSelector: NewProtocolSelector(t, styles),
		protocol:         "http",
		styles:           styles,
		width:            60,
		height:           20,
	}
}

// SetFocused sets whether the editor panel is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the panel dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Account for border (2 chars each side) + protocol selector line
	innerW := w - 2
	innerH := h - 3 // extra line for protocol selector
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 5 {
		innerH = 5
	}
	m.httpForm.SetSize(innerW, innerH)
	m.graphqlForm.SetSize(innerW, innerH)
	m.wsForm.SetSize(innerW, innerH)
	m.grpcForm.SetSize(innerW, innerH)
}

// Protocol returns the current protocol.
func (m Model) Protocol() string {
	return m.protocol
}

// SetProtocol sets the active protocol.
func (m *Model) SetProtocol(proto string) {
	m.protocol = proto
	m.protocolSelector.SetProtocol(proto)
}

// Editing returns whether the editor has an active text input.
func (m Model) Editing() bool {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.Editing()
	case "websocket":
		return m.wsForm.Editing()
	case "grpc":
		return m.grpcForm.Editing()
	default:
		return m.httpForm.Editing()
	}
}

// Form returns a pointer to the HTTPForm for external access.
// Maintained for backward compatibility.
func (m *Model) Form() *HTTPForm {
	return &m.httpForm
}

// GraphQLForm returns a pointer to the GraphQL form.
func (m *Model) GQLForm() *GraphQLForm {
	return &m.graphqlForm
}

// WSForm returns a pointer to the WebSocket form.
func (m *Model) WSForm() *WebSocketForm {
	return &m.wsForm
}

// GRPCFormRef returns a pointer to the gRPC form.
func (m *Model) GRPCFormRef() *GRPCForm {
	return &m.grpcForm
}

// BuildRequest constructs a request from the active form.
func (m *Model) BuildRequest() *protocol.Request {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.BuildRequest()
	case "websocket":
		return m.wsForm.BuildRequest()
	case "grpc":
		return m.grpcForm.BuildRequest()
	default:
		return m.httpForm.BuildRequest()
	}
}

// GetParams returns params from the active form.
func (m Model) GetParams() []components.KVPair {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.GetParams()
	case "websocket":
		return m.wsForm.GetParams()
	case "grpc":
		return m.grpcForm.GetParams()
	default:
		return m.httpForm.GetParams()
	}
}

// GetHeaders returns headers from the active form.
func (m Model) GetHeaders() []components.KVPair {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.GetHeaders()
	case "websocket":
		return m.wsForm.GetHeaders()
	case "grpc":
		return m.grpcForm.GetHeaders()
	default:
		return m.httpForm.GetHeaders()
	}
}

// GetBodyContent returns body content from the active form.
func (m Model) GetBodyContent() string {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.GetBodyContent()
	case "websocket":
		return m.wsForm.GetBodyContent()
	case "grpc":
		return m.grpcForm.GetBodyContent()
	default:
		return m.httpForm.GetBodyContent()
	}
}

// SetBody sets body content on the active form.
func (m *Model) SetBody(content string) {
	switch m.protocol {
	case "graphql":
		m.graphqlForm.SetBody(content)
	case "websocket":
		m.wsForm.SetBody(content)
	case "grpc":
		m.grpcForm.SetBody(content)
	default:
		m.httpForm.SetBody(content)
	}
}

// BuildAuth returns auth config from the active form.
func (m Model) BuildAuth() *protocol.AuthConfig {
	switch m.protocol {
	case "graphql":
		return m.graphqlForm.BuildAuth()
	case "websocket":
		return m.wsForm.BuildAuth()
	case "grpc":
		return m.grpcForm.BuildAuth()
	default:
		return m.httpForm.BuildAuth()
	}
}

// FocusURL focuses the URL input on the active form.
func (m *Model) FocusURL() {
	switch m.protocol {
	case "graphql":
		m.graphqlForm.FocusURL()
	case "websocket":
		m.wsForm.FocusURL()
	case "grpc":
		m.grpcForm.FocusURL()
	default:
		m.httpForm.FocusURL()
	}
}

// LoadRequest loads a collection request into the appropriate form.
func (m *Model) LoadRequest(req *collection.Request) {
	// Detect protocol from request
	proto := "http"
	if req.GraphQL != nil {
		proto = "graphql"
	} else if req.GRPC != nil {
		proto = "grpc"
	} else if req.WebSocket != nil {
		proto = "websocket"
	}

	m.protocol = proto
	m.protocolSelector.SetProtocol(proto)

	switch proto {
	case "graphql":
		m.graphqlForm.LoadRequest(req)
	case "websocket":
		m.wsForm.LoadRequest(req)
	case "grpc":
		m.grpcForm.LoadRequest(req)
	default:
		m.httpForm.LoadRequest(req)
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.SwitchProtocolMsg:
		m.protocol = msg.Protocol
		m.protocolSelector.SetProtocol(msg.Protocol)
		return m, nil

	case tea.KeyMsg:
		// Ctrl+Enter sends the request regardless of mode
		if msg.String() == "ctrl+enter" {
			return m, func() tea.Msg {
				return msgs.SendRequestMsg{}
			}
		}

		// Protocol selector cycling (ctrl+p in normal mode)
		if msg.String() == "ctrl+p" && !m.Editing() {
			m.protocolSelector.CycleNext()
			m.protocol = m.protocolSelector.Current()
			return m, nil
		}
	}

	// Delegate to active form
	var cmd tea.Cmd
	switch m.protocol {
	case "graphql":
		m.graphqlForm, cmd = m.graphqlForm.Update(msg)
	case "websocket":
		m.wsForm, cmd = m.wsForm.Update(msg)
	case "grpc":
		m.grpcForm, cmd = m.grpcForm.Update(msg)
	default:
		m.httpForm, cmd = m.httpForm.Update(msg)
	}
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}

	// Protocol selector line
	protoView := m.protocolSelector.View(m.protoFocused)
	sendHint := m.styles.Hint.Render("ctrl+enter to send  ctrl+p protocol")

	protoLineLen := lipgloss.Width(protoView)
	hintLen := lipgloss.Width(sendHint)
	gap := innerW - protoLineLen - hintLen
	if gap < 1 {
		gap = 1
	}
	protoLine := protoView + strings.Repeat(" ", gap) + sendHint

	// Active form view
	var formView string
	switch m.protocol {
	case "graphql":
		formView = m.graphqlForm.View()
	case "websocket":
		formView = m.wsForm.View()
	case "grpc":
		formView = m.grpcForm.View()
	default:
		formView = m.httpForm.View()
	}

	content := protoLine + "\n" + formView

	// Apply border
	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = m.styles.FocusedBorder
	} else {
		borderStyle = m.styles.UnfocusedBorder
	}
	borderStyle = borderStyle.Width(m.width - 2).Height(m.height - 2)

	return borderStyle.Render(content)
}
