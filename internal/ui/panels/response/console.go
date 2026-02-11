package response

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/theme"
)

// ScriptTestResult holds a single test assertion result for display.
type ScriptTestResult struct {
	Name   string
	Passed bool
	Error  string
}

// ConsoleModel displays script logs and test results.
type ConsoleModel struct {
	viewport    viewport.Model
	logs        []string
	testResults []ScriptTestResult
	err         string
	styles      theme.Styles
	th          theme.Theme
	width       int
	height      int
}

// NewConsoleModel creates a new console model.
func NewConsoleModel(t theme.Theme, s theme.Styles) ConsoleModel {
	vp := viewport.New(40, 10)
	return ConsoleModel{
		viewport: vp,
		styles:   s,
		th:       t,
	}
}

// SetResults populates the console with script execution results.
func (m *ConsoleModel) SetResults(logs []string, tests []ScriptTestResult, errMsg string) {
	m.logs = logs
	m.testResults = tests
	m.err = errMsg
	m.updateContent()
}

// Clear resets the console.
func (m *ConsoleModel) Clear() {
	m.logs = nil
	m.testResults = nil
	m.err = ""
	m.viewport.SetContent("")
}

// SetSize updates the dimensions.
func (m *ConsoleModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.updateContent()
}

func (m *ConsoleModel) updateContent() {
	var lines []string

	// Error
	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(m.th.Red).Bold(true)
		lines = append(lines, errStyle.Render("Error: "+m.err))
		lines = append(lines, "")
	}

	// Test results
	if len(m.testResults) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(m.th.Text).Render("Tests:"))
		passStyle := lipgloss.NewStyle().Foreground(m.th.Green)
		failStyle := lipgloss.NewStyle().Foreground(m.th.Red)
		for _, tr := range m.testResults {
			if tr.Passed {
				lines = append(lines, passStyle.Render("  PASS ")+tr.Name)
			} else {
				lines = append(lines, failStyle.Render("  FAIL ")+tr.Name)
				if tr.Error != "" {
					lines = append(lines, failStyle.Render("       "+tr.Error))
				}
			}
		}
		lines = append(lines, "")
	}

	// Logs
	if len(m.logs) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(m.th.Text).Render("Console:"))
		logStyle := lipgloss.NewStyle().Foreground(m.th.Muted)
		for _, log := range m.logs {
			lines = append(lines, logStyle.Render("  "+log))
		}
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.Muted.Render("No script output"))
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

// HasContent returns whether the console has any content.
func (m ConsoleModel) HasContent() bool {
	return len(m.logs) > 0 || len(m.testResults) > 0 || m.err != ""
}

func (m ConsoleModel) Update(msg tea.Msg) (ConsoleModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the console.
func (m ConsoleModel) View() string {
	return m.viewport.View()
}
