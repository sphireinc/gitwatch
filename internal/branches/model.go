package branches

import (
	"context"
	"github.com/jusanchez/gitwatch/internal/git"
	"strings"
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
	return r.Run(ctx, "switch", "--create", "--", name)
}
