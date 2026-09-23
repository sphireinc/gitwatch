package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/customcmd"
	"github.com/sphireinc/git-watch/internal/platform"
)

// CurrentVersion is the configuration schema consumed by the version-3 loader.
// Older unversioned, v1, and v2 files migrate in memory; newer versions are
// rejected until this contract is intentionally advanced.
const CurrentVersion = 3

type Config struct {
	Version           int                          `json:"version"`
	Theme             string                       `json:"theme"`
	Motion            string                       `json:"motion"`
	Watch             string                       `json:"watch"`
	Interval          time.Duration                `json:"interval"`
	Reconciliation    time.Duration                `json:"reconciliation"`
	ShowUntracked     bool                         `json:"show_untracked"`
	ShowIgnored       bool                         `json:"show_ignored"`
	Mouse             bool                         `json:"mouse"`
	Debounce          time.Duration                `json:"debounce"`
	Repositories      RepositoryConfig             `json:"repositories"`
	Remote            RemoteConfig                 `json:"remote"`
	GitHub            GitHubConfig                 `json:"github"`
	Plugins           PluginConfig                 `json:"plugins"`
	Notifications     NotificationConfig           `json:"notifications"`
	Layout            LayoutConfig                 `json:"layout"`
	Diff              DiffConfig                   `json:"diff"`
	Tools             ToolsConfig                  `json:"tools"`
	CustomCommands    []customcmd.Definition       `json:"custom_commands,omitempty"`
	GitignoreMaxBytes int64                        `json:"gitignore_max_bytes"`
	ShowCommitTree    bool                         `json:"show_commit_tree"`
	CommitTree        CommitTreeConfig             `json:"commit_tree"`
	Workspace         WorkspaceConfig              `json:"workspace"`
	Visuals           VisualizationConfig          `json:"visuals"`
	Keymap            map[string]string            `json:"keymap"`
	Profile           string                       `json:"profile,omitempty"`
	KeymapProfiles    map[string]map[string]string `json:"keymap_profiles,omitempty"`
}

type RepositoryConfig struct {
	Roots           []string                 `json:"roots"`
	Groups          map[string][]string      `json:"groups"`
	GroupRefresh    map[string]time.Duration `json:"group_refresh"`
	GroupAutoFetch  map[string]time.Duration `json:"group_auto_fetch"`
	IgnoreDirs      []string                 `json:"ignore_dirs"`
	MaxDepth        int                      `json:"max_depth"`
	MaxRepositories int                      `json:"max_repositories"`
}

type RemoteConfig struct {
	PullStrategy        string                      `json:"pull_strategy"`
	StaleAfter          time.Duration               `json:"stale_after"`
	Workers             int                         `json:"workers"`
	AutoFetch           bool                        `json:"auto_fetch"`
	AutoFetchInterval   time.Duration               `json:"auto_fetch_interval"`
	AutoFetchJitter     time.Duration               `json:"auto_fetch_jitter"`
	AutoFetchBackoff    time.Duration               `json:"auto_fetch_backoff"`
	AutoFetchBackoffMax time.Duration               `json:"auto_fetch_backoff_max"`
	AutoFetchProfiles   map[string]AutoFetchProfile `json:"auto_fetch_profiles,omitempty"`
}

type AutoFetchProfile struct {
	Enabled    bool          `json:"enabled"`
	Interval   time.Duration `json:"interval"`
	Jitter     time.Duration `json:"jitter"`
	Backoff    time.Duration `json:"backoff"`
	BackoffMax time.Duration `json:"backoff_max"`
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

// ToolConfig describes one executable and its already-tokenized argv
// template. Supported placeholders are interpreted by the platform boundary;
// no shell command string is accepted.
type ToolConfig struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

// ToolsConfig contains optional editor, file-opener, and diff-tool commands.
type ToolsConfig struct {
	Editor   ToolConfig `json:"editor"`
	Opener   ToolConfig `json:"opener"`
	Difftool ToolConfig `json:"difftool"`
}

// CommitTreeConfig bounds the optional status-workspace history graph.
type CommitTreeConfig struct {
	MaxCommits int `json:"max_commits"`
}

// WorkspaceConfig bounds in-memory navigation indexes and terminal overscan.
type WorkspaceConfig struct {
	PaletteMaxResults int `json:"palette_max_results"`
	StatusOverscan    int `json:"status_overscan"`
}

// VisualizationConfig controls optional dense dashboard indicators.
type VisualizationConfig struct {
	Enabled         bool `json:"enabled"`
	ActivityBuckets int  `json:"activity_buckets"`
}

const DefaultCommitTreeCommits = 100
const MaxCommitTreeCommits = 1000

func Defaults() Config {
	return Config{Version: CurrentVersion, Theme: "auto", Motion: "full", Watch: "auto", Interval: 2 * time.Second, Reconciliation: 30 * time.Second, ShowUntracked: true, Mouse: true, Debounce: 75 * time.Millisecond, Repositories: RepositoryConfig{MaxDepth: 4, MaxRepositories: 256}, Remote: RemoteConfig{PullStrategy: "ff-only", StaleAfter: 30 * time.Minute, Workers: 2, AutoFetchInterval: 30 * time.Minute, AutoFetchJitter: 30 * time.Second, AutoFetchBackoff: time.Minute, AutoFetchBackoffMax: 30 * time.Minute}, GitHub: GitHubConfig{TokenEnv: "GITHUB_TOKEN", CacheTTL: 2 * time.Minute}, Plugins: PluginConfig{MaxOutput: 1 << 20}, Layout: LayoutConfig{FilesPercent: 60, DetailsPercent: 40}, Diff: DiffConfig{MaxBytes: 4 << 20, MaxLines: 20_000}, GitignoreMaxBytes: 8 << 20, CommitTree: CommitTreeConfig{MaxCommits: DefaultCommitTreeCommits}, Workspace: WorkspaceConfig{PaletteMaxResults: 200, StatusOverscan: 4}, Visuals: VisualizationConfig{Enabled: true, ActivityBuckets: 8}, Keymap: DefaultKeymap()}
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
	if c.Repositories.MaxDepth < 0 || c.Repositories.MaxRepositories < 0 || c.Remote.Workers < 0 || c.Plugins.MaxOutput < 0 || c.Diff.MaxBytes <= 0 || c.Diff.MaxLines <= 0 || c.GitignoreMaxBytes <= 0 || c.CommitTree.MaxCommits <= 0 || c.CommitTree.MaxCommits > MaxCommitTreeCommits {
		return fmt.Errorf("config limits cannot be negative")
	}
	if c.Workspace.PaletteMaxResults < 1 || c.Workspace.PaletteMaxResults > 2000 {
		return fmt.Errorf("workspace.palette_max_results must be between 1 and 2000")
	}
	if c.Workspace.StatusOverscan < 0 || c.Workspace.StatusOverscan > 32 {
		return fmt.Errorf("workspace.status_overscan must be between 0 and 32")
	}
	if c.Visuals.ActivityBuckets < 1 || c.Visuals.ActivityBuckets > 32 {
		return fmt.Errorf("visuals.activity_buckets must be between 1 and 32")
	}
	if c.Layout.FilesPercent <= 0 || c.Layout.DetailsPercent <= 0 || c.Layout.FilesPercent+c.Layout.DetailsPercent != 100 {
		return fmt.Errorf("layout files_percent and details_percent must be positive and sum to 100")
	}
	for group, duration := range c.Repositories.GroupRefresh {
		if strings.TrimSpace(group) == "" || duration < 0 {
			return fmt.Errorf("invalid group refresh policy %q", group)
		}
	}
	for group, duration := range c.Repositories.GroupAutoFetch {
		if strings.TrimSpace(group) == "" || duration < 0 {
			return fmt.Errorf("invalid group auto-fetch policy %q", group)
		}
	}
	for profile, policy := range c.Remote.AutoFetchProfiles {
		if strings.TrimSpace(profile) == "" || policy.Interval < 0 || policy.Jitter < 0 || policy.Backoff < 0 || policy.BackoffMax < 0 {
			return fmt.Errorf("invalid auto-fetch profile %q", profile)
		}
		if policy.Enabled && policy.Interval <= 0 {
			return fmt.Errorf("auto-fetch interval must be positive for profile %q", profile)
		}
	}
	if c.Remote.PullStrategy != "merge" && c.Remote.PullStrategy != "rebase" && c.Remote.PullStrategy != "ff-only" {
		return fmt.Errorf("invalid pull strategy %q", c.Remote.PullStrategy)
	}
	if c.Remote.StaleAfter < 0 || c.Remote.AutoFetchInterval < 0 || c.Remote.AutoFetchJitter < 0 || c.Remote.AutoFetchBackoff < 0 || c.Remote.AutoFetchBackoffMax < 0 || c.GitHub.CacheTTL < 0 {
		return fmt.Errorf("config durations cannot be negative")
	}
	if c.Remote.AutoFetch && c.Remote.AutoFetchInterval <= 0 {
		return fmt.Errorf("auto-fetch interval must be positive when enabled")
	}
	if collisions := BindingCollisions(c.Keymap); len(collisions) > 0 {
		return fmt.Errorf("key binding collision: %s", collisions[0])
	}
	if err := ValidateKeymaps(c); err != nil {
		return err
	}
	customBindings := make(map[string]string)
	for _, command := range c.CustomCommands {
		if err := command.Validate(); err != nil {
			return err
		}
		if command.Binding != "" {
			if previous := customBindings[command.Binding]; previous != "" {
				return fmt.Errorf("custom command binding %q collides with %s", command.Binding, previous)
			}
			customBindings[command.Binding] = command.Name
		}
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

// Inspect returns effective configuration with secret-like fields and inline
// credential values redacted. The token environment-variable name remains
// visible because it is configuration metadata, not a credential.
func Inspect(c Config) ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	redactInspection(value)
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

func redactInspection(value any) {
	switch item := value.(type) {
	case map[string]any:
		for name, child := range item {
			lower := strings.ToLower(name)
			if strings.HasSuffix(lower, "_env") || lower == "token_env" {
				redactInspection(child)
				continue
			}
			if strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
				item[name] = "<redacted>"
				continue
			}
			redactInspection(child)
		}
	case []any:
		redactNext := false
		for index, child := range item {
			if redactNext {
				if _, ok := child.(string); ok {
					item[index] = "<redacted>"
				}
				redactNext = false
				continue
			}
			if text, ok := child.(string); ok && looksLikeInlineCredential(text) {
				item[index] = "<redacted>"
				continue
			}
			if text, ok := child.(string); ok {
				item[index] = platform.RedactSecrets(text)
				if isCredentialFlag(text) {
					redactNext = true
				}
				continue
			}
			redactInspection(child)
		}
	}
}

func looksLikeInlineCredential(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"token=", "token:", "password=", "password:", "secret=", "secret:", "authorization: bearer ", "authorization:bearer ", "basic "} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isCredentialFlag(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "--token", "--password", "--passwd", "--secret", "--credential", "--authorization", "-p":
		return true
	default:
		return false
	}
}
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
