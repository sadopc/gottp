package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/ui/theme"
)

// CookiesModel displays parsed Set-Cookie headers.
type CookiesModel struct {
	viewport   viewport.Model
	styles     theme.Styles
	width      int
	height     int
	hasCookies bool
}

// NewCookiesModel creates a new cookies viewer.
func NewCookiesModel(s theme.Styles) CookiesModel {
	vp := viewport.New(0, 0)
	return CookiesModel{
		viewport: vp,
		styles:   s,
	}
}

// SetHeaders parses cookies from response headers.
func (m *CookiesModel) SetHeaders(headers http.Header) {
	setCookies := headers.Values("Set-Cookie")
	m.hasCookies = len(setCookies) > 0
	if !m.hasCookies {
		return
	}

	var b strings.Builder

	// Table header
	headerLine := fmt.Sprintf("  %-20s %-30s %-15s %-10s %-8s %-8s",
		"Name", "Value", "Domain", "Path", "HttpOnly", "Secure")
	b.WriteString(m.styles.Key.Render(headerLine))
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render(strings.Repeat("─", m.width-2)))
	b.WriteString("\n")

	// Parse each Set-Cookie header using net/http
	fakeHeader := http.Header{}
	for _, sc := range setCookies {
		fakeHeader.Add("Set-Cookie", sc)
	}

	resp := &http.Response{Header: fakeHeader}
	cookies := resp.Cookies()

	for _, c := range cookies {
		name := truncateCookie(c.Name, 20)
		value := truncateCookie(c.Value, 30)
		domain := truncateCookie(c.Domain, 15)
		path := truncateCookie(c.Path, 10)

		httpOnly := "no"
		if c.HttpOnly {
			httpOnly = "yes"
		}
		secure := "no"
		if c.Secure {
			secure = "yes"
		}

		nameStr := m.styles.KVKey.Render(fmt.Sprintf("%-20s", name))
		valStr := m.styles.Normal.Render(fmt.Sprintf("%-30s", value))
		domStr := m.styles.Muted.Render(fmt.Sprintf("%-15s", domain))
		pathStr := m.styles.Muted.Render(fmt.Sprintf("%-10s", path))
		httpStr := m.styles.Muted.Render(fmt.Sprintf("%-8s", httpOnly))
		secStr := m.styles.Muted.Render(fmt.Sprintf("%-8s", secure))

		line := fmt.Sprintf("  %s %s %s %s %s %s", nameStr, valStr, domStr, pathStr, httpStr, secStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

// SetSize updates the viewport dimensions.
func (m *CookiesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
}

func (m CookiesModel) Init() tea.Cmd {
	return nil
}

func (m CookiesModel) Update(msg tea.Msg) (CookiesModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m CookiesModel) View() string {
	if !m.hasCookies {
		return m.styles.Muted.Render("No cookies in response")
	}
	return m.viewport.View()
}

func truncateCookie(s string, maxW int) string {
	if len(s) > maxW {
		if maxW > 3 {
			return s[:maxW-3] + "..."
		}
		return s[:maxW]
	}
	return s
}
