package theme

import "github.com/charmbracelet/lipgloss"

var CatppuccinFrappe = Theme{
	Name:    "Catppuccin Frapp\u00e9",
	Base:    lipgloss.Color("#303446"),
	Mantle:  lipgloss.Color("#292c3c"),
	Crust:   lipgloss.Color("#232634"),
	Surface: lipgloss.Color("#414559"),
	Overlay: lipgloss.Color("#51576d"),

	Text:    lipgloss.Color("#c6d0f5"),
	Subtext: lipgloss.Color("#a5adce"),
	Muted:   lipgloss.Color("#626880"),

	Rosewater: lipgloss.Color("#f2d5cf"),
	Flamingo:  lipgloss.Color("#eebebe"),
	Pink:      lipgloss.Color("#f4b8e4"),
	Mauve:     lipgloss.Color("#ca9ee6"),
	Red:       lipgloss.Color("#e78284"),
	Maroon:    lipgloss.Color("#ea999c"),
	Peach:     lipgloss.Color("#ef9f76"),
	Yellow:    lipgloss.Color("#e5c890"),
	Green:     lipgloss.Color("#a6d189"),
	Teal:      lipgloss.Color("#81c8be"),
	Sky:       lipgloss.Color("#99d1db"),
	Sapphire:  lipgloss.Color("#85c1dc"),
	Blue:      lipgloss.Color("#8caaee"),
	Lavender:  lipgloss.Color("#babbf1"),

	BorderFocused:   lipgloss.Color("#ca9ee6"),
	BorderUnfocused: lipgloss.Color("#626880"),
	StatusOK:        lipgloss.Color("#a6d189"),
	StatusError:     lipgloss.Color("#e78284"),
	StatusWarning:   lipgloss.Color("#e5c890"),
}
