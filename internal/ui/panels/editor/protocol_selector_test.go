package editor

import (
	"strings"
	"testing"

	"github.com/serdar/gottp/internal/ui/theme"
)

func TestProtocolSelector_CycleAndSet(t *testing.T) {
	th := theme.Resolve("catppuccin-mocha")
	styles := theme.NewStyles(th)
	p := NewProtocolSelector(th, styles)

	if got := p.Current(); got != "http" {
		t.Fatalf("default protocol = %q, want http", got)
	}
	if got := p.Label(); got != "HTTP" {
		t.Fatalf("default label = %q, want HTTP", got)
	}

	p.CycleNext()
	if got := p.Current(); got != "graphql" {
		t.Fatalf("after CycleNext current = %q, want graphql", got)
	}

	p.CyclePrev()
	if got := p.Current(); got != "http" {
		t.Fatalf("after CyclePrev current = %q, want http", got)
	}

	p.SetProtocol("websocket")
	if got := p.Current(); got != "websocket" {
		t.Fatalf("SetProtocol(websocket) current = %q", got)
	}

	p.SetProtocol("gRPC")
	if got := p.Current(); got != "grpc" {
		t.Fatalf("SetProtocol(gRPC) current = %q", got)
	}

	before := p.Current()
	p.SetProtocol("unknown")
	if got := p.Current(); got != before {
		t.Fatalf("unknown protocol should keep previous value, got %q want %q", got, before)
	}
}

func TestProtocolSelector_View(t *testing.T) {
	th := theme.Resolve("catppuccin-mocha")
	styles := theme.NewStyles(th)
	p := NewProtocolSelector(th, styles)
	p.SetProtocol("graphql")

	focused := p.View(true)
	if !strings.Contains(focused, "GraphQL") {
		t.Fatalf("focused view missing label: %q", focused)
	}

	unfocused := p.View(false)
	if !strings.Contains(unfocused, "GraphQL") {
		t.Fatalf("unfocused view missing label: %q", unfocused)
	}
}
