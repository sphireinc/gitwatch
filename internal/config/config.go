package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Theme          string        `json:"theme"`
	Motion         string        `json:"motion"`
	Watch          string        `json:"watch"`
	Interval       time.Duration `json:"interval"`
	Reconciliation time.Duration `json:"reconciliation"`
	ShowUntracked  bool          `json:"show_untracked"`
	ShowIgnored    bool          `json:"show_ignored"`
	Mouse          bool          `json:"mouse"`
	Debounce       time.Duration `json:"debounce"`
}

func Defaults() Config {
	return Config{Theme: "auto", Motion: "full", Watch: "auto", Interval: 2 * time.Second, Reconciliation: 30 * time.Second, ShowUntracked: true, Mouse: true, Debounce: 75 * time.Millisecond}
}
func Path() (string, error) {
	if p := os.Getenv("GITWATCH_CONFIG"); p != "" {
		return p, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "gitwatch", "config.json"), nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "gitwatch", "config.json"), nil
	}
	return "", fmt.Errorf("cannot resolve config directory")
}
func Load(path string) (Config, error) {
	c := Defaults()
	if path == "" {
		var err error
		path, err = Path()
		if err != nil {
			return c, err
		}
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return applyEnv(c), nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("invalid config %s: %w", path, err)
	}
	c = applyEnv(c)
	if err := Validate(c); err != nil {
		return c, err
	}
	return c, nil
}
func ApplyCLI(c Config, theme, motion, watch string, interval time.Duration) Config {
	if theme != "" {
		c.Theme = theme
	}
	if motion != "" {
		c.Motion = motion
	}
	if watch != "" {
		c.Watch = watch
	}
	if interval > 0 {
		c.Interval = interval
	}
	return c
}
func Validate(c Config) error {
	if c.Theme != "auto" && c.Theme != "dark" && c.Theme != "light" && c.Theme != "high-contrast" {
		return fmt.Errorf("invalid theme %q", c.Theme)
	}
	if c.Motion != "full" && c.Motion != "reduced" && c.Motion != "off" {
		return fmt.Errorf("invalid motion %q", c.Motion)
	}
	if c.Watch != "auto" && c.Watch != "fs" && c.Watch != "poll" {
		return fmt.Errorf("invalid watch mode %q", c.Watch)
	}
	if c.Interval <= 0 || c.Debounce < 0 {
		return fmt.Errorf("interval must be positive and debounce non-negative")
	}
	return nil
}
func Inspect(c Config) ([]byte, error) { return json.MarshalIndent(c, "", "  ") }
func applyEnv(c Config) Config {
	if v := os.Getenv("GITWATCH_THEME"); v != "" {
		c.Theme = v
	}
	if v := os.Getenv("GITWATCH_MOTION"); v != "" {
		c.Motion = v
	}
	if v := os.Getenv("GITWATCH_WATCH"); v != "" {
		c.Watch = v
	}
	if v := os.Getenv("GITWATCH_INTERVAL"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Interval = time.Duration(n) * time.Second
		}
	}
	return c
}
