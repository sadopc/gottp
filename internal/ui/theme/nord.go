package theme

import "github.com/charmbracelet/lipgloss"

var Nord = Theme{
	Name:    "Nord",
	Base:    lipgloss.Color("#2e3440"),
	Mantle:  lipgloss.Color("#292e39"),
	Crust:   lipgloss.Color("#242933"),
	Surface: lipgloss.Color("#3b4252"),
	Overlay: lipgloss.Color("#434c5e"),

	Text:    lipgloss.Color("#eceff4"),
	Subtext: lipgloss.Color("#d8dee9"),
	Muted:   lipgloss.Color("#4c566a"),

	Rosewater: lipgloss.Color("#d08770"),
	Flamingo:  lipgloss.Color("#d08770"),
	Pink:      lipgloss.Color("#b48ead"),
	Mauve:     lipgloss.Color("#b48ead"),
	Red:       lipgloss.Color("#bf616a"),
	Maroon:    lipgloss.Color("#bf616a"),
	Peach:     lipgloss.Color("#d08770"),
	Yellow:    lipgloss.Color("#ebcb8b"),
	Green:     lipgloss.Color("#a3be8c"),
	Teal:      lipgloss.Color("#8fbcbb"),
	Sky:       lipgloss.Color("#88c0d0"),
	Sapphire:  lipgloss.Color("#81a1c1"),
	Blue:      lipgloss.Color("#5e81ac"),
	Lavender:  lipgloss.Color("#b48ead"),

	BorderFocused:   lipgloss.Color("#88c0d0"),
	BorderUnfocused: lipgloss.Color("#4c566a"),
	StatusOK:        lipgloss.Color("#a3be8c"),
	StatusError:     lipgloss.Color("#bf616a"),
	StatusWarning:   lipgloss.Color("#ebcb8b"),
}
