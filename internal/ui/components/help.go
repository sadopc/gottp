package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

type helpSection struct {
	Title    string
	Bindings []helpBinding
}

type helpBinding struct {
	Key  string
	Desc string
}

var helpSections = []helpSection{
	{
		Title: "General",
		Bindings: []helpBinding{
			{"Ctrl+C", "Quit application"},
			{"Ctrl+K", "Open command palette"},
			{"?", "Toggle this help"},
			{"Tab", "Cycle focus forward"},
			{"Shift+Tab", "Cycle focus backward"},
			{"Ctrl+Enter", "Send request"},
			{"Ctrl+N", "New request"},
			{"Ctrl+W", "Close current tab"},
			{"Ctrl+S", "Save request"},
			{"Ctrl+E", "Switch environment"},
			{"[ / ]", "Previous / next tab"},
			{"f", "Jump mode (quick navigation)"},
			{"E", "Edit body in $EDITOR"},
			{"S", "Send request (normal mode)"},
		},
	},
	{
		Title: "Sidebar",
		Bindings: []helpBinding{
			{"b", "Toggle sidebar"},
			{"j / k", "Move cursor down / up"},
			{"Enter", "Open selected request"},
			{"/", "Search collections"},
		},
	},
	{
		Title: "Editor",
		Bindings: []helpBinding{
			{"i", "Enter insert mode"},
			{"Esc", "Return to normal mode"},
			{"1-4", "Switch editor tabs (Params, Headers, Auth, Body)"},
		},
	},
	{
		Title: "Response",
		Bindings: []helpBinding{
			{"j / k", "Scroll down / up"},
			{"1-4", "Switch response tabs (Body, Headers, Cookies, Timing)"},
			{"/ / Ctrl+F", "Search in response body"},
			{"n / N", "Next / previous search match"},
			{"w", "Toggle word wrap"},
		},
	},
}

// Help is a help overlay showing keybindings.
type Help struct {
	Visible  bool
	viewport viewport.Model
	theme    theme.Theme
	styles   theme.Styles
	width    int
	height   int
	ready    bool
}

// NewHelp creates a new help overlay.
func NewHelp(t theme.Theme, s theme.Styles) Help {
	return Help{
		theme:  t,
		styles: s,
	}
}

// SetSize sets the terminal dimensions for centering.
func (m *Help) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Toggle toggles help visibility.
func (m *Help) Toggle() {
	m.Visible = !m.Visible
	if m.Visible {
		m.buildViewport()
	}
}

func (m *Help) buildViewport() {
	boxWidth := 70
	contentWidth := boxWidth - 6 // padding + border

	keyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Mauve).
		Bold(true).
		Width(16).
		Align(lipgloss.Right)

	descStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text)

	sectionStyle := lipgloss.NewStyle().
		Foreground(m.theme.Lavender).
		Bold(true).
		MarginTop(1)

	sepStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	var lines []string
	for _, section := range helpSections {
		lines = append(lines, sectionStyle.Render(section.Title))
		lines = append(lines, sepStyle.Render(strings.Repeat("─", contentWidth)))

		for _, b := range section.Bindings {
			line := keyStyle.Render(b.Key) + sepStyle.Render(" │ ") + descStyle.Render(b.Desc)
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n")

	// Set viewport height with padding for border/title
	vpHeight := m.height - 8
	if vpHeight < 10 {
		vpHeight = 10
	}

	m.viewport = viewport.New(contentWidth, vpHeight)
	m.viewport.SetContent(content)
	m.ready = true
}

// Init implements tea.Model.
func (m Help) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Help) Update(msg tea.Msg) (Help, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?":
			m.Visible = false
			return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the help overlay.
func (m Help) View() string {
	if !m.Visible {
		return ""
	}

	if !m.ready {
		m.buildViewport()
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Bold(true).
		Width(64).
		Align(lipgloss.Center)
	title := titleStyle.Render("Keyboard Shortcuts")

	content := title + "\n\n" + m.viewport.View()

	box := lipgloss.NewStyle().
		Width(70).
		Background(m.theme.Surface).
		Foreground(m.theme.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocused).
		Padding(1, 2).
		Render(content)

	return box
}
