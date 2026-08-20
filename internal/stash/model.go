package stash

import (
	"context"
	"github.com/jusanchez/gitwatch/internal/git"
	"strings"
)

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
	return r.Run(ctx, "stash", "push", "-m", message, "--include-untracked", "--")
}
func Apply(ctx context.Context, r git.Runner, ref string) (git.Result, error) {
	return r.Run(ctx, "stash", "apply", "--", ref)
}
