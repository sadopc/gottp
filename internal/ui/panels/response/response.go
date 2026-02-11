package response

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/theme"
)

type subTab int

const (
	tabBody subTab = iota
	tabHeaders
	tabCookies
	tabTiming
)

var subTabLabels = []string{"Body", "Headers", "Cookies", "Timing"}

// Model is the response panel container wrapping body, headers, cookies, and timing.
type Model struct {
	body    BodyModel
	headers HeadersModel
	cookies CookiesModel
	timing  TimingModel
	spinner spinner.Model

	styles  theme.Styles
	th      theme.Theme
	active  subTab
	focused bool
	loading bool
	hasResp bool
	status  string
	code    int
	width   int
	height  int
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
		timing:  NewTimingModel(s),
		spinner: sp,
		styles:  s,
		th:      t,
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
}

// SetLoading puts the panel into loading state.
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SetFocused sets whether this panel has focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
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
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.active = (m.active + 1) % subTab(len(subTabLabels))
			return m, nil
		case "shift+tab":
			if m.active == 0 {
				m.active = subTab(len(subTabLabels) - 1)
			} else {
				m.active--
			}
			return m, nil
		case "1":
			m.active = tabBody
			return m, nil
		case "2":
			m.active = tabHeaders
			return m, nil
		case "3":
			m.active = tabCookies
			return m, nil
		case "4":
			m.active = tabTiming
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
	switch m.active {
	case tabBody:
		m.body, cmd = m.body.Update(msg)
	case tabHeaders:
		m.headers, cmd = m.headers.Update(msg)
	case tabCookies:
		m.cookies, cmd = m.cookies.Update(msg)
	case tabTiming:
		m.timing, cmd = m.timing.Update(msg)
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
	switch m.active {
	case tabBody:
		body = m.body.View()
	case tabHeaders:
		body = m.headers.View()
	case tabCookies:
		body = m.cookies.View()
	case tabTiming:
		body = m.timing.View()
	}

	body = lipgloss.NewStyle().Width(w).Height(contentH).Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, status, body)
}

func (m Model) renderTabs(width int) string {
	var tabs []string
	for i, label := range subTabLabels {
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
	color := m.th.StatusColor(m.code)
	statusStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	return statusStyle.Width(width).Render(m.status)
}
