package theme

import "github.com/charmbracelet/lipgloss"

var TokyoNight = Theme{
	Name:    "Tokyo Night",
	Base:    lipgloss.Color("#1a1b26"),
	Mantle:  lipgloss.Color("#16161e"),
	Crust:   lipgloss.Color("#13131a"),
	Surface: lipgloss.Color("#292e42"),
	Overlay: lipgloss.Color("#3b4261"),

	Text:    lipgloss.Color("#c0caf5"),
	Subtext: lipgloss.Color("#a9b1d6"),
	Muted:   lipgloss.Color("#565f89"),

	Rosewater: lipgloss.Color("#ff9e64"),
	Flamingo:  lipgloss.Color("#ff007c"),
	Pink:      lipgloss.Color("#ff007c"),
	Mauve:     lipgloss.Color("#bb9af7"),
	Red:       lipgloss.Color("#f7768e"),
	Maroon:    lipgloss.Color("#db4b4b"),
	Peach:     lipgloss.Color("#ff9e64"),
	Yellow:    lipgloss.Color("#e0af68"),
	Green:     lipgloss.Color("#9ece6a"),
	Teal:      lipgloss.Color("#73daca"),
	Sky:       lipgloss.Color("#7dcfff"),
	Sapphire:  lipgloss.Color("#2ac3de"),
	Blue:      lipgloss.Color("#7aa2f7"),
	Lavender:  lipgloss.Color("#bb9af7"),

	BorderFocused:   lipgloss.Color("#bb9af7"),
	BorderUnfocused: lipgloss.Color("#565f89"),
	StatusOK:        lipgloss.Color("#9ece6a"),
	StatusError:     lipgloss.Color("#f7768e"),
	StatusWarning:   lipgloss.Color("#e0af68"),
}
