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
	Version        int                          `json:"version"`
	Theme          string                       `json:"theme"`
	Motion         string                       `json:"motion"`
	Watch          string                       `json:"watch"`
	Interval       time.Duration                `json:"interval"`
	Reconciliation time.Duration                `json:"reconciliation"`
	ShowUntracked  bool                         `json:"show_untracked"`
	ShowIgnored    bool                         `json:"show_ignored"`
	Mouse          bool                         `json:"mouse"`
	Debounce       time.Duration                `json:"debounce"`
	Repositories   RepositoryConfig             `json:"repositories"`
	Remote         RemoteConfig                 `json:"remote"`
	GitHub         GitHubConfig                 `json:"github"`
	Plugins        PluginConfig                 `json:"plugins"`
	Notifications  NotificationConfig           `json:"notifications"`
	Layout         LayoutConfig                 `json:"layout"`
	Diff           DiffConfig                   `json:"diff"`
	ShowCommitTree bool                         `json:"show_commit_tree"`
	CommitTree     CommitTreeConfig             `json:"commit_tree"`
	Keymap         map[string]string            `json:"keymap"`
	Profile        string                       `json:"profile,omitempty"`
	KeymapProfiles map[string]map[string]string `json:"keymap_profiles,omitempty"`
}

type RepositoryConfig struct {
	Roots           []string                 `json:"roots"`
	Groups          map[string][]string      `json:"groups"`
	GroupRefresh    map[string]time.Duration `json:"group_refresh"`
	IgnoreDirs      []string                 `json:"ignore_dirs"`
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

type LayoutConfig struct {
	FilesPercent   int `json:"files_percent"`
	DetailsPercent int `json:"details_percent"`
}

// DiffConfig bounds diff loading and rendering work.
type DiffConfig struct {
	MaxBytes int64 `json:"max_bytes"`
	MaxLines int   `json:"max_lines"`
}

// CommitTreeConfig bounds the optional status-workspace history graph.
type CommitTreeConfig struct {
	MaxCommits int `json:"max_commits"`
}

const DefaultCommitTreeCommits = 100
const MaxCommitTreeCommits = 1000

func Defaults() Config {
	return Config{Version: CurrentVersion, Theme: "auto", Motion: "full", Watch: "auto", Interval: 2 * time.Second, Reconciliation: 30 * time.Second, ShowUntracked: true, Mouse: true, Debounce: 75 * time.Millisecond, Repositories: RepositoryConfig{MaxDepth: 4, MaxRepositories: 256}, Remote: RemoteConfig{PullStrategy: "ff-only", StaleAfter: 30 * time.Minute, Workers: 2}, GitHub: GitHubConfig{TokenEnv: "GITHUB_TOKEN", CacheTTL: 2 * time.Minute}, Plugins: PluginConfig{MaxOutput: 1 << 20}, Layout: LayoutConfig{FilesPercent: 60, DetailsPercent: 40}, Diff: DiffConfig{MaxBytes: 4 << 20, MaxLines: 20_000}, CommitTree: CommitTreeConfig{MaxCommits: DefaultCommitTreeCommits}, Keymap: DefaultKeymap()}
}

func DefaultKeymap() map[string]string {
	return map[string]string{"quit": "q", "help": "?", "status": "1", "branches": "b", "stashes": "s", "history": "l", "remotes": "n", "worktrees": "w", "repositories": "v", "commit": "c", "refresh": "r", "commit_tree": "T", "unpushed": "P", "branch_summary": "B"}
}

// KnownKeymapActions is the non-dangerous action surface that may be rebound.
var KnownKeymapActions = map[string]bool{"quit": true, "help": true, "status": true, "branches": true, "stashes": true, "history": true, "remotes": true, "worktrees": true, "repositories": true, "commit": true, "refresh": true, "commit_tree": true, "unpushed": true, "branch_summary": true}

// EffectiveKeymap merges defaults, the selected profile, and direct overrides.
func EffectiveKeymap(c Config) map[string]string {
	keymap := DefaultKeymap()
	for action, key := range c.KeymapProfiles[c.Profile] {
		keymap[action] = key
	}
	for action, key := range c.Keymap {
		keymap[action] = key
	}
	return keymap
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
	var layoutAdjusted bool
	c, layoutAdjusted = NormalizeLayout(c)
	if layoutAdjusted {
		fmt.Fprintln(os.Stderr, "gitwatch: config: error: layout percentages exceed 100; using 50/50 panel widths")
	}
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

func NormalizeLayout(c Config) (Config, bool) {
	if c.Layout.FilesPercent+c.Layout.DetailsPercent > 100 {
		c.Layout = LayoutConfig{FilesPercent: 50, DetailsPercent: 50}
		return c, true
	}
	return c, false
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
	if c.Repositories.MaxDepth < 0 || c.Repositories.MaxRepositories < 0 || c.Remote.Workers < 0 || c.Plugins.MaxOutput < 0 || c.Diff.MaxBytes <= 0 || c.Diff.MaxLines <= 0 || c.CommitTree.MaxCommits <= 0 || c.CommitTree.MaxCommits > MaxCommitTreeCommits {
		return fmt.Errorf("config limits cannot be negative")
	}
	if c.Layout.FilesPercent <= 0 || c.Layout.DetailsPercent <= 0 || c.Layout.FilesPercent+c.Layout.DetailsPercent != 100 {
		return fmt.Errorf("layout files_percent and details_percent must be positive and sum to 100")
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
	if err := ValidateKeymaps(c); err != nil {
		return err
	}
	return nil
}

// ValidateKeymaps rejects unknown actions, unsafe controls, invalid profiles,
// and collisions before the TUI can start.
func ValidateKeymaps(c Config) error {
	if c.Profile != "" && c.KeymapProfiles[c.Profile] == nil {
		return fmt.Errorf("profile %q is not defined in keymap_profiles", c.Profile)
	}
	for profile, bindings := range c.KeymapProfiles {
		if strings.TrimSpace(profile) == "" {
			return fmt.Errorf("keymap_profiles contains an empty profile name")
		}
		if err := validateBindings("keymap_profiles."+profile, bindings); err != nil {
			return err
		}
	}
	return validateBindings("keymap", c.Keymap)
}

func validateBindings(field string, bindings map[string]string) error {
	seen := make(map[string]string)
	for action, key := range bindings {
		if !KnownKeymapActions[action] {
			return fmt.Errorf("%s.%s: unknown action; destructive actions cannot be remapped", field, action)
		}
		if strings.TrimSpace(key) == "" || len([]rune(key)) > 16 {
			return fmt.Errorf("%s.%s: key sequence must contain 1-16 characters", field, action)
		}
		lower := strings.ToLower(key)
		if lower == "ctrl+c" || lower == "ctrl+z" || lower == "ctrl+\\" {
			return fmt.Errorf("%s.%s: reserved terminal control sequence %q", field, action, key)
		}
		if prior := seen[key]; prior != "" {
			return fmt.Errorf("%s.%s: key %q collides with %s", field, action, key, prior)
		}
		seen[key] = action
	}
	return nil
}

// ResetKeymap returns a copy with custom bindings and profiles removed.
func ResetKeymap(c Config) Config {
	c.Profile, c.KeymapProfiles, c.Keymap = "", nil, DefaultKeymap()
	return c
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
	if v := os.Getenv("GITWATCH_PROFILE"); v != "" {
		c.Profile = v
	}
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
