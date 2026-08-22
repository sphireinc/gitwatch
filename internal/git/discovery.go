package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Discovery describes the repository and Git metadata found from a directory.
type Discovery struct {
	Root         string
	GitDir       string
	CommonDir    string
	Bare         bool
	Worktree     bool
	Linked       bool
	Submodule    bool
	GitVersion   string
	Capabilities Capabilities
	Head         string
	Detached     bool
	Unborn       bool
}

// Discover resolves repository topology and HEAD state for dir.
func Discover(ctx context.Context, dir string) (Discovery, error) {
	r := NewRunner(dir)
	values, err := r.Run(ctx, "rev-parse", "--show-toplevel", "--absolute-git-dir", "--git-common-dir", "--is-bare-repository", "--is-inside-work-tree", "--git-dir")
	if err != nil {
		return Discovery{}, err
	}
	lines := strings.Split(strings.TrimSpace(string(values.Stdout)), "\n")
	if len(lines) < 6 {
		return Discovery{}, fmt.Errorf("git discovery: expected six fields, got %d", len(lines))
	}
	bare, parseErr := strconv.ParseBool(lines[3])
	if parseErr != nil {
		return Discovery{}, fmt.Errorf("git discovery bare flag: %w", parseErr)
	}
	inside, parseErr := strconv.ParseBool(lines[4])
	if parseErr != nil {
		return Discovery{}, fmt.Errorf("git discovery worktree flag: %w", parseErr)
	}
	if bare {
		return Discovery{}, fmt.Errorf("gitwatch does not support bare repositories: %w", ErrCommandFailed)
	}
	root, err := filepath.Abs(lines[0])
	if err != nil {
		return Discovery{}, fmt.Errorf("git discovery root: %w", err)
	}
	gitDir, err := resolvePath(dir, lines[5])
	if err != nil {
		return Discovery{}, fmt.Errorf("git discovery git dir: %w", err)
	}
	commonDir, err := resolvePath(dir, lines[2])
	if err != nil {
		return Discovery{}, fmt.Errorf("git discovery common dir: %w", err)
	}
	versionResult, err := r.Run(ctx, "--version")
	if err != nil {
		return Discovery{}, err
	}
	headResult, headErr := r.Run(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	capabilities, err := r.Probe(ctx)
	if err != nil {
		return Discovery{}, err
	}
	d := Discovery{Root: root, GitDir: gitDir, CommonDir: commonDir, Bare: bare, Worktree: inside, GitVersion: strings.TrimSpace(string(versionResult.Stdout)), Capabilities: capabilities}
	d.Linked = !samePath(d.GitDir, d.CommonDir)
	d.Submodule = filepath.Base(root) != filepath.Base(commonDir) && strings.Contains(d.GitDir, filepath.Join(".git", "modules"))
	headOID, oidErr := r.Run(ctx, "rev-parse", "--verify", "HEAD")
	if headErr == nil {
		d.Head = strings.TrimSpace(string(headResult.Stdout))
		if oidErr != nil {
			d.Unborn = true
		}
	} else {
		d.Detached = true
		if oidErr != nil {
			d.Unborn = true
		} else {
			d.Head = strings.TrimSpace(string(headOID.Stdout))
		}
	}
	return d, nil
}

func resolvePath(base, value string) (string, error) {
	if filepath.IsAbs(value) {
		return filepath.Abs(value)
	}
	return filepath.Abs(filepath.Join(base, value))
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(aa); err == nil {
		aa = resolved
	}
	if resolved, err := filepath.EvalSymlinks(bb); err == nil {
		bb = resolved
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}
