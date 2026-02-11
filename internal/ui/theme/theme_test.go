package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeKey(t *testing.T) {
	got := normalizeKey("  Catppuccin Mocha  ")
	if got != "catppuccin-mocha" {
		t.Fatalf("normalizeKey() = %q, want catppuccin-mocha", got)
	}
}

func TestGetBuiltInTheme(t *testing.T) {
	got, ok := Get("  catppuccin mocha ")
	if !ok {
		t.Fatal("expected built-in theme to be found")
	}
	if got.Name != "Catppuccin Mocha" {
		t.Fatalf("theme name = %q, want Catppuccin Mocha", got.Name)
	}
}

func TestNamesIncludesBuiltIns(t *testing.T) {
	names := Names()
	if len(names) < 8 {
		t.Fatalf("expected at least 8 built-in themes, got %d", len(names))
	}

	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}

	for _, want := range []string{"Catppuccin Mocha", "Nord", "Tokyo Night"} {
		if !have[want] {
			t.Fatalf("theme %q not found in Names()", want)
		}
	}
}

func TestResolveBuiltInTheme(t *testing.T) {
	got := Resolve("dracula")
	if got.Name != "Dracula" {
		t.Fatalf("Resolve(dracula) returned %q, want Dracula", got.Name)
	}
}

func TestResolveCustomThemeFromHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	themesDir := filepath.Join(home, ".config", "gottp", "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	yaml := "name: Ocean Breeze\nbase: \"#001122\"\ntext: \"#ffffff\"\n"
	path := filepath.Join(themesDir, "ocean-breeze.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got := Resolve("ocean breeze")
	if got.Name != "Ocean Breeze" {
		t.Fatalf("Resolve(custom) name = %q, want Ocean Breeze", got.Name)
	}
	if got.Base != "#001122" {
		t.Fatalf("Resolve(custom) base = %q, want #001122", got.Base)
	}
}

func TestResolveFallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := Resolve("not-a-real-theme")
	if got.Name != CatppuccinMocha.Name {
		t.Fatalf("Resolve(unknown) name = %q, want %q", got.Name, CatppuccinMocha.Name)
	}
}

func TestLoadCustomThemeUsesFilenameWhenNameMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-theme.yaml")

	yaml := "base: \"#010203\"\ntext: \"#eeeeee\"\n"
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got, err := LoadCustomTheme(path)
	if err != nil {
		t.Fatalf("LoadCustomTheme() failed: %v", err)
	}
	if got.Name != "my-theme" {
		t.Fatalf("Name = %q, want my-theme", got.Name)
	}
}

func TestLoadCustomThemeInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.yaml")

	if err := os.WriteFile(path, []byte("name: [\n"), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	if _, err := LoadCustomTheme(path); err == nil {
		t.Fatal("expected parsing error for invalid yaml")
	}
}

func TestLoadCustomThemesSkipsInvalidFiles(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "forest.yaml")
	invalid := filepath.Join(dir, "broken.yaml")
	nonYAML := filepath.Join(dir, "readme.txt")

	if err := os.WriteFile(valid, []byte("name: Forest\nbase: \"#102030\"\n"), 0644); err != nil {
		t.Fatalf("WriteFile(valid) failed: %v", err)
	}
	if err := os.WriteFile(invalid, []byte("name: [\n"), 0644); err != nil {
		t.Fatalf("WriteFile(invalid) failed: %v", err)
	}
	if err := os.WriteFile(nonYAML, []byte("ignore me"), 0644); err != nil {
		t.Fatalf("WriteFile(nonYAML) failed: %v", err)
	}

	themes := LoadCustomThemes(dir)
	if len(themes) != 1 {
		t.Fatalf("LoadCustomThemes() loaded %d themes, want 1", len(themes))
	}

	if got, ok := themes[normalizeKey("Forest")]; !ok || got.Name != "Forest" {
		t.Fatalf("expected Forest theme, got %#v (ok=%v)", got, ok)
	}
}

func TestThemeMethodColor(t *testing.T) {
	theme := Default()

	if got := theme.MethodColor("GET"); got != theme.Green {
		t.Fatalf("GET color = %q, want %q", got, theme.Green)
	}
	if got := theme.MethodColor("POST"); got != theme.Yellow {
		t.Fatalf("POST color = %q, want %q", got, theme.Yellow)
	}
	if got := theme.MethodColor("UNKNOWN"); got != theme.Text {
		t.Fatalf("UNKNOWN color = %q, want %q", got, theme.Text)
	}
}

func TestThemeStatusColor(t *testing.T) {
	theme := Default()

	if got := theme.StatusColor(204); got != theme.Green {
		t.Fatalf("2xx color = %q, want %q", got, theme.Green)
	}
	if got := theme.StatusColor(302); got != theme.Blue {
		t.Fatalf("3xx color = %q, want %q", got, theme.Blue)
	}
	if got := theme.StatusColor(404); got != theme.Yellow {
		t.Fatalf("4xx color = %q, want %q", got, theme.Yellow)
	}
	if got := theme.StatusColor(500); got != theme.Red {
		t.Fatalf("5xx color = %q, want %q", got, theme.Red)
	}
	if got := theme.StatusColor(100); got != theme.Text {
		t.Fatalf("default color = %q, want %q", got, theme.Text)
	}
}
