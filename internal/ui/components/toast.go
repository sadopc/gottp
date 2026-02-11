package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gottp/internal/ui/theme"
)

// toastDismissMsg dismisses the toast.
type toastDismissMsg struct{}

// Toast is an auto-dismiss notification.
type Toast struct {
	Visible  bool
	text     string
	isError  bool
	duration time.Duration
	theme    theme.Theme
	styles   theme.Styles
}

// NewToast creates a new toast component.
func NewToast(t theme.Theme, s theme.Styles) Toast {
	return Toast{
		theme:    t,
		styles:   s,
		duration: 3 * time.Second,
	}
}

// Show displays a toast message and returns a Cmd for auto-dismiss.
func (m *Toast) Show(text string, isError bool, duration time.Duration) tea.Cmd {
	m.Visible = true
	m.text = text
	m.isError = isError
	if duration > 0 {
		m.duration = duration
	} else {
		m.duration = 3 * time.Second
	}
	return tea.Tick(m.duration, func(time.Time) tea.Msg {
		return toastDismissMsg{}
	})
}

// Init implements tea.Model.
func (m Toast) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Toast) Update(msg tea.Msg) (Toast, tea.Cmd) {
	switch msg.(type) {
	case toastDismissMsg:
		m.Visible = false
		m.text = ""
	}
	return m, nil
}

// View renders the toast notification.
func (m Toast) View() string {
	if !m.Visible || m.text == "" {
		return ""
	}

	fg := m.theme.Green
	if m.isError {
		fg = m.theme.Red
	}

	style := lipgloss.NewStyle().
		Foreground(fg).
		Background(m.theme.Surface).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(fg)

	return style.Render(m.text)
}
