package stash

import (
	"context"
	"errors"
	"github.com/jusanchez/gitwatch/internal/git"
	"strings"
)

var ErrInvalidRef = errors.New("invalid stash reference")

type Entry struct {
	Ref, OID, Branch, Message string
	Unix                      int64
}

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
func List(ctx context.Context, r git.Runner) ([]Entry, error) {
	res, err := r.Run(ctx, "stash", "list", "--format=%gd %H %ct %s")
	if err != nil {
		return nil, err
	}
	return Parse(res.Stdout), nil
}
func Create(ctx context.Context, r git.Runner, message string) (git.Result, error) {
	return CreateWithOptions(ctx, r, message, true)
}

func CreateWithOptions(ctx context.Context, r git.Runner, message string, includeUntracked bool) (git.Result, error) {
	args := []string{"stash", "push", "-m", message}
	if includeUntracked {
		args = append(args, "--include-untracked")
	}
	return r.Run(ctx, append(args, "--")...)
}
func Apply(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "apply", ref)
}

func Pop(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "pop", ref)
}

func Drop(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "drop", ref)
}

func Branch(ctx context.Context, r git.Runner, name, ref string) (git.Result, error) {
	if !validRef(ref) || !validName(name) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "branch", name, ref)
}

func Show(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	if !validRef(ref) {
		return git.Result{}, ErrInvalidRef
	}
	return r.Run(ctx, "stash", "show", "--patch", "--stat", ref)
}

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
