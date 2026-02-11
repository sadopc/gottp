package response

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/theme"
)

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Direction string // "sent" or "received"
	Content   string
	Timestamp time.Time
	IsJSON    bool
}

// WSLogModel displays a scrollable log of WebSocket messages.
type WSLogModel struct {
	viewport viewport.Model
	messages []WSMessage
	styles   theme.Styles
	th       theme.Theme
	width    int
	height   int
}

// NewWSLogModel creates a new WebSocket log model.
func NewWSLogModel(t theme.Theme, s theme.Styles) WSLogModel {
	vp := viewport.New(40, 10)
	return WSLogModel{
		viewport: vp,
		styles:   s,
		th:       t,
	}
}

// AddMessage appends a message to the log.
func (m *WSLogModel) AddMessage(msg WSMessage) {
	m.messages = append(m.messages, msg)
	m.updateContent()
}

// Clear removes all messages.
func (m *WSLogModel) Clear() {
	m.messages = nil
	m.viewport.SetContent("")
}

// SetSize updates the dimensions.
func (m *WSLogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.updateContent()
}

func (m *WSLogModel) updateContent() {
	var lines []string
	for _, msg := range m.messages {
		ts := msg.Timestamp.Format("15:04:05")
		var prefix string
		var style lipgloss.Style

		if msg.Direction == "sent" {
			prefix = ">>> "
			style = lipgloss.NewStyle().Foreground(m.th.Green)
		} else {
			prefix = "<<< "
			style = lipgloss.NewStyle().Foreground(m.th.Blue)
		}

		tsStyle := lipgloss.NewStyle().Foreground(m.th.Muted)
		header := tsStyle.Render(ts) + " " + style.Render(prefix+msg.Direction)
		lines = append(lines, header)

		// Indent content
		for _, line := range strings.Split(msg.Content, "\n") {
			lines = append(lines, "    "+line)
		}
		lines = append(lines, "")
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.Muted.Render("No messages yet"))
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

// MessageCount returns the number of messages.
func (m WSLogModel) MessageCount() int {
	return len(m.messages)
}

func (m WSLogModel) Update(msg tea.Msg) (WSLogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the WebSocket log.
func (m WSLogModel) View() string {
	header := m.styles.Hint.Render(fmt.Sprintf("%d messages", len(m.messages)))
	return header + "\n" + m.viewport.View()
}
