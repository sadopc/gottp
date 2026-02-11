package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// Modal is a generic confirm dialog.
type Modal struct {
	Visible   bool
	Title     string
	Message   string
	onConfirm tea.Msg
	focusOK   bool
	theme     theme.Theme
	styles    theme.Styles
}

// NewModal creates a new modal dialog.
func NewModal(t theme.Theme, s theme.Styles) Modal {
	return Modal{
		theme:   t,
		styles:  s,
		focusOK: true,
	}
}

// Show displays the modal with the given title, message, and confirm action.
func (m *Modal) Show(title, message string, onConfirm tea.Msg) {
	m.Visible = true
	m.Title = title
	m.Message = message
	m.onConfirm = onConfirm
	m.focusOK = true
}

// Init implements tea.Model.
func (m Modal) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Modal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Visible = false
			return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
		case "tab", "shift+tab":
			m.focusOK = !m.focusOK
			return m, nil
		case "enter":
			m.Visible = false
			if m.focusOK && m.onConfirm != nil {
				return m, tea.Batch(
					func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} },
					func() tea.Msg { return m.onConfirm },
				)
			}
			return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
		}
	}

	return m, nil
}

// View renders the modal dialog.
func (m Modal) View() string {
	if !m.Visible {
		return ""
	}

	boxWidth := 50

	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Bold(true).
		Width(boxWidth - 4).
		Align(lipgloss.Center)

	messageStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Width(boxWidth - 4).
		Align(lipgloss.Center)

	// Buttons
	okStyle := lipgloss.NewStyle().
		Padding(0, 3)
	cancelStyle := lipgloss.NewStyle().
		Padding(0, 3)

	if m.focusOK {
		okStyle = okStyle.
			Background(m.theme.Mauve).
			Foreground(m.theme.Base).
			Bold(true)
		cancelStyle = cancelStyle.
			Background(m.theme.Surface).
			Foreground(m.theme.Subtext)
	} else {
		okStyle = okStyle.
			Background(m.theme.Surface).
			Foreground(m.theme.Subtext)
		cancelStyle = cancelStyle.
			Background(m.theme.Red).
			Foreground(m.theme.Base).
			Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		okStyle.Render("OK"),
		"  ",
		cancelStyle.Render("Cancel"),
	)

	buttonsRow := lipgloss.NewStyle().
		Width(boxWidth - 4).
		Align(lipgloss.Center).
		Render(buttons)

	content := titleStyle.Render(m.Title) + "\n\n" +
		messageStyle.Render(m.Message) + "\n\n" +
		buttonsRow

	box := lipgloss.NewStyle().
		Width(boxWidth).
		Background(m.theme.Surface).
		Foreground(m.theme.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocused).
		Padding(1, 2).
		Render(content)

	return box
}
