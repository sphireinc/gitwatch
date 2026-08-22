package remotes

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	// ErrMissingRemote indicates that an operation did not name an explicit remote.
	ErrMissingRemote    = errors.New("remote operation requires an explicit remote")
	ErrStrategyRequired = errors.New("pull strategy must be explicitly selected")
	ErrMissingTag       = errors.New("tag push requires an explicit tag")
)

// RefMovement describes the commits a remote operation would add or remove.
type RefMovement struct {
	Remote, Branch string
	LocalSHA       string
	RemoteSHA      string
}

// PreviewPush calculates remote ref movement without changing the repository.
func PreviewPush(ctx context.Context, runner git.Runner, remote, branch string) (RefMovement, error) {
	if !validArg(remote) || !validArg(branch) {
		return RefMovement{}, ErrMissingRemote
	}
	local, err := runner.Run(ctx, "rev-parse", "--verify", branch+"^{commit}")
	if err != nil {
		return RefMovement{}, err
	}
	remoteResult, err := runner.Run(ctx, "ls-remote", "--heads", remote, "refs/heads/"+branch)
	if err != nil {
		return RefMovement{}, err
	}
	return RefMovement{Remote: remote, Branch: branch, LocalSHA: strings.TrimSpace(string(local.Stdout)), RemoteSHA: parseRemoteSHA(remoteResult.Stdout)}, nil
}

func parseRemoteSHA(data []byte) string {
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Fetch updates remote-tracking refs for remote.
func Fetch(ctx context.Context, runner git.Runner, remote string) (git.Result, error) {
	if !validArg(remote) {
		return git.Result{}, ErrMissingRemote
	}
	return runner.Run(ctx, "fetch", "--progress", remote)
}

// Pull integrates branch from remote using the selected strategy.
func Pull(ctx context.Context, runner git.Runner, remote, branch, strategy string) (git.Result, error) {
	if !validArg(remote) || !validArg(branch) {
		return git.Result{}, ErrMissingRemote
	}
	if strategy != "merge" && strategy != "rebase" && strategy != "ff-only" {
		return git.Result{}, ErrStrategyRequired
	}
	args := []string{"pull", "--progress"}
	switch strategy {
	case "merge":
		args = append(args, "--no-rebase")
	case "rebase", "ff-only":
		args = append(args, "--"+strategy)
	}
	return runner.Run(ctx, append(args, remote, branch)...)
}

// Push publishes branch to remote, optionally using force-with-lease.
func Push(ctx context.Context, runner git.Runner, remote, branch string, forceWithLease bool) (git.Result, error) {
	return PushWithOptions(ctx, runner, remote, branch, PushOptions{ForceWithLease: forceWithLease})
}

// PushOptions controls safety checks and refspec behavior for a push.
type PushOptions struct {
	ForceWithLease bool
	SetUpstream    bool
	Tag            bool
}

// PushWithOptions publishes ref using the supplied push safety options.
func PushWithOptions(ctx context.Context, runner git.Runner, remote, ref string, options PushOptions) (git.Result, error) {
	if !validArg(remote) {
		return git.Result{}, ErrMissingRemote
	}
	if !validArg(ref) {
		if options.Tag {
			return git.Result{}, ErrMissingTag
		}
		return git.Result{}, ErrMissingRemote
	}
	args := []string{"push", "--progress"}
	if options.ForceWithLease {
		args = append(args, "--force-with-lease")
	}
	if options.SetUpstream {
		args = append(args, "--set-upstream")
	}
	if options.Tag {
		args = append(args, remote, "refs/tags/"+ref+":refs/tags/"+ref)
	} else {
		args = append(args, remote, ref)
	}
	return runner.Run(ctx, args...)
}

// PushTag publishes tag to remote.
func PushTag(ctx context.Context, runner git.Runner, remote, tag string) (git.Result, error) {
	return PushWithOptions(ctx, runner, remote, tag, PushOptions{Tag: true})
}

// PushSetUpstream publishes branch and records remote as its upstream.
func PushSetUpstream(ctx context.Context, runner git.Runner, remote, branch string) (git.Result, error) {
	return PushWithOptions(ctx, runner, remote, branch, PushOptions{SetUpstream: true})
}

func validArg(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}
