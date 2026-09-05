// Package submodules loads bounded, repository-scoped submodule health.
package submodules

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

const (
	defaultMaxModules = 128
	defaultMaxDepth   = 8
	defaultMaxOutput  = 1 << 20
)

// Limits bounds work performed while loading nested submodules.
type Limits struct {
	MaxModules     int
	MaxDepth       int
	MaxOutputBytes int
}

func (l Limits) normalized() Limits {
	if l.MaxModules <= 0 {
		l.MaxModules = defaultMaxModules
	}
	if l.MaxDepth <= 0 {
		l.MaxDepth = defaultMaxDepth
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = defaultMaxOutput
	}
	return l
}

// State describes the observable health of a configured submodule.
type State string

const (
	StateClean         State = "clean"
	StateDirty         State = "dirty"
	StateUninitialized State = "uninitialized"
	StateDetached      State = "detached"
	StateMissing       State = "missing"
	StateDiverged      State = "diverged"
	StateUnknown       State = "unknown"
)

// Module is a bounded projection of one configured submodule.
type Module struct {
	Name             string
	Path             string
	URL              string
	RecordedCommit   string
	CheckedOutCommit string
	State            State
	Depth            int
	Divergence       int
}

// Snapshot summarizes submodules without recursively loading unlimited repositories.
type Snapshot struct {
	Repository string
	Modules    []Module
	Truncated  bool
	MaxDepth   int
}

// LoadRequest identifies the repository and bounds for one read-only load.
type LoadRequest struct {
	Repository string
	Limits     Limits
}

// Load reads .gitmodules through Git's config parser and status through the
// machine-safe Git boundary. It never accepts or builds a shell command.
func Load(ctx context.Context, runner git.Runner, request LoadRequest) (Snapshot, error) {
	limits := request.Limits.normalized()
	if request.Repository == "" {
		return Snapshot{}, errors.New("submodule repository is required")
	}
	snapshot := Snapshot{Repository: request.Repository, MaxDepth: limits.MaxDepth}
	configResult, err := runner.RunBounded(ctx, limits.MaxOutputBytes, "config", "-z", "-f", ".gitmodules", "--get-regexp", `^submodule\..*`)
	if err != nil {
		if _, statErr := os.Stat(filepath.Join(request.Repository, ".gitmodules")); errors.Is(statErr, os.ErrNotExist) {
			return snapshot, nil
		}
		return Snapshot{}, fmt.Errorf("load submodule configuration: %w", err)
	}
	modules, err := parseConfig(configResult.Stdout)
	if err != nil {
		return Snapshot{}, err
	}
	if len(modules) > limits.MaxModules {
		modules = modules[:limits.MaxModules]
		snapshot.Truncated = true
	}
	statusResult, statusErr := runner.RunBounded(ctx, limits.MaxOutputBytes, "submodule", "status", "--recursive", "--")
	if statusErr != nil && len(statusResult.Stdout) == 0 {
		return Snapshot{}, fmt.Errorf("load submodule status: %w", statusErr)
	}
	statuses, err := parseStatus(statusResult.Stdout, limits.MaxDepth)
	if err != nil {
		return Snapshot{}, err
	}
	for i := range modules {
		module := &modules[i]
		module.Depth = pathDepth(module.Path)
		module.State = StateMissing
		if module.Depth > limits.MaxDepth {
			snapshot.Truncated = true
			continue
		}
		status, ok := statuses[module.Path]
		if !ok {
			if _, statErr := os.Stat(filepath.Join(request.Repository, module.Path)); errors.Is(statErr, os.ErrNotExist) {
				module.State = StateMissing
			} else {
				module.State = StateUninitialized
			}
			continue
		}
		module.RecordedCommit = status.RecordedCommit
		if recorded, recordedErr := recordedCommit(ctx, runner, module.Path); recordedErr == nil {
			module.RecordedCommit = recorded
		}
		module.CheckedOutCommit = status.CheckedOutCommit
		module.State = status.State
		module.Divergence = status.Divergence
		if status.State != StateUninitialized {
			moduleRoot := filepath.Join(request.Repository, module.Path)
			if _, symbolicErr := runner.Run(ctx, "-C", moduleRoot, "symbolic-ref", "--quiet", "--short", "HEAD"); symbolicErr != nil {
				module.State = StateDetached
			}
			working, workingErr := runner.RunBounded(ctx, limits.MaxOutputBytes, "-C", moduleRoot, "--no-optional-locks", "status", "--porcelain=v2", "-z", "--untracked-files=all")
			if workingErr == nil && len(working.Stdout) > 0 {
				module.State = StateDirty
			}
		}
	}
	snapshot.Modules = modules
	return snapshot, nil
}

func recordedCommit(ctx context.Context, runner git.Runner, path string) (string, error) {
	result, err := runner.RunBounded(ctx, 4096, "ls-tree", "-z", "HEAD", "--", path)
	if err != nil {
		return "", err
	}
	record := bytes.TrimSuffix(result.Stdout, []byte{0})
	fields := bytes.Fields(record)
	if len(fields) < 3 || string(fields[1]) != "commit" {
		return "", fmt.Errorf("submodule tree: malformed record for %q", path)
	}
	return string(fields[2]), nil
}

type configModule struct {
	name string
	path string
	url  string
}

func parseConfig(data []byte) ([]Module, error) {
	values := make(map[string]string)
	for _, record := range bytes.Split(data, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		separator := bytes.IndexByte(record, '\n')
		if separator <= 0 {
			return nil, fmt.Errorf("submodule config: malformed NUL record")
		}
		values[string(record[:separator])] = string(record[separator+1:])
	}
	byName := make(map[string]*configModule)
	var order []string
	for key, value := range values {
		parts := strings.Split(key, ".")
		if len(parts) != 3 || parts[0] != "submodule" || (parts[2] != "path" && parts[2] != "url") {
			continue
		}
		name := parts[1]
		module := byName[name]
		if module == nil {
			module = &configModule{name: name}
			byName[name] = module
			order = append(order, name)
		}
		if parts[2] == "path" {
			module.path = value
		} else {
			module.url = redactURL(value)
		}
	}
	// Git config output is not guaranteed to be useful as a map order. Sort
	// names so snapshots are deterministic and do not churn the UI.
	for i := 0; i < len(order); i++ {
		for j := i + 1; j < len(order); j++ {
			if order[j] < order[i] {
				order[i], order[j] = order[j], order[i]
			}
		}
	}
	modules := make([]Module, 0, len(order))
	for _, name := range order {
		module := byName[name]
		if module.path == "" {
			return nil, fmt.Errorf("submodule %q has no path", name)
		}
		modules = append(modules, Module{Name: module.name, Path: module.path, URL: module.url})
	}
	return modules, nil
}

type statusRecord struct {
	RecordedCommit   string
	CheckedOutCommit string
	State            State
	Divergence       int
}

func parseStatus(data []byte, maxDepth int) (map[string]statusRecord, error) {
	result := make(map[string]statusRecord)
	for _, raw := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if len(raw) < 43 {
			return nil, fmt.Errorf("submodule status: malformed record %q", raw)
		}
		marker := raw[0]
		if raw[41] != ' ' {
			return nil, fmt.Errorf("submodule status: malformed commit field %q", raw)
		}
		commit := raw[1:41]
		rest := strings.TrimSpace(raw[42:])
		path := rest
		if markerIndex := strings.LastIndex(rest, " ("); markerIndex >= 0 && strings.HasSuffix(rest, ")") {
			path = rest[:markerIndex]
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("submodule status: empty path in %q", raw)
		}
		depth := pathDepth(path)
		if depth > maxDepth {
			continue
		}
		state := StateClean
		switch marker {
		case '-':
			state = StateUninitialized
		case '+':
			state = StateDiverged
		case 'U':
			state = StateDirty
		case ' ':
		default:
			return nil, fmt.Errorf("submodule status: unknown state %q", marker)
		}
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "-dirty") || strings.Contains(lower, "uncommitted") || strings.Contains(lower, "modified") {
			state = StateDirty
		}
		result[path] = statusRecord{RecordedCommit: commit, CheckedOutCommit: commit, State: state}
	}
	return result, nil
}

func pathDepth(path string) int {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return 0
	}
	return strings.Count(clean, "/") + 1
}

var scpCredential = regexp.MustCompile(`^[^/@\s]+@`)

func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.User = nil
		return parsed.String()
	}
	return scpCredential.ReplaceAllString(raw, "")
}
