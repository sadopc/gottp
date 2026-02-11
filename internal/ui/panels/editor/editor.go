package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/ui/theme"
)

// Model is the editor panel container.
type Model struct {
	form    HTTPForm
	focused bool
	width   int
	height  int
	styles  theme.Styles
}

// New creates a new editor panel.
func New(styles theme.Styles) Model {
	return Model{
		form:   NewHTTPForm(styles),
		styles: styles,
		width:  60,
		height: 20,
	}
}

// SetFocused sets whether the editor panel is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the panel dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Account for border (2 chars each side)
	innerW := w - 2
	innerH := h - 2
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 5 {
		innerH = 5
	}
	m.form.SetSize(innerW, innerH)
}

// Editing returns whether the editor has an active text input.
func (m Model) Editing() bool {
	return m.form.Editing()
}

// Form returns a pointer to the HTTPForm for external access.
func (m *Model) Form() *HTTPForm {
	return &m.form
}

// LoadRequest loads a collection request into the editor.
func (m *Model) LoadRequest(req *collection.Request) {
	m.form.LoadRequest(req)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Ctrl+Enter sends the request regardless of mode
		if msg.String() == "ctrl+enter" {
			return m, func() tea.Msg {
				return msgs.SendRequestMsg{}
			}
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	// Build the send hint
	sendHint := m.styles.Hint.Render("ctrl+enter to send")

	// URL bar line from the form
	formView := m.form.View()

	// Right-align the hint on the first line
	lines := strings.SplitN(formView, "\n", 2)
	urlLine := lines[0]

	// Calculate padding for right-aligned hint
	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}
	urlLineLen := lipgloss.Width(urlLine)
	hintLen := lipgloss.Width(sendHint)
	gap := innerW - urlLineLen - hintLen
	if gap < 1 {
		gap = 1
	}

	header := urlLine + strings.Repeat(" ", gap) + sendHint
	var content string
	if len(lines) > 1 {
		content = header + "\n" + lines[1]
	} else {
		content = header
	}

	// Apply border
	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = m.styles.FocusedBorder
	} else {
		borderStyle = m.styles.UnfocusedBorder
	}
	borderStyle = borderStyle.Width(m.width - 2).Height(m.height - 2)

	return borderStyle.Render(content)
}
