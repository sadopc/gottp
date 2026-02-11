package response

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sadopc/gottp/internal/protocol"
	"github.com/sadopc/gottp/internal/ui/theme"
)

// TimingModel displays response timing and metadata.
type TimingModel struct {
	th          theme.Theme
	styles      theme.Styles
	width       int
	height      int
	hasResponse bool
	content     string
}

// NewTimingModel creates a new timing display.
func NewTimingModel(th theme.Theme, s theme.Styles) TimingModel {
	return TimingModel{
		th:     th,
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

	// Add waterfall if detailed timing is available
	if resp.Timing != nil {
		b.WriteString("\n")
		b.WriteString(m.styles.Bold.Render("Waterfall"))
		b.WriteString("\n\n")
		b.WriteString(m.renderWaterfall(resp.Timing))
	}

	m.content = strings.TrimRight(b.String(), "\n")
}

// renderWaterfall renders a horizontal bar chart waterfall of timing phases.
func (m *TimingModel) renderWaterfall(td *protocol.TimingDetail) string {
	type phase struct {
		label    string
		duration time.Duration
		color    lipgloss.Color
	}

	phases := []phase{
		{"DNS Lookup", td.DNSLookup, m.th.Sky},
		{"TCP Connect", td.TCPConnect, m.th.Green},
		{"TLS Handshake", td.TLSHandshake, m.th.Mauve},
		{"Server (TTFB)", td.TTFB, m.th.Yellow},
		{"Transfer", td.Transfer, m.th.Peach},
	}

	// Calculate the max bar width available
	labelWidth := 15
	durationWidth := 10
	padding := 4 // spaces between columns
	barMaxWidth := m.width - labelWidth - durationWidth - padding
	if barMaxWidth < 10 {
		barMaxWidth = 10
	}
	if barMaxWidth > 60 {
		barMaxWidth = 60
	}

	// Find the total for proportional bars
	total := td.Total
	if total == 0 {
		total = 1 // avoid division by zero
	}

	// Build cumulative offsets for waterfall positioning
	type barSegment struct {
		offset int
		width  int
	}

	var segments []barSegment
	cumulative := time.Duration(0)
	for _, p := range phases {
		offsetFrac := float64(cumulative) / float64(total)
		widthFrac := float64(p.duration) / float64(total)

		offset := int(math.Round(offsetFrac * float64(barMaxWidth)))
		w := int(math.Round(widthFrac * float64(barMaxWidth)))
		if p.duration > 0 && w < 1 {
			w = 1
		}

		segments = append(segments, barSegment{offset: offset, width: w})
		cumulative += p.duration
	}

	var b strings.Builder

	for i, p := range phases {
		label := m.styles.Key.Width(labelWidth).Render(p.label)

		// Build the bar line
		seg := segments[i]
		bar := strings.Repeat(" ", seg.offset)
		barStyle := lipgloss.NewStyle().Foreground(p.color)
		if seg.width > 0 {
			bar += barStyle.Render(strings.Repeat("█", seg.width))
		}
		// Pad to full bar width
		visLen := seg.offset + seg.width
		if visLen < barMaxWidth {
			bar += strings.Repeat(" ", barMaxWidth-visLen)
		}

		dur := m.styles.Muted.Width(durationWidth).Align(lipgloss.Right).Render(formatDuration(p.duration))

		fmt.Fprintf(&b, "%s %s %s\n", label, bar, dur)
	}

	// Total row
	totalLabel := m.styles.Bold.Width(labelWidth).Render("Total")
	totalBar := strings.Repeat(" ", barMaxWidth)
	totalDur := m.styles.Normal.Width(durationWidth).Align(lipgloss.Right).Bold(true).Render(formatDuration(td.Total))
	fmt.Fprintf(&b, "%s %s %s", totalLabel, totalBar, totalDur)

	return b.String()
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

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
