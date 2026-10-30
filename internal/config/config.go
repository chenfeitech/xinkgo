// Package config keeps the user configuration Xinkgo stores in its user
// data directory, as JSON, and reads it back on the next start.
//
// The package takes the directory it works in as an argument instead of
// asking the application for it, so its tests need no running app.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the name of the configuration file inside the directory the
// caller passes.
const FileName = "config.json"

// Themes the interface can follow.
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// Config is the configuration the user can change.
type Config struct {
	// Theme is one of ThemeSystem, ThemeLight and ThemeDark.
	Theme string `json:"theme"`
	// LogLevel is the minimum level the application logs: debug, info,
	// warn or error.
	LogLevel string `json:"logLevel"`
}

// Default returns the configuration of a first start.
func Default() Config {
	return Config{
		Theme:    ThemeSystem,
		LogLevel: "info",
	}
}

// Validate reports whether c is a configuration the app can use.
func (c Config) Validate() error {
	switch c.Theme {
	case ThemeSystem, ThemeLight, ThemeDark:
	default:
		return fmt.Errorf("unknown theme %q", c.Theme)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unknown log level %q", c.LogLevel)
	}
	return nil
}

// Load reads the configuration from dir, returning Default when there is
// no file yet. A file that is unreadable or wrong is an error: the app
// must not silently discard what the user set.
func Load(dir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read configuration: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parse configuration: %w", err)
	}
	if cfg.Theme == "" {
		cfg.Theme = Default().Theme
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = Default().LogLevel
	}
	if err := cfg.Validate(); err != nil {
		return Default(), err
	}
	return cfg, nil
}

// Save writes c to dir, replacing the file only once the new contents are
// on disk, so a crash while writing leaves the previous one.
func (c Config) Save(dir string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(dir, FileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	// Windows refuses to rename over an existing file; remove it first.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace configuration: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}
