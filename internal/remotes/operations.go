package remotes

import (
	"context"
	"errors"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

var (
	ErrMissingRemote    = errors.New("remote operation requires an explicit remote")
	ErrStrategyRequired = errors.New("pull strategy must be explicitly selected")
)

func Fetch(ctx context.Context, runner git.Runner, remote string) (git.Result, error) {
	if !validArg(remote) {
		return git.Result{}, ErrMissingRemote
	}
	return runner.Run(ctx, "fetch", "--progress", remote)
}

func Pull(ctx context.Context, runner git.Runner, remote, branch, strategy string) (git.Result, error) {
	if !validArg(remote) || !validArg(branch) {
		return git.Result{}, ErrMissingRemote
	}
	if strategy != "merge" && strategy != "rebase" && strategy != "ff-only" {
		return git.Result{}, ErrStrategyRequired
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
	if !validArg(remote) || !validArg(branch) {
		return git.Result{}, ErrMissingRemote
	}
	args := []string{"push", "--progress"}
	if forceWithLease {
		args = append(args, "--force-with-lease")
	}
	return runner.Run(ctx, append(args, remote, branch)...)
}

func validArg(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}
