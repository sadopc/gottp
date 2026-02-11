package response

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/theme"
)

// TimingModel displays response timing and metadata.
type TimingModel struct {
	styles      theme.Styles
	width       int
	height      int
	hasResponse bool
	content     string
}

// NewTimingModel creates a new timing display.
func NewTimingModel(s theme.Styles) TimingModel {
	return TimingModel{
		styles: s,
	}
}

// SetResponse populates timing data from a response.
func (m *TimingModel) SetResponse(resp *protocol.Response) {
	if resp == nil {
		m.hasResponse = false
		return
	}
	m.hasResponse = true

	var b strings.Builder

	row := func(label, value string) {
		fmt.Fprintf(&b, "%s  %s\n",
			m.styles.Key.Width(12).Render(label),
			m.styles.Normal.Render(value),
		)
	}

	row("Duration", resp.Duration.String())
	row("Size", formatSize(resp.Size))
	row("Protocol", resp.Proto)

	tlsStatus := "No"
	if resp.TLS {
		tlsStatus = "Yes"
	}
	row("TLS", tlsStatus)

	m.content = strings.TrimRight(b.String(), "\n")
}

// SetSize updates the dimensions.
func (m *TimingModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m TimingModel) Init() tea.Cmd {
	return nil
}

func (m TimingModel) Update(msg tea.Msg) (TimingModel, tea.Cmd) {
	return m, nil
}

func (m TimingModel) View() string {
	if !m.hasResponse {
		return m.styles.Muted.Render("No timing data")
	}
	return m.content
}

// formatSize returns a human-readable size string.
func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	case bytes < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	default:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	}
}
