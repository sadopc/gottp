package response

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tidwall/pretty"

	"github.com/serdar/gottp/internal/ui/theme"
)

// BodyModel displays the response body with syntax highlighting.
type BodyModel struct {
	viewport viewport.Model
	styles   theme.Styles
	width    int
	height   int
	wrap     bool
	hasBody  bool
	raw      []byte
	contType string
}

// NewBodyModel creates a new body viewer.
func NewBodyModel(s theme.Styles) BodyModel {
	vp := viewport.New(0, 0)
	return BodyModel{
		viewport: vp,
		styles:   s,
	}
}

// SetContent sets the body content and highlights it.
func (m *BodyModel) SetContent(body []byte, contentType string) {
	m.raw = body
	m.contType = contentType
	m.hasBody = len(body) > 0
	m.renderContent()
}

// SetSize updates the viewport dimensions.
func (m *BodyModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if m.hasBody {
		m.renderContent()
	}
}

func (m *BodyModel) renderContent() {
	if !m.hasBody {
		return
	}

	src := m.raw
	lexerName := detectLexer(m.contType)

	// Pretty-print JSON before highlighting
	if lexerName == "json" {
		src = pretty.Pretty(src)
	}

	highlighted := highlight(string(src), lexerName, m.width, m.wrap)
	m.viewport.SetContent(highlighted)
}

func (m BodyModel) Init() tea.Cmd {
	return nil
}

func (m BodyModel) Update(msg tea.Msg) (BodyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "w":
			m.wrap = !m.wrap
			m.renderContent()
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m BodyModel) View() string {
	if !m.hasBody {
		return m.styles.Muted.Render("No response yet")
	}
	return m.viewport.View()
}

// detectLexer maps Content-Type to a chroma lexer name.
func detectLexer(contentType string) string {
	ct := strings.ToLower(contentType)
	switch {
	case ct == "application/json" || strings.Contains(ct, "json"):
		return "json"
	case ct == "text/html" || strings.Contains(ct, "html"):
		return "html"
	case ct == "text/xml" || ct == "application/xml" || strings.Contains(ct, "xml"):
		return "xml"
	case ct == "text/css":
		return "css"
	case ct == "text/javascript" || ct == "application/javascript" || strings.Contains(ct, "javascript"):
		return "javascript"
	default:
		return "text"
	}
}

// highlight applies chroma syntax highlighting to source code.
func highlight(source, lexerName string, width int, wrap bool) string {
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := chromastyles.Get("monokai")
	if style == nil {
		style = chromastyles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source
	}

	result := buf.String()

	if wrap && width > 0 {
		result = wrapText(result, width)
	}

	return result
}

// wrapText performs simple word wrapping using lipgloss.
func wrapText(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}
