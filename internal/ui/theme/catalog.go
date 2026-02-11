package theme

import (
	"strings"
)

// Catalog maps theme names to themes.
var Catalog = map[string]Theme{}

func init() {
	register(CatppuccinMocha)
	register(CatppuccinLatte)
	register(CatppuccinFrappe)
	register(CatppuccinMacchiato)
	register(Nord)
	register(Dracula)
	register(GruvboxDark)
	register(TokyoNight)
}

func register(t Theme) {
	Catalog[normalizeKey(t.Name)] = t
}

// Get returns a theme by name.
func Get(name string) (Theme, bool) {
	t, ok := Catalog[normalizeKey(name)]
	return t, ok
}

// Names returns all registered theme names.
func Names() []string {
	var names []string
	for _, t := range Catalog {
		names = append(names, t.Name)
	}
	return names
}

func normalizeKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}
