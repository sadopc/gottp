package theme

import "github.com/charmbracelet/lipgloss"

var Dracula = Theme{
	Name:    "Dracula",
	Base:    lipgloss.Color("#282a36"),
	Mantle:  lipgloss.Color("#21222c"),
	Crust:   lipgloss.Color("#191a21"),
	Surface: lipgloss.Color("#44475a"),
	Overlay: lipgloss.Color("#6272a4"),

	Text:    lipgloss.Color("#f8f8f2"),
	Subtext: lipgloss.Color("#d0d0d0"),
	Muted:   lipgloss.Color("#6272a4"),

	Rosewater: lipgloss.Color("#ffb86c"),
	Flamingo:  lipgloss.Color("#ff79c6"),
	Pink:      lipgloss.Color("#ff79c6"),
	Mauve:     lipgloss.Color("#bd93f9"),
	Red:       lipgloss.Color("#ff5555"),
	Maroon:    lipgloss.Color("#ff5555"),
	Peach:     lipgloss.Color("#ffb86c"),
	Yellow:    lipgloss.Color("#f1fa8c"),
	Green:     lipgloss.Color("#50fa7b"),
	Teal:      lipgloss.Color("#8be9fd"),
	Sky:       lipgloss.Color("#8be9fd"),
	Sapphire:  lipgloss.Color("#8be9fd"),
	Blue:      lipgloss.Color("#6272a4"),
	Lavender:  lipgloss.Color("#bd93f9"),

	BorderFocused:   lipgloss.Color("#bd93f9"),
	BorderUnfocused: lipgloss.Color("#6272a4"),
	StatusOK:        lipgloss.Color("#50fa7b"),
	StatusError:     lipgloss.Color("#ff5555"),
	StatusWarning:   lipgloss.Color("#f1fa8c"),
}
