package theme

import "github.com/charmbracelet/lipgloss"

var CatppuccinMacchiato = Theme{
	Name:    "Catppuccin Macchiato",
	Base:    lipgloss.Color("#24273a"),
	Mantle:  lipgloss.Color("#1e2030"),
	Crust:   lipgloss.Color("#181926"),
	Surface: lipgloss.Color("#363a4f"),
	Overlay: lipgloss.Color("#494d64"),

	Text:    lipgloss.Color("#cad3f5"),
	Subtext: lipgloss.Color("#a5adcb"),
	Muted:   lipgloss.Color("#5b6078"),

	Rosewater: lipgloss.Color("#f4dbd6"),
	Flamingo:  lipgloss.Color("#f0c6c6"),
	Pink:      lipgloss.Color("#f5bde6"),
	Mauve:     lipgloss.Color("#c6a0f6"),
	Red:       lipgloss.Color("#ed8796"),
	Maroon:    lipgloss.Color("#ee99a0"),
	Peach:     lipgloss.Color("#f5a97f"),
	Yellow:    lipgloss.Color("#eed49f"),
	Green:     lipgloss.Color("#a6da95"),
	Teal:      lipgloss.Color("#8bd5ca"),
	Sky:       lipgloss.Color("#91d7e3"),
	Sapphire:  lipgloss.Color("#7dc4e4"),
	Blue:      lipgloss.Color("#8aadf4"),
	Lavender:  lipgloss.Color("#b7bdf8"),

	BorderFocused:   lipgloss.Color("#c6a0f6"),
	BorderUnfocused: lipgloss.Color("#5b6078"),
	StatusOK:        lipgloss.Color("#a6da95"),
	StatusError:     lipgloss.Color("#ed8796"),
	StatusWarning:   lipgloss.Color("#eed49f"),
}
