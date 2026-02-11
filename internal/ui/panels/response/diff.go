package response

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/serdar/gottp/internal/diff"
	"github.com/serdar/gottp/internal/ui/theme"
)

// DiffModel displays a diff between two response bodies.
type DiffModel struct {
	viewport viewport.Model
	styles   theme.Styles
	th       theme.Theme
	hasDiff  bool
	summary  string
	width    int
	height   int
}

// NewDiffModel creates a new diff viewer.
func NewDiffModel(t theme.Theme, s theme.Styles) DiffModel {
	return DiffModel{
		viewport: viewport.New(0, 0),
		styles:   s,
		th:       t,
	}
}

// SetDiff computes and displays a diff between baseline and current response.
func (m *DiffModel) SetDiff(baseline, current []byte) {
	lines := diff.DiffLinesWithWords(string(baseline), string(current))
	m.hasDiff = true

	added := 0
	removed := 0

	var b strings.Builder
	for _, line := range lines {
		switch line.Type {
		case diff.Added:
			added++
			prefix := lipgloss.NewStyle().Foreground(m.th.Green).Render("+ ")
			b.WriteString(prefix)
			if len(line.Words) > 0 {
				m.renderWordDiffAdded(&b, line.Words)
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(m.th.Green).Render(line.Content))
			}
			b.WriteString("\n")
		case diff.Removed:
			removed++
			prefix := lipgloss.NewStyle().Foreground(m.th.Red).Render("- ")
			b.WriteString(prefix)
			if len(line.Words) > 0 {
				m.renderWordDiffRemoved(&b, line.Words)
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(m.th.Red).Render(line.Content))
			}
			b.WriteString("\n")
		case diff.Same:
			prefix := lipgloss.NewStyle().Foreground(m.th.Muted).Render("  ")
			content := lipgloss.NewStyle().Foreground(m.th.Text).Render(line.Content)
			b.WriteString(prefix + content + "\n")
		}
	}

	m.summary = fmt.Sprintf("%d added, %d removed", added, removed)
	m.viewport.SetContent(b.String())
}

// renderWordDiffRemoved renders word-level detail for a Removed line.
// Same and Removed words are shown; Added words are skipped (they appear on the paired Added line).
// Removed words are highlighted with bold to show exactly what changed.
func (m *DiffModel) renderWordDiffRemoved(b *strings.Builder, words []diff.WordDiff) {
	baseStyle := lipgloss.NewStyle().Foreground(m.th.Red)
	highlightStyle := baseStyle.Bold(true)

	for _, w := range words {
		switch w.Type {
		case diff.Removed:
			b.WriteString(highlightStyle.Render(w.Content))
		case diff.Same:
			b.WriteString(baseStyle.Render(w.Content))
		case diff.Added:
			// Skip Added words on a Removed line; they belong to the Added line.
		}
	}
}

// renderWordDiffAdded renders word-level detail for an Added line.
// Same and Added words are shown; Removed words are skipped (they appear on the paired Removed line).
// Added words are highlighted with bold to show exactly what changed.
func (m *DiffModel) renderWordDiffAdded(b *strings.Builder, words []diff.WordDiff) {
	baseStyle := lipgloss.NewStyle().Foreground(m.th.Green)
	highlightStyle := baseStyle.Bold(true)

	for _, w := range words {
		switch w.Type {
		case diff.Added:
			b.WriteString(highlightStyle.Render(w.Content))
		case diff.Same:
			b.WriteString(baseStyle.Render(w.Content))
		case diff.Removed:
			// Skip Removed words on an Added line; they belong to the Removed line.
		}
	}
}

// SetSize updates the diff viewport dimensions.
func (m *DiffModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
}

// HasDiff returns whether a diff has been computed.
func (m DiffModel) HasDiff() bool {
	return m.hasDiff
}

// Clear resets the diff state.
func (m *DiffModel) Clear() {
	m.hasDiff = false
	m.viewport.SetContent("")
	m.summary = ""
}

func (m DiffModel) Init() tea.Cmd {
	return nil
}

func (m DiffModel) Update(msg tea.Msg) (DiffModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffModel) View() string {
	if !m.hasDiff {
		return m.styles.Muted.Render("No baseline set. Use command palette to set a baseline.")
	}

	header := m.styles.Hint.Render("Diff: " + m.summary)
	return header + "\n" + m.viewport.View()
}
