package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds all colors for the application.
type Theme struct {
	Name string

	// Base colors
	Base    lipgloss.Color
	Mantle  lipgloss.Color
	Crust   lipgloss.Color
	Surface lipgloss.Color
	Overlay lipgloss.Color

	// Text
	Text    lipgloss.Color
	Subtext lipgloss.Color
	Muted   lipgloss.Color

	// Accents
	Rosewater lipgloss.Color
	Flamingo  lipgloss.Color
	Pink      lipgloss.Color
	Mauve     lipgloss.Color
	Red       lipgloss.Color
	Maroon    lipgloss.Color
	Peach     lipgloss.Color
	Yellow    lipgloss.Color
	Green     lipgloss.Color
	Teal      lipgloss.Color
	Sky       lipgloss.Color
	Sapphire  lipgloss.Color
	Blue      lipgloss.Color
	Lavender  lipgloss.Color

	// Semantic
	BorderFocused   lipgloss.Color
	BorderUnfocused lipgloss.Color
	StatusOK        lipgloss.Color
	StatusError     lipgloss.Color
	StatusWarning   lipgloss.Color
}

// MethodColor returns the color for an HTTP method.
func (t Theme) MethodColor(method string) lipgloss.Color {
	switch method {
	case "GET":
		return t.Green
	case "POST":
		return t.Yellow
	case "PUT":
		return t.Blue
	case "PATCH":
		return t.Peach
	case "DELETE":
		return t.Red
	case "HEAD":
		return t.Teal
	case "OPTIONS":
		return t.Lavender
	default:
		return t.Text
	}
}

// StatusColor returns the color for an HTTP status code.
func (t Theme) StatusColor(code int) lipgloss.Color {
	switch {
	case code >= 200 && code < 300:
		return t.Green
	case code >= 300 && code < 400:
		return t.Blue
	case code >= 400 && code < 500:
		return t.Yellow
	case code >= 500:
		return t.Red
	default:
		return t.Text
	}
}
