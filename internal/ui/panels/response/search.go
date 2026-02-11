package response

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gottp/internal/ui/theme"
)

// SearchBar provides search functionality within the response body.
type SearchBar struct {
	input   textinput.Model
	active  bool
	query   string
	matches []int // line indices of matches
	current int   // index into matches
	styles  theme.Styles
	width   int
}

// NewSearchBar creates a new search bar.
func NewSearchBar(s theme.Styles) SearchBar {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 256
	ti.Prompt = "/ "
	return SearchBar{
		input:  ti,
		styles: s,
	}
}

// Active returns whether the search bar is visible.
func (m SearchBar) Active() bool {
	return m.active
}

// Query returns the current search query.
func (m SearchBar) Query() string {
	return m.query
}

// Open activates the search bar.
func (m *SearchBar) Open() {
	m.active = true
	m.input.SetValue("")
	m.input.Focus()
	m.query = ""
	m.matches = nil
	m.current = 0
}

// Close deactivates the search bar.
func (m *SearchBar) Close() {
	m.active = false
	m.input.Blur()
	m.query = ""
	m.matches = nil
	m.current = 0
}

// SetWidth sets the search bar width.
func (m *SearchBar) SetWidth(w int) {
	m.width = w
	m.input.Width = w - 20
	if m.input.Width < 10 {
		m.input.Width = 10
	}
}

// Update handles messages for the search bar.
func (m SearchBar) Update(msg tea.Msg) (SearchBar, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Close()
			return m, nil
		case "enter":
			m.query = m.input.Value()
			m.input.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.query = m.input.Value()
	return m, cmd
}

// SetMatches updates the match positions.
func (m *SearchBar) SetMatches(matches []int) {
	m.matches = matches
	if len(matches) == 0 {
		m.current = 0
	} else if m.current >= len(matches) {
		m.current = 0
	}
}

// NextMatch moves to the next match.
func (m *SearchBar) NextMatch() {
	if len(m.matches) > 0 {
		m.current = (m.current + 1) % len(m.matches)
	}
}

// PrevMatch moves to the previous match.
func (m *SearchBar) PrevMatch() {
	if len(m.matches) > 0 {
		m.current = (m.current - 1 + len(m.matches)) % len(m.matches)
	}
}

// CurrentMatchLine returns the line number of the current match.
func (m SearchBar) CurrentMatchLine() int {
	if len(m.matches) > 0 && m.current < len(m.matches) {
		return m.matches[m.current]
	}
	return -1
}

// View renders the search bar.
func (m SearchBar) View() string {
	if !m.active {
		return ""
	}

	var info string
	if m.query != "" {
		if len(m.matches) == 0 {
			info = m.styles.Error.Render(" No matches")
		} else {
			info = m.styles.Muted.Render(fmt.Sprintf(" %d/%d", m.current+1, len(m.matches)))
		}
	}

	bar := m.input.View() + info
	return lipgloss.NewStyle().Width(m.width).Render(bar)
}

// HighlightMatches highlights all occurrences of query in content.
func HighlightMatches(content, query string) (string, []int) {
	if query == "" {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	lowerQuery := strings.ToLower(query)
	var matchLines []int

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#f9e2af")).
		Foreground(lipgloss.Color("#1e1e2e")).
		Bold(true)

	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, lowerQuery) {
			matchLines = append(matchLines, i)
			// Highlight occurrences (case-insensitive, preserving original case)
			var result strings.Builder
			remaining := line
			lowerRemaining := lowerLine
			for {
				idx := strings.Index(lowerRemaining, lowerQuery)
				if idx < 0 {
					result.WriteString(remaining)
					break
				}
				result.WriteString(remaining[:idx])
				result.WriteString(highlightStyle.Render(remaining[idx : idx+len(query)]))
				remaining = remaining[idx+len(query):]
				lowerRemaining = lowerRemaining[idx+len(lowerQuery):]
			}
			lines[i] = result.String()
		}
	}

	return strings.Join(lines, "\n"), matchLines
}
