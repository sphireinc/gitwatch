// Package stash provides guarded Git stash operations and immutable stash rows.
package stash

import (
	"context"
	"errors"
	"github.com/sphireinc/git-watch/internal/git"
	"strings"
)

// ErrInvalidRef indicates that a stash reference is empty or malformed.
var ErrInvalidRef = errors.New("invalid stash reference")

// Entry is one stash entry returned by Git.
type Entry struct {
	Ref, OID, Branch, Message string
	Unix                      int64
}

// Parse converts formatted stash output into entries.
func Parse(lines []byte) []Entry {
	var out []Entry
	for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
		if line == "" {
			continue
		}
		p := strings.SplitN(line, " ", 4)
		if len(p) < 4 {
			continue
		}
		out = append(out, Entry{Ref: p[0], OID: p[1], Unix: parseInt(p[2]), Message: p[3]})
	}
	return out
}
func parseInt(s string) int64 {
	var n int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int64(r-'0')
		}
	}
	return n
}

// List returns the repository's stash entries, newest first.
func List(ctx context.Context, r git.Runner) ([]Entry, error) {
	res, err := r.Run(ctx, "stash", "list", "--format=%gd %H %ct %s")
	if err != nil {
		return nil, err
	}
	return Parse(res.Stdout), nil
}

// Create saves tracked worktree changes as a stash.
func Create(ctx context.Context, r git.Runner, message string) (git.Result, error) {
	return CreateWithOptions(ctx, r, message, true)
}

// CreateWithOptions saves changes and optionally includes untracked files.
func CreateWithOptions(ctx context.Context, r git.Runner, message string, includeUntracked bool) (git.Result, error) {
	args := []string{"stash", "push", "-m", message}
	if includeUntracked {
		args = append(args, "--include-untracked")
	}
	return r.Run(ctx, append(args, "--")...)
}

// Apply reapplies a stash while retaining it in the stash list.
func Apply(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "apply", ref)
}

// ApplyChecked refuses to merge stash contents into a dirty worktree.
func ApplyChecked(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if err := RequireClean(ctx, r); err != nil {
		return git.Result{}, err
	}
	return Apply(ctx, r, ref)
}

// Pop reapplies a stash and removes it when the operation succeeds.
func Pop(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "pop", ref)
}

// PopChecked refuses to pop a stash when the worktree is not clean.
func PopChecked(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if err := RequireClean(ctx, r); err != nil {
		return git.Result{}, err
	}
	return Pop(ctx, r, ref)
}

// Drop removes a stash after validating its reference.
func Drop(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "drop", ref)
}

// Branch creates a new branch from a stash and applies the stash changes.
func Branch(ctx context.Context, r git.Runner, name, ref string) (git.Result, error) {
	if !validRef(ref) || !validName(name) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "branch", name, ref)
}

// BranchChecked refuses to create and apply a stash branch over local changes.
func BranchChecked(ctx context.Context, r git.Runner, name, ref string) (git.Result, error) {
	if err := RequireClean(ctx, r); err != nil {
		return git.Result{}, err
	}
	return Branch(ctx, r, name, ref)
}

// Show returns the patch represented by a stash.
func Show(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "show", "--patch", "--stat", ref)
}

// RequireClean rejects stash application when tracked worktree changes exist.
func RequireClean(ctx context.Context, r git.Runner) error {
	result, err := r.Run(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return err
	}
	if len(result.Stdout) != 0 {
		return errors.New("working tree is not clean")
	}
	return nil
}

func validRef(ref string) bool {
	return strings.TrimSpace(ref) != "" && !strings.HasPrefix(strings.TrimSpace(ref), "-") && !strings.ContainsAny(ref, "\r\n\x00")
}

func validName(name string) bool {
	return strings.TrimSpace(name) != "" && !strings.HasPrefix(strings.TrimSpace(name), "-") && !strings.ContainsAny(name, "\r\n\x00")
}
