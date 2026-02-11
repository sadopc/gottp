package config

import (
	"time"

	gotls "github.com/serdar/gottp/internal/core/tls"
)

// Config holds the application configuration.
type Config struct {
	Theme          string        `yaml:"theme"`
	VimMode        bool          `yaml:"vim_mode"`
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	Editor         string        `yaml:"editor"`
	Pager          string        `yaml:"pager"`
	ScriptTimeout  time.Duration `yaml:"script_timeout"`
	ProxyURL       string        `yaml:"proxy_url,omitempty"`
	NoProxy        string        `yaml:"no_proxy,omitempty"`
	TLS            gotls.Config  `yaml:"tls,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Theme:          "catppuccin-mocha",
		VimMode:        true,
		DefaultTimeout: 30 * time.Second,
		Editor:         "",
		Pager:          "",
		ScriptTimeout:  5 * time.Second,
	}
}
