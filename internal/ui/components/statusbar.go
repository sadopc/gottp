package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/sadopc/gottp/internal/ui/msgs"
	"github.com/sadopc/gottp/internal/ui/theme"
)

// clearStatusMsg clears a temporary status message.
type clearStatusMsg struct{}

// StatusBar is a full-width bottom status bar.
type StatusBar struct {
	statusCode  int
	duration    time.Duration
	size        int64
	contentType string
	mode        msgs.AppMode
	message     string
	envName     string
	width       int
	theme       theme.Theme
	styles      theme.Styles
}

// NewStatusBar creates a new status bar.
func NewStatusBar(t theme.Theme, s theme.Styles) StatusBar {
	return StatusBar{
		theme:  t,
		styles: s,
		mode:   msgs.ModeNormal,
	}
}

// SetStatus sets the response status info.
func (m *StatusBar) SetStatus(code int, duration time.Duration, size int64, contentType string) {
	m.statusCode = code
	m.duration = duration
	m.size = size
	m.contentType = contentType
}

// SetMode sets the current app mode.
func (m *StatusBar) SetMode(mode msgs.AppMode) {
	m.mode = mode
}

// SetWidth sets the available width.
func (m *StatusBar) SetWidth(w int) {
	m.width = w
}

// SetMessage sets a temporary status message.
func (m *StatusBar) SetMessage(text string) {
	m.message = text
}

// SetEnv sets the active environment name displayed on the right.
func (m *StatusBar) SetEnv(name string) {
	m.envName = name
}

// Init implements tea.Model.
func (m StatusBar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	switch msg.(type) {
	case clearStatusMsg:
		m.message = ""
	}
	return m, nil
}

// View renders the status bar.
func (m StatusBar) View() string {
	barStyle := lipgloss.NewStyle().
		Background(m.theme.Surface).
		Foreground(m.theme.Text).
		Width(m.width)

	// Left section: status code, duration, size, content-type
	var leftParts []string

	if m.message != "" {
		leftParts = append(leftParts, lipgloss.NewStyle().
			Foreground(m.theme.Text).
			Background(m.theme.Surface).
			Render(m.message))
	} else {
		if m.statusCode > 0 {
			statusColor := m.theme.StatusColor(m.statusCode)
			codeStr := lipgloss.NewStyle().
				Foreground(statusColor).
				Background(m.theme.Surface).
				Bold(true).
				Render(fmt.Sprintf("%d", m.statusCode))
			leftParts = append(leftParts, codeStr)
		}

		if m.duration > 0 {
			dur := lipgloss.NewStyle().
				Foreground(m.theme.Subtext).
				Background(m.theme.Surface).
				Render(formatDuration(m.duration))
			leftParts = append(leftParts, dur)
		}

		if m.size > 0 {
			sz := lipgloss.NewStyle().
				Foreground(m.theme.Subtext).
				Background(m.theme.Surface).
				Render(humanize.IBytes(uint64(m.size)))
			leftParts = append(leftParts, sz)
		}

		if m.contentType != "" {
			ct := lipgloss.NewStyle().
				Foreground(m.theme.Muted).
				Background(m.theme.Surface).
				Render(m.contentType)
			leftParts = append(leftParts, ct)
		}
	}

	left := strings.Join(leftParts, " │ ")

	// Center: mode indicator
	modeStr := lipgloss.NewStyle().
		Foreground(m.theme.Mauve).
		Background(m.theme.Surface).
		Bold(true).
		Render("[" + m.mode.String() + "]")

	// Right: env + hints
	var rightParts []string
	if m.envName != "" {
		envStr := lipgloss.NewStyle().
			Foreground(m.theme.Teal).
			Background(m.theme.Surface).
			Bold(true).
			Render("[" + m.envName + "]")
		rightParts = append(rightParts, envStr)
	}
	rightParts = append(rightParts, lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Background(m.theme.Surface).
		Render("?:help  Ctrl+K:command"))
	hint := strings.Join(rightParts, " ")

	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(modeStr)
	rightWidth := lipgloss.Width(hint)

	// Calculate gaps
	totalContent := leftWidth + centerWidth + rightWidth
	if totalContent >= m.width {
		// Tight: just left + mode + hint
		gap1 := m.width - totalContent
		if gap1 < 1 {
			gap1 = 1
		}
		line := " " + left + strings.Repeat(" ", gap1) + modeStr + " " + hint
		return barStyle.Render(line)
	}

	remaining := m.width - totalContent - 2 // padding
	gap1 := remaining / 2
	gap2 := remaining - gap1

	line := " " + left +
		strings.Repeat(" ", gap1) + modeStr +
		strings.Repeat(" ", gap2) + hint

	return barStyle.Render(line)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
