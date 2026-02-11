package editor

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gottp/internal/ui/theme"
)

var protocols = []string{"HTTP", "GraphQL", "WebSocket", "gRPC"}

// ProtocolSelector is a cycling widget for protocol selection.
type ProtocolSelector struct {
	index  int
	styles theme.Styles
	th     theme.Theme
}

// NewProtocolSelector creates a new protocol selector.
func NewProtocolSelector(t theme.Theme, styles theme.Styles) ProtocolSelector {
	return ProtocolSelector{styles: styles, th: t}
}

// Current returns the current protocol name (lowercase).
func (p ProtocolSelector) Current() string {
	switch protocols[p.index] {
	case "HTTP":
		return "http"
	case "GraphQL":
		return "graphql"
	case "WebSocket":
		return "websocket"
	case "gRPC":
		return "grpc"
	}
	return "http"
}

// Label returns the display label.
func (p ProtocolSelector) Label() string {
	return protocols[p.index]
}

// CycleNext cycles to the next protocol.
func (p *ProtocolSelector) CycleNext() {
	p.index = (p.index + 1) % len(protocols)
}

// CyclePrev cycles to the previous protocol.
func (p *ProtocolSelector) CyclePrev() {
	p.index = (p.index - 1 + len(protocols)) % len(protocols)
}

// SetProtocol sets the protocol by name.
func (p *ProtocolSelector) SetProtocol(name string) {
	for i, proto := range protocols {
		if proto == name || (proto == "HTTP" && name == "http") ||
			(proto == "GraphQL" && name == "graphql") ||
			(proto == "WebSocket" && name == "websocket") ||
			(proto == "gRPC" && name == "grpc") {
			p.index = i
			return
		}
	}
}

// View renders the protocol selector.
func (p ProtocolSelector) View(focused bool) string {
	label := p.Label()
	if focused {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(p.th.Surface).
			Background(p.th.Mauve).
			Padding(0, 1).
			Render(label)
	}
	return lipgloss.NewStyle().
		Foreground(p.th.Mauve).
		Bold(true).
		Padding(0, 1).
		Render(label)
}
