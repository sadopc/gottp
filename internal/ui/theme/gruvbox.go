package theme

import "github.com/charmbracelet/lipgloss"

var GruvboxDark = Theme{
	Name:    "Gruvbox Dark",
	Base:    lipgloss.Color("#282828"),
	Mantle:  lipgloss.Color("#1d2021"),
	Crust:   lipgloss.Color("#1d2021"),
	Surface: lipgloss.Color("#3c3836"),
	Overlay: lipgloss.Color("#504945"),

	Text:    lipgloss.Color("#ebdbb2"),
	Subtext: lipgloss.Color("#d5c4a1"),
	Muted:   lipgloss.Color("#665c54"),

	Rosewater: lipgloss.Color("#d65d0e"),
	Flamingo:  lipgloss.Color("#d65d0e"),
	Pink:      lipgloss.Color("#d3869b"),
	Mauve:     lipgloss.Color("#b16286"),
	Red:       lipgloss.Color("#cc241d"),
	Maroon:    lipgloss.Color("#fb4934"),
	Peach:     lipgloss.Color("#d65d0e"),
	Yellow:    lipgloss.Color("#d79921"),
	Green:     lipgloss.Color("#98971a"),
	Teal:      lipgloss.Color("#689d6a"),
	Sky:       lipgloss.Color("#83a598"),
	Sapphire:  lipgloss.Color("#83a598"),
	Blue:      lipgloss.Color("#458588"),
	Lavender:  lipgloss.Color("#b16286"),

	BorderFocused:   lipgloss.Color("#d79921"),
	BorderUnfocused: lipgloss.Color("#665c54"),
	StatusOK:        lipgloss.Color("#98971a"),
	StatusError:     lipgloss.Color("#cc241d"),
	StatusWarning:   lipgloss.Color("#d79921"),
}
