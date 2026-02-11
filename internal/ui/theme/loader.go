package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// yamlTheme is the YAML representation of a theme.
type yamlTheme struct {
	Name    string `yaml:"name"`
	Base    string `yaml:"base"`
	Mantle  string `yaml:"mantle"`
	Crust   string `yaml:"crust"`
	Surface string `yaml:"surface"`
	Overlay string `yaml:"overlay"`

	Text    string `yaml:"text"`
	Subtext string `yaml:"subtext"`
	Muted   string `yaml:"muted"`

	Rosewater string `yaml:"rosewater"`
	Flamingo  string `yaml:"flamingo"`
	Pink      string `yaml:"pink"`
	Mauve     string `yaml:"mauve"`
	Red       string `yaml:"red"`
	Maroon    string `yaml:"maroon"`
	Peach     string `yaml:"peach"`
	Yellow    string `yaml:"yellow"`
	Green     string `yaml:"green"`
	Teal      string `yaml:"teal"`
	Sky       string `yaml:"sky"`
	Sapphire  string `yaml:"sapphire"`
	Blue      string `yaml:"blue"`
	Lavender  string `yaml:"lavender"`

	BorderFocused   string `yaml:"border_focused"`
	BorderUnfocused string `yaml:"border_unfocused"`
	StatusOK        string `yaml:"status_ok"`
	StatusError     string `yaml:"status_error"`
	StatusWarning   string `yaml:"status_warning"`
}

// LoadCustomTheme loads a theme from a YAML file.
func LoadCustomTheme(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("reading theme file: %w", err)
	}

	var yt yamlTheme
	if err := yaml.Unmarshal(data, &yt); err != nil {
		return Theme{}, fmt.Errorf("parsing theme YAML: %w", err)
	}

	if yt.Name == "" {
		base := filepath.Base(path)
		yt.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return Theme{
		Name:            yt.Name,
		Base:            lipgloss.Color(yt.Base),
		Mantle:          lipgloss.Color(yt.Mantle),
		Crust:           lipgloss.Color(yt.Crust),
		Surface:         lipgloss.Color(yt.Surface),
		Overlay:         lipgloss.Color(yt.Overlay),
		Text:            lipgloss.Color(yt.Text),
		Subtext:         lipgloss.Color(yt.Subtext),
		Muted:           lipgloss.Color(yt.Muted),
		Rosewater:       lipgloss.Color(yt.Rosewater),
		Flamingo:        lipgloss.Color(yt.Flamingo),
		Pink:            lipgloss.Color(yt.Pink),
		Mauve:           lipgloss.Color(yt.Mauve),
		Red:             lipgloss.Color(yt.Red),
		Maroon:          lipgloss.Color(yt.Maroon),
		Peach:           lipgloss.Color(yt.Peach),
		Yellow:          lipgloss.Color(yt.Yellow),
		Green:           lipgloss.Color(yt.Green),
		Teal:            lipgloss.Color(yt.Teal),
		Sky:             lipgloss.Color(yt.Sky),
		Sapphire:        lipgloss.Color(yt.Sapphire),
		Blue:            lipgloss.Color(yt.Blue),
		Lavender:        lipgloss.Color(yt.Lavender),
		BorderFocused:   lipgloss.Color(yt.BorderFocused),
		BorderUnfocused: lipgloss.Color(yt.BorderUnfocused),
		StatusOK:        lipgloss.Color(yt.StatusOK),
		StatusError:     lipgloss.Color(yt.StatusError),
		StatusWarning:   lipgloss.Color(yt.StatusWarning),
	}, nil
}

// LoadCustomThemes loads all YAML themes from a directory.
func LoadCustomThemes(dir string) map[string]Theme {
	themes := make(map[string]Theme)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return themes
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		t, err := LoadCustomTheme(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		themes[normalizeKey(t.Name)] = t
	}
	return themes
}
