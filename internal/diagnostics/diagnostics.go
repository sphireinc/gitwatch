// Package diagnostics builds bounded, local-only support information.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/platform"
)

const MaxBundleBytes = 1 << 20

// Report is a stable, sanitized diagnostic summary. It contains metadata only.
type Report struct {
	Schema       int             `json:"schema"`
	Correlation  string          `json:"correlation"`
	Created      string          `json:"created"`
	Version      string          `json:"gitwatch_version"`
	GoVersion    string          `json:"go_version"`
	OS           string          `json:"os"`
	Config       string          `json:"config"`
	Repository   string          `json:"repository,omitempty"`
	GitVersion   string          `json:"git_version,omitempty"`
	Capabilities map[string]bool `json:"capabilities,omitempty"`
	Mode         string          `json:"mode,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// Build creates a report without reading repository files or Git status.
func Build(version string, cfg config.Config, discovery git.Discovery, mode string, cause error) Report {
	report := Report{Schema: 1, Correlation: correlation(), Created: time.Now().UTC().Format(time.RFC3339), Version: version, GoVersion: runtime.Version(), OS: runtime.GOOS + "/" + runtime.GOARCH, Config: "schema-" + fmt.Sprint(cfg.Version), Repository: sanitizePath(discovery.Root), GitVersion: platform.SafeText(discovery.GitVersion), Mode: platform.SafeText(mode)}
	if discovery.Capabilities.Version.Major > 0 {
		report.Capabilities = map[string]bool{"restore": discovery.Capabilities.Restore, "switch": discovery.Capabilities.Switch, "worktree_porcelain": discovery.Capabilities.WorktreePorcelain, "tracking_metadata": discovery.Capabilities.TrackingMetadata}
	}
	if cause != nil {
		report.Error = platform.RedactSecrets(platform.SafeText(cause.Error()))
	}
	return report
}

// WriteBundle writes a private atomic JSON bundle and refuses oversized output.
func WriteBundle(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > MaxBundleBytes {
		return fmt.Errorf("diagnostic bundle exceeds %d bytes", MaxBundleBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gitwatch-diagnostics-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func sanitizePath(path string) string {
	if path == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err == nil {
		path = strings.ReplaceAll(path, cwd, "<working-directory>")
	}
	return platform.SafeText(path)
}

func correlation() string { return fmt.Sprintf("local-%d", time.Now().UnixNano()) }
