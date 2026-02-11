package theme

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// CatppuccinMocha is the default dark theme.
var CatppuccinMocha = Theme{
	Name:    "Catppuccin Mocha",
	Base:    lipgloss.Color("#1e1e2e"),
	Mantle:  lipgloss.Color("#181825"),
	Crust:   lipgloss.Color("#11111b"),
	Surface: lipgloss.Color("#313244"),
	Overlay: lipgloss.Color("#45475a"),

	Text:    lipgloss.Color("#cdd6f4"),
	Subtext: lipgloss.Color("#a6adc8"),
	Muted:   lipgloss.Color("#585b70"),

	Rosewater: lipgloss.Color("#f5e0dc"),
	Flamingo:  lipgloss.Color("#f2cdcd"),
	Pink:      lipgloss.Color("#f5c2e7"),
	Mauve:     lipgloss.Color("#cba6f7"),
	Red:       lipgloss.Color("#f38ba8"),
	Maroon:    lipgloss.Color("#eba0ac"),
	Peach:     lipgloss.Color("#fab387"),
	Yellow:    lipgloss.Color("#f9e2af"),
	Green:     lipgloss.Color("#a6e3a1"),
	Teal:      lipgloss.Color("#94e2d5"),
	Sky:       lipgloss.Color("#89dceb"),
	Sapphire:  lipgloss.Color("#74c7ec"),
	Blue:      lipgloss.Color("#89b4fa"),
	Lavender:  lipgloss.Color("#b4befe"),

	BorderFocused:   lipgloss.Color("#cba6f7"),
	BorderUnfocused: lipgloss.Color("#585b70"),
	StatusOK:        lipgloss.Color("#a6e3a1"),
	StatusError:     lipgloss.Color("#f38ba8"),
	StatusWarning:   lipgloss.Color("#f9e2af"),
}

// Default returns the default theme.
func Default() Theme {
	return CatppuccinMocha
}

// Resolve looks up a theme by name: catalog -> custom themes -> fallback to Mocha.
func Resolve(name string) Theme {
	// Try built-in catalog
	if t, ok := Get(name); ok {
		return t
	}

	// Try custom themes from ~/.config/gottp/themes/
	home, err := os.UserHomeDir()
	if err == nil {
		customDir := filepath.Join(home, ".config", "gottp", "themes")
		customs := LoadCustomThemes(customDir)
		if t, ok := customs[normalizeKey(name)]; ok {
			return t
		}
	}

	return CatppuccinMocha
}
