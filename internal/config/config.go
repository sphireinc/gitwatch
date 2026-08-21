package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CurrentVersion is the configuration schema consumed by the version-2 loader.
// Older unversioned and v1 files migrate in memory; newer versions are
// rejected until this contract is intentionally advanced.
const CurrentVersion = 2

type Config struct {
	Version        int                `json:"version"`
	Theme          string             `json:"theme"`
	Motion         string             `json:"motion"`
	Watch          string             `json:"watch"`
	Interval       time.Duration      `json:"interval"`
	Reconciliation time.Duration      `json:"reconciliation"`
	ShowUntracked  bool               `json:"show_untracked"`
	ShowIgnored    bool               `json:"show_ignored"`
	Mouse          bool               `json:"mouse"`
	Debounce       time.Duration      `json:"debounce"`
	Repositories   RepositoryConfig   `json:"repositories"`
	Remote         RemoteConfig       `json:"remote"`
	GitHub         GitHubConfig       `json:"github"`
	Plugins        PluginConfig       `json:"plugins"`
	Notifications  NotificationConfig `json:"notifications"`
	Keymap         map[string]string  `json:"keymap"`
}

type RepositoryConfig struct {
	Roots           []string                 `json:"roots"`
	Groups          map[string][]string      `json:"groups"`
	GroupRefresh    map[string]time.Duration `json:"group_refresh"`
	MaxDepth        int                      `json:"max_depth"`
	MaxRepositories int                      `json:"max_repositories"`
}

type RemoteConfig struct {
	PullStrategy string        `json:"pull_strategy"`
	StaleAfter   time.Duration `json:"stale_after"`
	Workers      int           `json:"workers"`
}

type GitHubConfig struct {
	Enabled  bool          `json:"enabled"`
	TokenEnv string        `json:"token_env"`
	CacheTTL time.Duration `json:"cache_ttl"`
}

type PluginConfig struct {
	Enabled     bool     `json:"enabled"`
	Directories []string `json:"directories"`
	MaxOutput   int64    `json:"max_output"`
}

type NotificationConfig struct {
	Quiet bool `json:"quiet"`
}

func Defaults() Config {
	return Config{Version: CurrentVersion, Theme: "auto", Motion: "full", Watch: "auto", Interval: 2 * time.Second, Reconciliation: 30 * time.Second, ShowUntracked: true, Mouse: true, Debounce: 75 * time.Millisecond, Repositories: RepositoryConfig{MaxDepth: 4, MaxRepositories: 256}, Remote: RemoteConfig{PullStrategy: "ff-only", StaleAfter: 30 * time.Minute, Workers: 2}, GitHub: GitHubConfig{TokenEnv: "GITHUB_TOKEN", CacheTTL: 2 * time.Minute}, Plugins: PluginConfig{MaxOutput: 1 << 20}, Keymap: DefaultKeymap()}
}

func DefaultKeymap() map[string]string {
	return map[string]string{"quit": "q", "help": "?", "status": "1", "branches": "b", "stashes": "s", "history": "l", "remotes": "n", "worktrees": "w", "repositories": "v", "commit": "c", "refresh": "r"}
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
	var envelope struct {
		Version *int `json:"version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return c, fmt.Errorf("invalid config %s: %w", path, err)
	}
	if envelope.Version != nil && *envelope.Version > c.Version {
		return c, fmt.Errorf("unsupported config version %d", *envelope.Version)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("invalid config %s: %w", path, err)
	}
	// Version 1 and files without a version used the same scalar settings;
	// Unmarshal over schema-version-2 defaults supplies newly introduced module defaults.
	c.Version = CurrentVersion
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
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Repositories.MaxDepth < 0 || c.Repositories.MaxRepositories < 0 || c.Remote.Workers < 0 || c.Plugins.MaxOutput < 0 {
		return fmt.Errorf("config limits cannot be negative")
	}
	for group, duration := range c.Repositories.GroupRefresh {
		if strings.TrimSpace(group) == "" || duration < 0 {
			return fmt.Errorf("invalid group refresh policy %q", group)
		}
	}
	if c.Remote.PullStrategy != "merge" && c.Remote.PullStrategy != "rebase" && c.Remote.PullStrategy != "ff-only" {
		return fmt.Errorf("invalid pull strategy %q", c.Remote.PullStrategy)
	}
	if c.Remote.StaleAfter < 0 || c.GitHub.CacheTTL < 0 {
		return fmt.Errorf("config durations cannot be negative")
	}
	if collisions := BindingCollisions(c.Keymap); len(collisions) > 0 {
		return fmt.Errorf("key binding collision: %s", collisions[0])
	}
	return nil
}

func BindingCollisions(bindings map[string]string) []string {
	owners := make(map[string]string)
	var collisions []string
	for action, key := range bindings {
		if previous, ok := owners[key]; ok && previous != action {
			collisions = append(collisions, key+" ("+previous+", "+action+")")
		}
		owners[key] = action
	}
	sort.Strings(collisions)
	return collisions
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
