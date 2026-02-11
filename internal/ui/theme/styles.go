package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds pre-computed Lip Gloss styles for the current theme.
type Styles struct {
	// Panel borders
	FocusedBorder   lipgloss.Style
	UnfocusedBorder lipgloss.Style

	// Text styles
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Normal     lipgloss.Style
	Muted      lipgloss.Style
	Bold       lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	URL        lipgloss.Style
	Key        lipgloss.Style
	Value      lipgloss.Style
	Hint       lipgloss.Style
	StatusText lipgloss.Style

	// HTTP method styles
	MethodGET    lipgloss.Style
	MethodPOST   lipgloss.Style
	MethodPUT    lipgloss.Style
	MethodPATCH  lipgloss.Style
	MethodDELETE lipgloss.Style

	// Components
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	StatusBar   lipgloss.Style
	Sidebar     lipgloss.Style
	TreeItem    lipgloss.Style
	TreeFolder  lipgloss.Style
	Selected    lipgloss.Style
	Cursor      lipgloss.Style

	// KV table
	KVKey       lipgloss.Style
	KVValue     lipgloss.Style
	KVSeparator lipgloss.Style
	KVDisabled  lipgloss.Style
}

// NewStyles creates a Styles set from a Theme.
func NewStyles(t Theme) Styles {
	return Styles{
		FocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocused),
		UnfocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderUnfocused),

		Title:    lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		Subtitle: lipgloss.NewStyle().Foreground(t.Subtext),
		Normal:   lipgloss.NewStyle().Foreground(t.Text),
		Muted:    lipgloss.NewStyle().Foreground(t.Muted),
		Bold:     lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		Error:    lipgloss.NewStyle().Foreground(t.Red),
		Success:  lipgloss.NewStyle().Foreground(t.Green),
		Warning:  lipgloss.NewStyle().Foreground(t.Yellow),
		URL:      lipgloss.NewStyle().Foreground(t.Blue).Underline(true),
		Key:      lipgloss.NewStyle().Foreground(t.Mauve),
		Value:    lipgloss.NewStyle().Foreground(t.Text),
		Hint:     lipgloss.NewStyle().Foreground(t.Muted).Italic(true),
		StatusText: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Surface).
			Padding(0, 1),

		MethodGET:    lipgloss.NewStyle().Foreground(t.Green).Bold(true),
		MethodPOST:   lipgloss.NewStyle().Foreground(t.Yellow).Bold(true),
		MethodPUT:    lipgloss.NewStyle().Foreground(t.Blue).Bold(true),
		MethodPATCH:  lipgloss.NewStyle().Foreground(t.Peach).Bold(true),
		MethodDELETE: lipgloss.NewStyle().Foreground(t.Red).Bold(true),

		TabActive: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Surface).
			Bold(true).
			Padding(0, 2),
		TabInactive: lipgloss.NewStyle().
			Foreground(t.Subtext).
			Padding(0, 2),
		StatusBar: lipgloss.NewStyle().
			Background(t.Surface).
			Foreground(t.Text).
			Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			Foreground(t.Text),
		TreeItem: lipgloss.NewStyle().
			Foreground(t.Text).
			PaddingLeft(2),
		TreeFolder: lipgloss.NewStyle().
			Foreground(t.Mauve).
			Bold(true),
		Selected: lipgloss.NewStyle().
			Background(t.Surface).
			Foreground(t.Text),
		Cursor: lipgloss.NewStyle().
			Background(t.Overlay).
			Foreground(t.Text),

		KVKey: lipgloss.NewStyle().
			Foreground(t.Mauve),
		KVValue: lipgloss.NewStyle().
			Foreground(t.Text),
		KVSeparator: lipgloss.NewStyle().
			Foreground(t.Muted),
		KVDisabled: lipgloss.NewStyle().
			Foreground(t.Muted).
			Strikethrough(true),
	}
}

// MethodStyle returns the style for an HTTP method.
func (s Styles) MethodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return s.MethodGET
	case "POST":
		return s.MethodPOST
	case "PUT":
		return s.MethodPUT
	case "PATCH":
		return s.MethodPATCH
	case "DELETE":
		return s.MethodDELETE
	default:
		return s.Normal
	}
}
