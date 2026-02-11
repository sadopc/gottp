package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// TabItem represents a single tab.
type TabItem struct {
	Name   string
	Method string
}

// TabBar is a horizontal tab bar for open requests.
type TabBar struct {
	tabs   []TabItem
	active int
	width  int
	theme  theme.Theme
	styles theme.Styles
}

// NewTabBar creates a new tab bar.
func NewTabBar(t theme.Theme, s theme.Styles) TabBar {
	return TabBar{
		theme:  t,
		styles: s,
	}
}

// SetTabs sets the tab items.
func (m *TabBar) SetTabs(tabs []TabItem) {
	m.tabs = tabs
	if m.active >= len(tabs) && len(tabs) > 0 {
		m.active = len(tabs) - 1
	}
}

// SetActive sets the active tab index.
func (m *TabBar) SetActive(index int) {
	if index >= 0 && index < len(m.tabs) {
		m.active = index
	}
}

// SetWidth sets the available width.
func (m *TabBar) SetWidth(w int) {
	m.width = w
}

// Init implements tea.Model.
func (m TabBar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m TabBar) Update(msg tea.Msg) (TabBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("["))):
			return m, func() tea.Msg { return msgs.PrevTabMsg{} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("]"))):
			return m, func() tea.Msg { return msgs.NextTabMsg{} }
		}
	}
	return m, nil
}

// View renders the tab bar.
func (m TabBar) View() string {
	if len(m.tabs) == 0 {
		return ""
	}

	sep := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("│")

	// Calculate available width for tabs
	// Reserve space for separators, [+] button, and surrounding space
	plusBtn := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(" [+]")
	separatorCount := len(m.tabs) // separators between tabs + before [+]
	reservedWidth := lipgloss.Width(plusBtn) + separatorCount

	availableForTabs := m.width - reservedWidth
	if availableForTabs < 0 {
		availableForTabs = 0
	}

	// Each tab gets roughly equal share of available space
	maxTabWidth := 30
	if len(m.tabs) > 0 {
		perTab := availableForTabs / len(m.tabs)
		if perTab < maxTabWidth {
			maxTabWidth = perTab
		}
	}
	if maxTabWidth < 8 {
		maxTabWidth = 8
	}

	var parts []string
	for i, tab := range m.tabs {
		// Build method badge (3 chars)
		method := tab.Method
		if len(method) > 3 {
			method = method[:3]
		}
		for len(method) < 3 {
			method += " "
		}
		badge := lipgloss.NewStyle().
			Foreground(m.theme.MethodColor(tab.Method)).
			Bold(true).
			Render(method)

		// Truncate name to fit
		nameWidth := maxTabWidth - 4 // 3 for method + 1 space
		if nameWidth < 1 {
			nameWidth = 1
		}
		name := tab.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		label := badge + " " + name

		var rendered string
		if i == m.active {
			rendered = m.styles.TabActive.Render(label)
		} else {
			rendered = m.styles.TabInactive.Render(label)
		}
		parts = append(parts, rendered)
	}

	result := strings.Join(parts, sep) + sep + plusBtn

	// Pad or truncate to width
	rendered := result
	renderedWidth := lipgloss.Width(rendered)
	if renderedWidth < m.width {
		rendered += strings.Repeat(" ", m.width-renderedWidth)
	}

	return rendered
}
