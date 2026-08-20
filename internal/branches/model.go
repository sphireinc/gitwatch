package branches

import (
	"context"
	"errors"
	"github.com/jusanchez/gitwatch/internal/git"
	"strings"
)

var (
	ErrInvalidName   = errors.New("invalid branch name")
	ErrCurrentBranch = errors.New("cannot delete the checked-out branch")
	ErrConfirmation  = errors.New("branch mutation confirmation did not match")
)

type Branch struct {
	Name, OID, Upstream string
	Current, Remote     bool
	Ahead, Behind       int
}

func Parse(lines []byte) []Branch {
	var out []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
		p := strings.Split(line, "\x00")
		if len(p) < 4 {
			continue
		}
		out = append(out, Branch{Name: p[0], OID: p[1], Upstream: p[2], Current: p[3] == "*", Remote: strings.HasPrefix(p[0], "remotes/")})
	}
	return out
}
func List(ctx context.Context, r git.Runner) ([]Branch, error) {
	res, err := r.Run(ctx, "for-each-ref", "--format=%(refname:short)\x00%(objectname)\x00%(upstream:short)\x00 ", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	return Parse(res.Stdout), nil
}
func Checkout(ctx context.Context, r git.Runner, name string) (git.Result, error) {
	return r.Run(ctx, "switch", "--", name)
}
func Create(ctx context.Context, r git.Runner, name string) (git.Result, error) {
	if !validName(name) {
		return git.Result{}, ErrInvalidName
	}
	return r.Run(ctx, "switch", "--create", "--", name)
}

func Rename(ctx context.Context, r git.Runner, oldName, newName string) (git.Result, error) {
	if !validName(oldName) || !validName(newName) {
		return git.Result{}, ErrInvalidName
	}
	return r.Run(ctx, "branch", "--move", oldName, newName)
}

func SetUpstream(ctx context.Context, r git.Runner, local, upstream string) (git.Result, error) {
	if !validName(local) || !validName(upstream) {
		return git.Result{}, ErrInvalidName
	}
	return r.Run(ctx, "branch", "--set-upstream-to", upstream, local)
}

func UnsetUpstream(ctx context.Context, r git.Runner, local string) (git.Result, error) {
	if !validName(local) {
		return git.Result{}, ErrInvalidName
	}
	return r.Run(ctx, "branch", "--unset-upstream", local)
}

func Delete(ctx context.Context, r git.Runner, branch Branch, confirmation Confirmation, input string) (git.Result, error) {
	if !validName(branch.Name) || branch.Current {
		if branch.Current {
			return git.Result{}, ErrCurrentBranch
		}
		return git.Result{}, ErrInvalidName
	}
	if confirmation.Name != branch.Name || !confirmation.Accept(input) {
		return git.Result{}, ErrConfirmation
	}
	args := []string{"branch", "--delete"}
	if confirmation.Force {
		args = append(args, "--force")
	}
	return r.Run(ctx, append(args, branch.Name)...)
}

func validName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !strings.HasPrefix(name, "-") && !strings.ContainsAny(name, "\r\n\x00")
}
