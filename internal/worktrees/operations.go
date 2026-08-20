package worktrees

import (
	"context"
	"errors"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

var ErrInvalidTarget = errors.New("invalid worktree target")

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

func Prune(ctx context.Context, runner git.Runner, dryRun bool) (git.Result, error) {
	args := []string{"worktree", "prune"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return runner.Run(ctx, args...)
}

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
