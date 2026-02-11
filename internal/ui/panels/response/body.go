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

	"github.com/sadopc/gottp/internal/ui/theme"
)

// BodyModel displays the response body with syntax highlighting.
type BodyModel struct {
	viewport  viewport.Model
	search    SearchBar
	styles    theme.Styles
	width     int
	height    int
	wrap      bool
	hasBody   bool
	searching bool
	raw       []byte
	contType  string
}

// NewBodyModel creates a new body viewer.
func NewBodyModel(s theme.Styles) BodyModel {
	vp := viewport.New(0, 0)
	return BodyModel{
		viewport: vp,
		search:   NewSearchBar(s),
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
	m.search.SetWidth(w)
	vpH := h
	if m.searching {
		vpH-- // Reserve 1 line for search bar
	}
	m.viewport.Width = w
	m.viewport.Height = vpH
	if m.hasBody {
		m.renderContent()
	}
}

// Searching returns whether search is active.
func (m BodyModel) Searching() bool {
	return m.searching
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

func (m *BodyModel) renderContentWithSearch() {
	if !m.hasBody {
		return
	}

	src := m.raw
	lexerName := detectLexer(m.contType)
	if lexerName == "json" {
		src = pretty.Pretty(src)
	}

	// For search highlighting, use plain text to avoid ANSI interference
	content := string(src)
	if m.wrap && m.width > 0 {
		content = wrapText(content, m.width)
	}

	highlighted, matchLines := HighlightMatches(content, m.search.Query())
	m.search.SetMatches(matchLines)
	m.viewport.SetContent(highlighted)

	// Jump to first match
	if len(matchLines) > 0 {
		m.viewport.SetYOffset(matchLines[0])
	}
}

func (m BodyModel) Init() tea.Cmd {
	return nil
}

func (m BodyModel) Update(msg tea.Msg) (BodyModel, tea.Cmd) {
	// If search input is active, delegate to search bar
	if m.searching && m.search.input.Focused() {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if !m.search.Active() {
			// Search was closed with Esc
			m.searching = false
			m.viewport.Height = m.height
			m.renderContent()
		} else if m.search.Query() != "" {
			// Re-render with highlights
			m.renderContentWithSearch()
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/", "ctrl+f":
			m.searching = true
			m.search.Open()
			m.viewport.Height = m.height - 1
			return m, nil
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
		case "n":
			if m.searching && m.search.Query() != "" {
				m.search.NextMatch()
				line := m.search.CurrentMatchLine()
				if line >= 0 {
					m.viewport.SetYOffset(line)
				}
				return m, nil
			}
		case "N":
			if m.searching && m.search.Query() != "" {
				m.search.PrevMatch()
				line := m.search.CurrentMatchLine()
				if line >= 0 {
					m.viewport.SetYOffset(line)
				}
				return m, nil
			}
		case "esc":
			if m.searching {
				m.searching = false
				m.search.Close()
				m.viewport.Height = m.height
				m.renderContent()
				return m, nil
			}
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
	if m.searching {
		return m.viewport.View() + "\n" + m.search.View()
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
