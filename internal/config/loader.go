package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from ~/.config/gottp/config.yaml.
func Load() Config {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".config", "gottp", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	return cfg
}
