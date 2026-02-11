package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/ui/theme"
)

// Model is the sidebar panel showing collections and history.
type Model struct {
	items    []collection.FlatItem
	filtered []int // indices into items that match the filter
	cursor   int   // index into filtered

	width   int
	height  int
	focused bool

	filtering   bool
	filterInput textinput.Model

	theme  theme.Theme
	styles theme.Styles
}

// New creates a new sidebar model.
func New(t theme.Theme, s theme.Styles) Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.CharLimit = 128

	return Model{
		theme:       t,
		styles:      s,
		filterInput: ti,
	}
}

// SetItems replaces the displayed items.
func (m *Model) SetItems(items []collection.FlatItem) {
	m.items = items
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// SetSize sets the panel dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused sets whether this panel has focus.
func (m *Model) SetFocused(f bool) {
	m.focused = f
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.filtering {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		if msg.String() == "/" {
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}
		return m, nil
	}

	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.filtered) - 1
	case "enter", "l":
		idx := m.filtered[m.cursor]
		item := &m.items[idx]
		if item.IsFolder {
			m.toggleFolder(idx)
		} else if item.Request != nil {
			return m, func() tea.Msg {
				return msgs.RequestSelectedMsg{RequestID: item.Request.ID}
			}
		}
	case "h":
		idx := m.filtered[m.cursor]
		item := &m.items[idx]
		if item.IsFolder && item.Expanded {
			m.toggleFolder(idx)
		}
	case "/":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

func (m Model) updateFilter(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.filtering = false
			m.filterInput.Blur()
			if msg.String() == "esc" {
				m.filterInput.SetValue("")
				m.applyFilter()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m *Model) toggleFolder(idx int) {
	folder := &m.items[idx]
	folder.Expanded = !folder.Expanded
	m.rebuildFromToggle()
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// rebuildFromToggle hides/shows children of collapsed folders.
// We mark collapsed children by checking parent expand state during filter.
func (m *Model) rebuildFromToggle() {
	// No structural rebuild needed; visibility is handled during rendering
	// by checking if ancestors are expanded.
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filterInput.Value())
	m.filtered = m.filtered[:0]

	// Track collapsed folder depth: if > 0, skip items at deeper depths.
	skipDepth := -1

	for i, item := range m.items {
		// Handle collapsed folders: skip children of collapsed folders.
		if skipDepth >= 0 && item.Depth > skipDepth {
			continue
		}
		skipDepth = -1

		if item.IsFolder && !item.Expanded {
			skipDepth = item.Depth
		}

		if query == "" {
			m.filtered = append(m.filtered, i)
			continue
		}

		name := ""
		if item.IsFolder && item.Folder != nil {
			name = item.Folder.Name
		} else if item.Request != nil {
			name = item.Request.Name
		}
		if strings.Contains(strings.ToLower(name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	border := m.styles.UnfocusedBorder
	if m.focused {
		border = m.styles.FocusedBorder
	}

	// Account for border (2 chars each side)
	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := m.height - 2
	if innerH < 1 {
		innerH = 1
	}

	// Title
	title := m.styles.Title.Render("Collections")

	// Build tree lines
	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	if len(m.filtered) == 0 {
		lines = append(lines, m.styles.Muted.Render("  No items"))
	} else {
		for vi, idx := range m.filtered {
			item := m.items[idx]
			line := m.renderItem(item, vi == m.cursor, innerW)
			lines = append(lines, line)
		}
	}

	// History placeholder
	historyHeader := fmt.Sprintf("\n%s\n%s",
		m.styles.Title.Render("History"),
		m.styles.Muted.Render("  No history yet"),
	)

	// Calculate available space
	treeContent := strings.Join(lines, "\n")
	filterLine := ""
	if m.filtering {
		filterLine = m.filterInput.View()
	}

	// Compose content
	var content string
	if m.filtering {
		// Reserve 1 line for filter at bottom
		availH := innerH - 1
		content = m.fitHeight(treeContent+historyHeader, availH)
		content += "\n" + filterLine
	} else {
		content = m.fitHeight(treeContent+historyHeader, innerH)
	}

	return border.
		Width(innerW).
		Height(innerH).
		Render(content)
}

func (m Model) renderItem(item collection.FlatItem, isCursor bool, maxWidth int) string {
	indent := strings.Repeat("  ", item.Depth)

	var line string
	if item.IsFolder && item.Folder != nil {
		icon := "▶ "
		if item.Expanded {
			icon = "▼ "
		}
		name := item.Folder.Name
		line = indent + m.styles.TreeFolder.Render(icon+name)
	} else if item.Request != nil {
		method := padMethod(item.Request.Method)
		badge := m.styles.MethodStyle(item.Request.Method).Render(method)
		name := m.styles.TreeItem.
			PaddingLeft(0). // override default padding; we handle indent ourselves
			Render(item.Request.Name)
		line = indent + badge + " " + name
	}

	if isCursor {
		// Render with cursor highlight across full width
		plain := stripForWidth(line, maxWidth)
		return m.styles.Cursor.Width(maxWidth).Render(plain)
	}

	return line
}

// padMethod pads an HTTP method to 6 chars.
func padMethod(method string) string {
	if len(method) >= 6 {
		return method[:6]
	}
	return method + strings.Repeat(" ", 6-len(method))
}

// fitHeight truncates or pads content to the given height.
func (m Model) fitHeight(content string, h int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// stripForWidth returns the raw text for cursor rendering.
// We use lipgloss to measure and truncate.
func stripForWidth(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// Truncate by runes until it fits
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > w {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}
