package branches

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	ErrInvalidRemoteRef   = errors.New("invalid remote branch ref")
	ErrRemoteConfirmation = errors.New("remote branch confirmation did not match")
)

type remoteGitRunner interface {
	Run(context.Context, ...string) (git.Result, error)
}

// CheckoutRemote creates a local branch that tracks one fully qualified
// remote branch. The remote and branch are separate arguments at the Git
// boundary so similarly named remotes cannot be confused.
func CheckoutRemote(ctx context.Context, runner remoteGitRunner, remote, remoteBranch, localBranch string) (git.Result, error) {
	if !validRemoteRefParts(remote, remoteBranch) || !validName(localBranch) {
		return git.Result{}, ErrInvalidRemoteRef
	}
	return runner.Run(ctx, "switch", "--track", "--create", localBranch, remote+"/"+remoteBranch)
}

// CheckoutRemoteDetached checks out a remote branch without creating a local
// tracking branch.
func CheckoutRemoteDetached(ctx context.Context, runner remoteGitRunner, remote, remoteBranch string) (git.Result, error) {
	if !validRemoteRefParts(remote, remoteBranch) {
		return git.Result{}, ErrInvalidRemoteRef
	}
	return runner.Run(ctx, "switch", "--detach", "--", remote+"/"+remoteBranch)
}

// DeleteRemote deletes exactly one branch from exactly one named remote. The
// confirmation must contain the full remote-qualified branch ref.
func DeleteRemote(ctx context.Context, runner remoteGitRunner, remote, remoteBranch, confirmation string) (git.Result, error) {
	if !validRemoteRefParts(remote, remoteBranch) {
		return git.Result{}, ErrInvalidRemoteRef
	}
	fullRef := remote + "/" + remoteBranch
	if strings.TrimSpace(confirmation) != fullRef {
		return git.Result{}, ErrRemoteConfirmation
	}
	return runner.Run(ctx, "push", remote, ":refs/heads/"+remoteBranch)
}

func validRemoteRefParts(remote, remoteBranch string) bool {
	return validName(remote) && validName(remoteBranch) && !strings.Contains(remote, "/") && !strings.HasPrefix(remoteBranch, "-") && !strings.ContainsAny(remoteBranch, "\r\n\x00")
}
