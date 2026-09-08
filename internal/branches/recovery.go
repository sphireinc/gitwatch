package branches

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

// ErrInvalidRecoveryRef indicates that a recovery target cannot be passed to
// Git as a single revision argument.
var ErrInvalidRecoveryRef = errors.New("invalid branch recovery ref")

type recoveryGitRunner interface {
	Run(context.Context, ...string) (git.Result, error)
}

// ResetMode identifies the only reset modes exposed by the safe branch
// recovery workflow. Neither mode changes tracked worktree files.
type ResetMode uint8

const (
	ResetSoft ResetMode = iota
	ResetMixed
)

// FastForward advances the current branch only when Git can do so without a
// merge commit or history rewrite. The source is passed after -- so a remote
// qualified ref cannot be interpreted as an option.
func FastForward(ctx context.Context, runner recoveryGitRunner, upstream string) (git.Result, error) {
	if !validRecoveryRef(upstream) {
		return git.Result{}, ErrInvalidRecoveryRef
	}
	return runner.Run(ctx, "merge", "--ff-only", "--", upstream)
}

// Reset moves HEAD and the index to target while preserving worktree content.
// Soft preserves the index; mixed resets the index but leaves files in place.
// Hard reset is deliberately not represented by this API.
func Reset(ctx context.Context, runner recoveryGitRunner, mode ResetMode, target string) (git.Result, error) {
	if !validRecoveryRef(target) {
		return git.Result{}, ErrInvalidRecoveryRef
	}
	flag := ""
	switch mode {
	case ResetSoft:
		flag = "--soft"
	case ResetMixed:
		flag = "--mixed"
	default:
		return git.Result{}, errors.New("unsupported branch reset mode")
	}
	return runner.Run(ctx, "reset", flag, "--", target)
}

func validRecoveryRef(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, " \t\r\n\x00")
}
