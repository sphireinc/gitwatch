package remotes

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	// ErrMissingRemote indicates that an operation did not name an explicit remote.
	ErrMissingRemote     = errors.New("remote operation requires an explicit remote")
	ErrStrategyRequired  = errors.New("pull strategy must be explicitly selected")
	ErrMissingTag        = errors.New("tag push requires an explicit tag")
	ErrMissingURL        = errors.New("remote operation requires an explicit URL")
	ErrInvalidRemoteName = errors.New("remote name is invalid")
)

// RefMovement describes the commits a remote operation would add or remove.
type RefMovement struct {
	Remote, Branch string
	LocalSHA       string
	RemoteSHA      string
}

type commandRunner interface {
	Run(context.Context, ...string) (git.Result, error)
}

// Add creates a named remote with the URL passed as one opaque argv value.
func Add(ctx context.Context, runner commandRunner, name, remoteURL string) (git.Result, error) {
	if !validRemoteName(name) {
		return git.Result{}, ErrInvalidRemoteName
	}
	if !validURL(remoteURL) {
		return git.Result{}, ErrMissingURL
	}
	return runner.Run(ctx, "remote", "add", name, remoteURL)
}

// Rename changes a remote name without touching its configured URLs.
func Rename(ctx context.Context, runner commandRunner, oldName, newName string) (git.Result, error) {
	if !validRemoteName(oldName) || !validRemoteName(newName) {
		return git.Result{}, ErrInvalidRemoteName
	}
	return runner.Run(ctx, "remote", "rename", oldName, newName)
}

// SetURL replaces the fetch URL for a remote. The URL is never included in a
// returned error or operation label by this package.
func SetURL(ctx context.Context, runner commandRunner, name, remoteURL string) (git.Result, error) {
	if !validRemoteName(name) {
		return git.Result{}, ErrInvalidRemoteName
	}
	if !validURL(remoteURL) {
		return git.Result{}, ErrMissingURL
	}
	return runner.Run(ctx, "remote", "set-url", name, remoteURL)
}

// Remove deletes one configured remote.
func Remove(ctx context.Context, runner commandRunner, name string) (git.Result, error) {
	if !validRemoteName(name) {
		return git.Result{}, ErrInvalidRemoteName
	}
	return runner.Run(ctx, "remote", "remove", name)
}

// Prune removes stale remote-tracking refs. DryRun exposes Git's affected-ref
// preview without changing the repository.
func Prune(ctx context.Context, runner commandRunner, name string, dryRun bool) (git.Result, error) {
	if !validRemoteName(name) {
		return git.Result{}, ErrInvalidRemoteName
	}
	args := []string{"remote", "prune"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return runner.Run(ctx, append(args, name)...)
}

// GetURL returns one redacted configured remote URL.
func GetURL(ctx context.Context, runner commandRunner, name string, push bool) (string, error) {
	if !validRemoteName(name) {
		return "", ErrInvalidRemoteName
	}
	args := []string{"remote", "get-url"}
	if push {
		args = append(args, "--push")
	}
	result, err := runner.Run(ctx, append(args, name)...)
	if err != nil {
		return "", err
	}
	return Redact(string(result.Stdout)), nil
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

// DeleteTag removes exactly one named tag from a remote. It is intentionally
// separate from PushTag so callers must opt into the destructive refspec.
func DeleteTag(ctx context.Context, runner git.Runner, remote, tag string) (git.Result, error) {
	if !validArg(remote) {
		return git.Result{}, ErrMissingRemote
	}
	if !validArg(tag) {
		return git.Result{}, ErrMissingTag
	}
	return runner.Run(ctx, "push", "--progress", remote, ":refs/tags/"+tag)
}

// PushSetUpstream publishes branch and records remote as its upstream.
func PushSetUpstream(ctx context.Context, runner git.Runner, remote, branch string) (git.Result, error) {
	return PushWithOptions(ctx, runner, remote, branch, PushOptions{SetUpstream: true})
}

func validArg(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}

func validRemoteName(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00") && !strings.Contains(value, "..") && !strings.Contains(value, "/")
}

func validURL(value string) bool {
	return strings.TrimSpace(value) != "" && !strings.ContainsAny(value, "\r\n\x00")
}
