package response

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sadopc/gottp/internal/ui/theme"
)

// HeadersModel displays response headers as a two-column list.
type HeadersModel struct {
	viewport   viewport.Model
	styles     theme.Styles
	width      int
	height     int
	hasHeaders bool
}

// NewHeadersModel creates a new headers viewer.
func NewHeadersModel(s theme.Styles) HeadersModel {
	vp := viewport.New(0, 0)
	return HeadersModel{
		viewport: vp,
		styles:   s,
	}
}

// SetHeaders populates the header display.
func (m *HeadersModel) SetHeaders(headers http.Header) {
	m.hasHeaders = len(headers) > 0
	if !m.hasHeaders {
		return
	}

	// Sort header keys for consistent display
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		values := headers[k]
		key := m.styles.Key.Render(k)
		sep := m.styles.Muted.Render(" : ")
		val := m.styles.Normal.Render(strings.Join(values, ", "))
		fmt.Fprintf(&b, "%s%s%s\n", key, sep, val)
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

// SetSize updates the viewport dimensions.
func (m *HeadersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
}

func (m HeadersModel) Init() tea.Cmd {
	return nil
}

func (m HeadersModel) Update(msg tea.Msg) (HeadersModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m HeadersModel) View() string {
	if !m.hasHeaders {
		return m.styles.Muted.Render("No headers")
	}
	return m.viewport.View()
}
