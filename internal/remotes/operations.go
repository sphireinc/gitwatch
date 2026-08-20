package remotes

import (
	"context"
	"errors"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

var ErrMissingRemote = errors.New("remote operation requires an explicit remote")

func Fetch(ctx context.Context, runner git.Runner, remote string) (git.Result, error) {
	if strings.TrimSpace(remote) == "" {
		return git.Result{}, ErrMissingRemote
	}
	return runner.Run(ctx, "fetch", "--progress", "--", remote)
}

func Pull(ctx context.Context, runner git.Runner, remote, branch, strategy string) (git.Result, error) {
	if strings.TrimSpace(remote) == "" || strings.TrimSpace(branch) == "" {
		return git.Result{}, ErrMissingRemote
	}
	args := []string{"pull", "--progress"}
	if strategy == "merge" {
		args = append(args, "--no-rebase")
	} else if strategy == "rebase" || strategy == "ff-only" {
		args = append(args, "--"+strategy)
	}
	return runner.Run(ctx, append(args, remote, branch)...)
}

func Push(ctx context.Context, runner git.Runner, remote, branch string, forceWithLease bool) (git.Result, error) {
	if strings.TrimSpace(remote) == "" || strings.TrimSpace(branch) == "" {
		return git.Result{}, ErrMissingRemote
	}
	args := []string{"push", "--progress"}
	if forceWithLease {
		args = append(args, "--force-with-lease")
	}
	return runner.Run(ctx, append(args, remote, branch)...)
}
