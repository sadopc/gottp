package response

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sadopc/gottp/internal/protocol"
	"github.com/sadopc/gottp/internal/ui/theme"
)

type subTab int

const (
	tabBody subTab = iota
	tabHeaders
	tabCookies
	tabTiming
	tabDiff
	tabConsole
)

// responseMode determines which tab set to show.
type responseMode int

const (
	modeHTTP responseMode = iota
	modeWebSocket
)

var httpTabLabels = []string{"Body", "Headers", "Cookies", "Timing", "Diff", "Console"}
var wsTabLabels = []string{"Messages", "Headers", "Timing"}

// ws-specific tabs
const (
	wsTabMessages subTab = 0
	wsTabHeaders  subTab = 1
	wsTabTiming   subTab = 2
)

// Model is the response panel container wrapping body, headers, cookies, timing, diff, console, and WS log.
type Model struct {
	body    BodyModel
	headers HeadersModel
	cookies CookiesModel
	timing  TimingModel
	diff    DiffModel
	console ConsoleModel
	wslog   WSLogModel
	spinner spinner.Model

	styles   theme.Styles
	th       theme.Theme
	active   subTab
	mode     responseMode
	focused  bool
	loading  bool
	hasResp  bool
	status   string
	code     int
	width    int
	height   int
	baseline []byte
}

// New creates a new response panel model.
func New(t theme.Theme, s theme.Styles) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(t.Mauve)

	return Model{
		body:    NewBodyModel(s),
		headers: NewHeadersModel(s),
		cookies: NewCookiesModel(s),
		timing:  NewTimingModel(t, s),
		diff:    NewDiffModel(t, s),
		console: NewConsoleModel(t, s),
		wslog:   NewWSLogModel(t, s),
		spinner: sp,
		styles:  s,
		th:      t,
	}
}

// SetMode switches between HTTP and WebSocket response modes.
func (m *Model) SetMode(proto string) {
	if proto == "websocket" {
		m.mode = modeWebSocket
		m.active = wsTabMessages
	} else {
		m.mode = modeHTTP
		m.active = tabBody
	}
}

// SetResponse populates all sub-models from a response.
func (m *Model) SetResponse(resp *protocol.Response) {
	m.loading = false
	if resp == nil {
		m.hasResp = false
		return
	}
	m.hasResp = true
	m.code = resp.StatusCode
	m.status = resp.Status

	m.body.SetContent(resp.Body, resp.ContentType)
	m.headers.SetHeaders(resp.Headers)
	m.cookies.SetHeaders(resp.Headers)
	m.timing.SetResponse(resp)

	// Auto-compute diff if baseline exists
	if m.baseline != nil {
		m.diff.SetDiff(m.baseline, resp.Body)
	}
}

// SetBaseline saves the current response body as the diff baseline.
func (m *Model) SetBaseline(body []byte) {
	m.baseline = make([]byte, len(body))
	copy(m.baseline, body)
}

// ClearBaseline removes the saved diff baseline.
func (m *Model) ClearBaseline() {
	m.baseline = nil
	m.diff.Clear()
}

// HasBaseline returns whether a baseline is set.
func (m Model) HasBaseline() bool {
	return m.baseline != nil
}

// ResponseBody returns the current response body.
func (m Model) ResponseBody() []byte {
	return m.body.raw
}

// SetLoading puts the panel into loading state.
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SetFocused sets whether this panel has focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// AddWSMessage adds a WebSocket message to the log.
func (m *Model) AddWSMessage(msg WSMessage) {
	m.wslog.AddMessage(msg)
	m.hasResp = true
}

// ClearWSLog clears the WebSocket message log.
func (m *Model) ClearWSLog() {
	m.wslog.Clear()
}

// SetScriptResults sets the script console output.
func (m *Model) SetScriptResults(logs []string, tests []ScriptTestResult, errMsg string) {
	m.console.SetResults(logs, tests, errMsg)
}

// SetSize updates the panel dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Reserve space: 1 for tab bar, 1 for status line, 2 for border
	innerW := w - 2
	innerH := h - 4
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	m.body.SetSize(innerW, innerH)
	m.headers.SetSize(innerW, innerH)
	m.cookies.SetSize(innerW, innerH)
	m.timing.SetSize(innerW, innerH)
	m.diff.SetSize(innerW, innerH)
	m.console.SetSize(innerW, innerH)
	m.wslog.SetSize(innerW, innerH)
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) tabLabels() []string {
	if m.mode == modeWebSocket {
		return wsTabLabels
	}
	return httpTabLabels
}

func (m Model) tabCount() int {
	return len(m.tabLabels())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.active = (m.active + 1) % subTab(m.tabCount())
			return m, nil
		case "shift+tab":
			if m.active == 0 {
				m.active = subTab(m.tabCount() - 1)
			} else {
				m.active--
			}
			return m, nil
		case "1":
			m.active = 0
			return m, nil
		case "2":
			if m.tabCount() > 1 {
				m.active = 1
			}
			return m, nil
		case "3":
			if m.tabCount() > 2 {
				m.active = 2
			}
			return m, nil
		case "4":
			if m.tabCount() > 3 {
				m.active = 3
			}
			return m, nil
		case "5":
			if m.tabCount() > 4 {
				m.active = 4
			}
			return m, nil
		case "6":
			if m.tabCount() > 5 {
				m.active = 5
			}
			return m, nil
		}
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Delegate to active sub-model
	var cmd tea.Cmd
	if m.mode == modeWebSocket {
		switch m.active {
		case wsTabMessages:
			m.wslog, cmd = m.wslog.Update(msg)
		case wsTabHeaders:
			m.headers, cmd = m.headers.Update(msg)
		case wsTabTiming:
			m.timing, cmd = m.timing.Update(msg)
		}
	} else {
		switch m.active {
		case tabBody:
			m.body, cmd = m.body.Update(msg)
		case tabHeaders:
			m.headers, cmd = m.headers.Update(msg)
		case tabCookies:
			m.cookies, cmd = m.cookies.Update(msg)
		case tabTiming:
			m.timing, cmd = m.timing.Update(msg)
		case tabDiff:
			m.diff, cmd = m.diff.Update(msg)
		case tabConsole:
			m.console, cmd = m.console.Update(msg)
		}
	}

	return m, cmd
}

func (m Model) View() string {
	border := m.styles.UnfocusedBorder
	if m.focused {
		border = m.styles.FocusedBorder
	}

	innerW := m.width - 2
	if innerW < 0 {
		innerW = 0
	}
	innerH := m.height - 2
	if innerH < 0 {
		innerH = 0
	}

	var content string
	if m.loading {
		content = m.renderLoading(innerW, innerH)
	} else if !m.hasResp {
		content = m.renderEmpty(innerW, innerH)
	} else {
		content = m.renderResponse(innerW, innerH)
	}

	return border.Width(innerW).Height(innerH).Render(content)
}

func (m Model) renderLoading(w, h int) string {
	msg := fmt.Sprintf("%s Sending request...", m.spinner.View())
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
}

func (m Model) renderEmpty(w, h int) string {
	msg := m.styles.Muted.Render("Send a request to see the response")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
}

func (m Model) renderResponse(w, h int) string {
	tabs := m.renderTabs(w)
	status := m.renderStatus(w)

	// Content height: total minus tab bar and status line
	contentH := h - 2
	if contentH < 0 {
		contentH = 0
	}

	var body string
	if m.mode == modeWebSocket {
		switch m.active {
		case wsTabMessages:
			body = m.wslog.View()
		case wsTabHeaders:
			body = m.headers.View()
		case wsTabTiming:
			body = m.timing.View()
		}
	} else {
		switch m.active {
		case tabBody:
			body = m.body.View()
		case tabHeaders:
			body = m.headers.View()
		case tabCookies:
			body = m.cookies.View()
		case tabTiming:
			body = m.timing.View()
		case tabDiff:
			body = m.diff.View()
		case tabConsole:
			body = m.console.View()
		}
	}

	body = lipgloss.NewStyle().Width(w).Height(contentH).Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, status, body)
}

func (m Model) renderTabs(width int) string {
	labels := m.tabLabels()
	var tabs []string
	for i, label := range labels {
		if subTab(i) == m.active {
			tabs = append(tabs, m.styles.TabActive.Render(label))
		} else {
			tabs = append(tabs, m.styles.TabInactive.Render(label))
		}
	}
	row := strings.Join(tabs, " ")
	return lipgloss.NewStyle().Width(width).Render(row)
}

func (m Model) renderStatus(width int) string {
	if m.mode == modeWebSocket {
		if m.wslog.MessageCount() > 0 {
			statusStyle := lipgloss.NewStyle().Foreground(m.th.Green).Bold(true)
			return statusStyle.Width(width).Render("WebSocket Connected")
		}
		return lipgloss.NewStyle().Foreground(m.th.Muted).Width(width).Render("WebSocket")
	}
	color := m.th.StatusColor(m.code)
	statusStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	return statusStyle.Width(width).Render(m.status)
}
