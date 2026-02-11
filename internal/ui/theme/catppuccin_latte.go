package theme

import "github.com/charmbracelet/lipgloss"

var CatppuccinLatte = Theme{
	Name:    "Catppuccin Latte",
	Base:    lipgloss.Color("#eff1f5"),
	Mantle:  lipgloss.Color("#e6e9ef"),
	Crust:   lipgloss.Color("#dce0e8"),
	Surface: lipgloss.Color("#ccd0da"),
	Overlay: lipgloss.Color("#9ca0b0"),

	Text:    lipgloss.Color("#4c4f69"),
	Subtext: lipgloss.Color("#6c6f85"),
	Muted:   lipgloss.Color("#8c8fa1"),

	Rosewater: lipgloss.Color("#dc8a78"),
	Flamingo:  lipgloss.Color("#dd7878"),
	Pink:      lipgloss.Color("#ea76cb"),
	Mauve:     lipgloss.Color("#8839ef"),
	Red:       lipgloss.Color("#d20f39"),
	Maroon:    lipgloss.Color("#e64553"),
	Peach:     lipgloss.Color("#fe640b"),
	Yellow:    lipgloss.Color("#df8e1d"),
	Green:     lipgloss.Color("#40a02b"),
	Teal:      lipgloss.Color("#179299"),
	Sky:       lipgloss.Color("#04a5e5"),
	Sapphire:  lipgloss.Color("#209fb5"),
	Blue:      lipgloss.Color("#1e66f5"),
	Lavender:  lipgloss.Color("#7287fd"),

	BorderFocused:   lipgloss.Color("#8839ef"),
	BorderUnfocused: lipgloss.Color("#8c8fa1"),
	StatusOK:        lipgloss.Color("#40a02b"),
	StatusError:     lipgloss.Color("#d20f39"),
	StatusWarning:   lipgloss.Color("#df8e1d"),
}
