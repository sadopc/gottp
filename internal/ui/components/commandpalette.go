package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// paletteCommand is a command entry in the palette.
type paletteCommand struct {
	Name     string
	Shortcut string
	Msg      tea.Msg
}

var defaultCommands = []paletteCommand{
	{Name: "Send Request", Shortcut: "Ctrl+Enter", Msg: msgs.SendRequestMsg{}},
	{Name: "New Request", Shortcut: "Ctrl+N", Msg: msgs.NewRequestMsg{}},
	{Name: "Close Tab", Shortcut: "Ctrl+W", Msg: msgs.CloseTabMsg{}},
	{Name: "Save Request", Shortcut: "Ctrl+S", Msg: msgs.SaveRequestMsg{}},
	{Name: "Switch Environment", Shortcut: "Ctrl+E", Msg: msgs.SwitchEnvMsg{}},
	{Name: "Switch Theme", Shortcut: "", Msg: msgs.SwitchThemeMsg{}},
	{Name: "Toggle Sidebar", Shortcut: "b", Msg: msgs.ToggleSidebarMsg{}},
	{Name: "Help", Shortcut: "?", Msg: msgs.ShowHelpMsg{}},
	{Name: "Copy as cURL", Shortcut: "", Msg: msgs.CopyAsCurlMsg{}},
	{Name: "Import from cURL", Shortcut: "", Msg: msgs.ImportCurlMsg{}},
	{Name: "Import from Postman", Shortcut: "", Msg: msgs.ImportFileMsg{Path: "postman"}},
	{Name: "Import from Insomnia", Shortcut: "", Msg: msgs.ImportFileMsg{Path: "insomnia"}},
	{Name: "Import from OpenAPI", Shortcut: "", Msg: msgs.ImportFileMsg{Path: "openapi"}},
	{Name: "Set Response as Baseline", Shortcut: "", Msg: msgs.SetBaselineMsg{}},
	{Name: "Clear Baseline", Shortcut: "", Msg: msgs.ClearBaselineMsg{}},
	{Name: "Edit Body in $EDITOR", Shortcut: "E", Msg: msgs.OpenEditorMsg{}},
	{Name: "Generate Code: Go", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "go"}},
	{Name: "Generate Code: Python", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "python"}},
	{Name: "Generate Code: JavaScript", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "javascript"}},
	{Name: "Generate Code: cURL", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "curl"}},
	{Name: "Generate Code: Ruby", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "ruby"}},
	{Name: "Generate Code: Java", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "java"}},
	{Name: "Generate Code: Rust", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "rust"}},
	{Name: "Generate Code: PHP", Shortcut: "", Msg: msgs.GenerateCodeMsg{Language: "php"}},
	{Name: "Template: GET JSON API", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "GET JSON API"}},
	{Name: "Template: POST JSON", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "POST JSON"}},
	{Name: "Template: PUT Update", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "PUT Update"}},
	{Name: "Template: GraphQL Query", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "GraphQL Query"}},
	{Name: "Template: OAuth2 Token", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "OAuth2 Token Request"}},
	{Name: "Template: WebSocket Echo", Shortcut: "", Msg: msgs.InsertTemplateMsg{TemplateName: "WebSocket Echo"}},
	{Name: "Quit", Shortcut: "Ctrl+C", Msg: tea.Quit()},
}

// CommandPalette is a fuzzy command palette overlay.
type CommandPalette struct {
	Visible  bool
	input    textinput.Model
	commands []paletteCommand
	filtered []paletteCommand
	cursor   int
	theme    theme.Theme
	styles   theme.Styles
}

// NewCommandPalette creates a new command palette.
func NewCommandPalette(t theme.Theme, s theme.Styles) CommandPalette {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.CharLimit = 64
	ti.Width = 54

	return CommandPalette{
		input:    ti,
		commands: defaultCommands,
		filtered: defaultCommands,
		theme:    t,
		styles:   s,
	}
}

// Open shows the command palette.
func (m *CommandPalette) Open() {
	m.Visible = true
	m.input.SetValue("")
	m.input.Focus()
	m.filtered = m.commands
	m.cursor = 0
}

// Close hides the command palette.
func (m *CommandPalette) Close() {
	m.Visible = false
	m.input.Blur()
}

// OpenEnvPicker opens the palette in environment selection mode.
func (m *CommandPalette) OpenEnvPicker(envNames []string) {
	cmds := make([]paletteCommand, len(envNames))
	for i, name := range envNames {
		cmds[i] = paletteCommand{
			Name: name,
			Msg:  msgs.SwitchEnvMsg{Name: name},
		}
	}
	m.Visible = true
	m.input.SetValue("")
	m.input.Placeholder = "Select environment..."
	m.input.Focus()
	m.commands = cmds
	m.filtered = cmds
	m.cursor = 0
}

// OpenThemePicker opens the palette in theme selection mode.
func (m *CommandPalette) OpenThemePicker(themeNames []string) {
	cmds := make([]paletteCommand, len(themeNames))
	for i, name := range themeNames {
		cmds[i] = paletteCommand{
			Name: name,
			Msg:  msgs.SwitchThemeMsg{Name: name},
		}
	}
	m.Visible = true
	m.input.SetValue("")
	m.input.Placeholder = "Select theme..."
	m.input.Focus()
	m.commands = cmds
	m.filtered = cmds
	m.cursor = 0
}

// ResetCommands restores default commands after env picker.
func (m *CommandPalette) ResetCommands() {
	m.commands = defaultCommands
	m.filtered = defaultCommands
	m.input.Placeholder = "Type a command..."
}

// Init implements tea.Model.
func (m CommandPalette) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Close()
			m.ResetCommands()
			return m, func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} }
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selected := m.filtered[m.cursor]
				m.Close()
				m.ResetCommands()
				return m, tea.Batch(
					func() tea.Msg { return msgs.SetModeMsg{Mode: msgs.ModeNormal} },
					func() tea.Msg { return selected.Msg },
				)
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Filter commands by query
	query := m.input.Value()
	if query == "" {
		m.filtered = m.commands
	} else {
		names := make([]string, len(m.commands))
		for i, c := range m.commands {
			names[i] = c.Name
		}
		matches := fuzzy.Find(query, names)
		m.filtered = make([]paletteCommand, len(matches))
		for i, match := range matches {
			m.filtered[i] = m.commands[match.Index]
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m, cmd
}

// View renders the command palette overlay.
func (m CommandPalette) View() string {
	if !m.Visible {
		return ""
	}

	boxWidth := 60

	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Bold(true).
		Width(boxWidth - 4).
		Align(lipgloss.Center)
	title := titleStyle.Render("Command Palette")

	inputView := m.input.View()

	// Build command list
	maxItems := 15
	if len(m.filtered) < maxItems {
		maxItems = len(m.filtered)
	}

	var items []string
	for i := 0; i < maxItems; i++ {
		cmd := m.filtered[i]

		nameStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
		shortcutStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

		name := cmd.Name
		shortcut := cmd.Shortcut

		nameWidth := boxWidth - 6
		if shortcut != "" {
			nameWidth -= len(shortcut) + 1
		}
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		gap := boxWidth - 6 - len(name) - len(shortcut)
		if gap < 1 {
			gap = 1
		}

		line := nameStyle.Render(name) + strings.Repeat(" ", gap) + shortcutStyle.Render(shortcut)

		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(m.theme.Overlay).
				Foreground(m.theme.Text).
				Width(boxWidth - 4).
				Render(name + strings.Repeat(" ", gap) + shortcut)
		}

		items = append(items, line)
	}

	content := title + "\n\n" + inputView + "\n\n" + strings.Join(items, "\n")

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
