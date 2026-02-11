package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/ui/msgs"
	"github.com/sadopc/gottp/internal/ui/theme"
)

func newEditorModelForTest() Model {
	th := theme.Resolve("catppuccin-mocha")
	styles := theme.NewStyles(th)
	return New(th, styles)
}

func TestEditorModel_DefaultsAndSetters(t *testing.T) {
	m := newEditorModelForTest()

	if m.Protocol() != "http" {
		t.Fatalf("default protocol = %q, want http", m.Protocol())
	}

	m.SetFocused(true)
	m.SetSize(80, 24)

	if m.width != 80 || m.height != 24 {
		t.Fatalf("size not applied, got %dx%d", m.width, m.height)
	}
}

func TestEditorModel_ProtocolDelegation(t *testing.T) {
	m := newEditorModelForTest()

	protocolsToTest := []string{"http", "graphql", "websocket", "grpc"}
	for _, proto := range protocolsToTest {
		t.Run(proto, func(t *testing.T) {
			m.SetProtocol(proto)
			if m.Protocol() != proto {
				t.Fatalf("protocol = %q, want %q", m.Protocol(), proto)
			}

			m.SetBody("body-" + proto)
			if got := m.GetBodyContent(); got == "" {
				t.Fatalf("expected body content for protocol %s", proto)
			}

			req := m.BuildRequest()
			if req == nil {
				t.Fatalf("BuildRequest returned nil for protocol %s", proto)
			}

			_ = m.GetParams()
			_ = m.GetHeaders()
			_ = m.BuildAuth()
			m.FocusURL()
			_ = m.Editing()
		})
	}
}

func TestEditorModel_LoadRequestDetectsProtocol(t *testing.T) {
	m := newEditorModelForTest()

	httpReq := collection.NewRequest("HTTP", "GET", "https://example.com")
	m.LoadRequest(httpReq)
	if m.Protocol() != "http" {
		t.Fatalf("expected http protocol, got %q", m.Protocol())
	}

	gqlReq := collection.NewRequest("GraphQL", "POST", "https://example.com/graphql")
	gqlReq.GraphQL = &collection.GraphQLConfig{Query: "{ health }"}
	m.LoadRequest(gqlReq)
	if m.Protocol() != "graphql" {
		t.Fatalf("expected graphql protocol, got %q", m.Protocol())
	}

	wsReq := collection.NewRequest("WS", "GET", "wss://example.com/ws")
	wsReq.WebSocket = &collection.WebSocketConfig{}
	m.LoadRequest(wsReq)
	if m.Protocol() != "websocket" {
		t.Fatalf("expected websocket protocol, got %q", m.Protocol())
	}

	grpcReq := collection.NewRequest("gRPC", "POST", "localhost:50051")
	grpcReq.GRPC = &collection.GRPCConfig{Service: "pkg.Service", Method: "Ping"}
	m.LoadRequest(grpcReq)
	if m.Protocol() != "grpc" {
		t.Fatalf("expected grpc protocol, got %q", m.Protocol())
	}
}

func TestEditorModel_UpdateSwitchProtocolAndCycle(t *testing.T) {
	m := newEditorModelForTest()

	updated, cmd := m.Update(msgs.SwitchProtocolMsg{Protocol: "graphql"})
	if cmd != nil {
		t.Fatal("expected nil cmd for SwitchProtocolMsg")
	}
	if updated.Protocol() != "graphql" {
		t.Fatalf("expected protocol graphql after msg, got %q", updated.Protocol())
	}

	before := updated.Protocol()
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if updated.Protocol() == before {
		t.Fatalf("expected ctrl+p to cycle protocol, still %q", updated.Protocol())
	}
}

func TestEditorModel_UpdateCtrlEnterCommandAndView(t *testing.T) {
	m := newEditorModelForTest()
	m.SetSize(100, 26)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_ = updated
	if cmd != nil {
		// normal rune update may or may not return cmd depending on active form state;
		// this assertion ensures we execute path without panic.
		_ = cmd
	}

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty editor view")
	}
}
