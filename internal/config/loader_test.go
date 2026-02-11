package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()

	if got.Theme != "catppuccin-mocha" {
		t.Fatalf("Theme = %q, want catppuccin-mocha", got.Theme)
	}
	if !got.VimMode {
		t.Fatal("VimMode = false, want true")
	}
	if got.DefaultTimeout != 30*time.Second {
		t.Fatalf("DefaultTimeout = %s, want 30s", got.DefaultTimeout)
	}
	if got.ScriptTimeout != 5*time.Second {
		t.Fatalf("ScriptTimeout = %s, want 5s", got.ScriptTimeout)
	}
}

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := Load()
	want := DefaultConfig()

	if got != want {
		t.Fatalf("Load() = %#v, want defaults %#v", got, want)
	}
}

func TestLoadReadsConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "gottp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	configYAML := "theme: nord\nvim_mode: false\ndefault_timeout: 42s\neditor: nvim\npager: less -R\nscript_timeout: 9s\n"
	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got := Load()

	if got.Theme != "nord" {
		t.Fatalf("Theme = %q, want nord", got.Theme)
	}
	if got.VimMode {
		t.Fatal("VimMode = true, want false")
	}
	if got.DefaultTimeout != 42*time.Second {
		t.Fatalf("DefaultTimeout = %s, want 42s", got.DefaultTimeout)
	}
	if got.Editor != "nvim" {
		t.Fatalf("Editor = %q, want nvim", got.Editor)
	}
	if got.Pager != "less -R" {
		t.Fatalf("Pager = %q, want less -R", got.Pager)
	}
	if got.ScriptTimeout != 9*time.Second {
		t.Fatalf("ScriptTimeout = %s, want 9s", got.ScriptTimeout)
	}
}

func TestLoadMergesPartialConfigWithDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "gottp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte("theme: gruvbox\n"), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got := Load()
	want := DefaultConfig()
	want.Theme = "gruvbox"

	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadInvalidYAMLKeepsDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "gottp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte("theme: [\n"), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got := Load()
	want := DefaultConfig()

	if got != want {
		t.Fatalf("Load() = %#v, want defaults %#v", got, want)
	}
}
