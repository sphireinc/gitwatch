package worktrees

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

// ErrInvalidTarget indicates that a worktree path or branch is unsafe to use.
var ErrInvalidTarget = errors.New("invalid worktree target")

// Add creates a linked worktree at path for branch.
func Add(ctx context.Context, runner git.Runner, path, branch string) (git.Result, error) {
	if !validTarget(path) || (branch != "" && !validTarget(branch)) {
		return git.Result{}, ErrInvalidTarget
	}
	args := []string{"worktree", "add"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	return runner.Run(ctx, append(args, path)...)
}

// AddWithCommit creates a worktree at path, optionally creating branch from
// commit. Keeping the commit argument separate avoids shell-like command
// construction while allowing callers to expose a useful open workflow.
func AddWithCommit(ctx context.Context, runner git.Runner, path, branch, commit string) (git.Result, error) {
	if !validTarget(path) || (branch != "" && !validTarget(branch)) || (commit != "" && !validTarget(commit)) {
		return git.Result{}, ErrInvalidTarget
	}
	args := []string{"worktree", "add"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, path)
	if commit != "" {
		args = append(args, commit)
	}
	return runner.Run(ctx, args...)
}

// Remove deletes a linked worktree, optionally forcing removal of changes.
func Remove(ctx context.Context, runner git.Runner, path string, force bool) (git.Result, error) {
	if !validTarget(path) {
		return git.Result{}, ErrInvalidTarget
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	return runner.Run(ctx, append(args, path)...)
}

// Prune removes stale linked-worktree metadata, or previews that removal.
func Prune(ctx context.Context, runner git.Runner, dryRun bool) (git.Result, error) {
	args := []string{"worktree", "prune"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return runner.Run(ctx, args...)
}

// Occupancy maps each branch to the worktree path currently using it.
func Occupancy(entries []Entry) map[string]string {
	occupied := make(map[string]string)
	for _, entry := range entries {
		branch := strings.TrimPrefix(entry.Branch, "refs/heads/")
		if branch != "" {
			occupied[branch] = entry.Path
		}
	}
	return occupied
}

func validTarget(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}
