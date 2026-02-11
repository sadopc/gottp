package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// JumpTarget represents a focusable target in the UI.
type JumpTarget struct {
	Label  string        // displayed label (a-z, aa-az, ...)
	Name   string        // human-readable name
	Panel  msgs.PanelFocus
	Action tea.Msg       // message to emit when selected
}

// JumpOverlay manages jump-to-target navigation.
type JumpOverlay struct {
	Visible  bool
	targets  []JumpTarget
	typed    string
	theme    theme.Theme
	styles   theme.Styles
}

// NewJumpOverlay creates a new jump overlay.
func NewJumpOverlay(t theme.Theme, s theme.Styles) JumpOverlay {
	return JumpOverlay{
		theme:  t,
		styles: s,
	}
}

// Open activates jump mode with the given targets.
func (m *JumpOverlay) Open(targets []JumpTarget) {
	// Generate labels: a-z, then aa-az
	labeled := make([]JumpTarget, len(targets))
	for i, t := range targets {
		t.Label = generateLabel(i)
		labeled[i] = t
	}
	m.targets = labeled
	m.typed = ""
	m.Visible = true
}

// Close hides the jump overlay.
func (m *JumpOverlay) Close() {
	m.Visible = false
	m.targets = nil
	m.typed = ""
}

// Update handles key input during jump mode.
func (m JumpOverlay) Update(msg tea.Msg) (JumpOverlay, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Close()
			return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
		default:
			ch := msg.String()
			if len(ch) == 1 && ch[0] >= 'a' && ch[0] <= 'z' {
				m.typed += ch
				// Check for unique match
				var matching []JumpTarget
				for _, t := range m.targets {
					if strings.HasPrefix(t.Label, m.typed) {
						matching = append(matching, t)
					}
				}
				if len(matching) == 1 {
					// Unique match found
					target := matching[0]
					m.Close()
					return m, tea.Batch(
						func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} },
						func() tea.Msg { return target.Action },
					)
				}
				if len(matching) == 0 {
					// No match, close
					m.Close()
					return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
				}
			}
		}
	}
	return m, nil
}

// View renders the jump target labels as an overlay hint.
func (m JumpOverlay) View() string {
	if !m.Visible || len(m.targets) == 0 {
		return ""
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Base).
		Background(m.theme.Yellow).
		Bold(true).
		Padding(0, 1)

	nameStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text)

	var lines []string
	for _, t := range m.targets {
		if m.typed != "" && !strings.HasPrefix(t.Label, m.typed) {
			continue
		}
		label := labelStyle.Render(t.Label)
		name := nameStyle.Render(t.Name)
		lines = append(lines, "  "+label+" "+name)
	}

	if len(lines) == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Bold(true).
		Render("Jump to:")

	content := title + "\n\n" + strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(m.theme.Surface).
		Foreground(m.theme.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Yellow).
		Padding(1, 2).
		Render(content)

	return box
}

// generateLabel creates a label for the given index: a-z, then aa-az, ba-bz, etc.
func generateLabel(idx int) string {
	if idx < 26 {
		return string(rune('a' + idx))
	}
	idx -= 26
	first := idx / 26
	second := idx % 26
	return string(rune('a'+first)) + string(rune('a'+second))
}
