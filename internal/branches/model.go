package branches

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

var (
	ErrInvalidName   = errors.New("invalid branch name")
	ErrCurrentBranch = errors.New("cannot delete the checked-out branch")
	ErrConfirmation  = errors.New("branch mutation confirmation did not match")
)

type Branch struct {
	Name, OID, Upstream string
	OccupiedPath        string
	LastCommitUnix      int64
	Subject             string
	Current, Remote     bool
	Merged              bool
	Ahead, Behind       int
}

func Parse(lines []byte) []Branch {
	var out []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
		p := strings.Split(line, "\x00")
		if len(p) < 4 {
			continue
		}
		branch := Branch{Name: p[0], OID: p[1], Upstream: p[2], Current: strings.TrimSpace(p[3]) == "*", Remote: strings.HasPrefix(p[0], "remotes/")}
		if len(p) > 4 {
			branch.Ahead, branch.Behind = ParseTracking(p[4])
		}
		if len(p) > 5 {
			if value, err := strconv.ParseInt(strings.TrimSpace(p[5]), 10, 64); err == nil {
				branch.LastCommitUnix = value
			}
		}
		if len(p) > 6 {
			branch.Subject = p[6]
		}
		out = append(out, branch)
	}
	return out
}
func List(ctx context.Context, r git.Runner) ([]Branch, error) {
	res, err := r.Run(ctx, "for-each-ref", "--format=%(refname:short)\x00%(objectname)\x00%(upstream:short)\x00%(HEAD)\x00%(upstream:trackshort)\x00%(creatordate:unix)\x00%(subject)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	entries := Parse(res.Stdout)
	for i := range entries {
		if entries[i].Upstream == "" || entries[i].Remote {
			continue
		}
		behind, ahead, err := Divergence(ctx, r, entries[i].Upstream, entries[i].Name)
		if err != nil {
			return nil, err
		}
		entries[i].Ahead, entries[i].Behind = ahead, behind
	}
	merged, err := r.Run(ctx, "for-each-ref", "--merged=HEAD", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	mergedNames := make(map[string]struct{})
	for _, name := range strings.Split(strings.TrimSpace(string(merged.Stdout)), "\n") {
		if name != "" {
			mergedNames[name] = struct{}{}
		}
	}
	for i := range entries {
		_, entries[i].Merged = mergedNames[entries[i].Name]
	}
	return entries, nil
}

// Divergence returns the number of commits unique to upstream and local.
// rev-list emits the left (upstream-only) and right (local-only) counts.
func Divergence(ctx context.Context, r git.Runner, upstream, local string) (behind, ahead int, err error) {
	if strings.TrimSpace(upstream) == "" || strings.TrimSpace(local) == "" {
		return 0, 0, nil
	}
	result, err := r.Run(ctx, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", upstream, local))
	if err != nil {
		return 0, 0, err
	}
	return ParseDivergence(result.Stdout)
}

func ListWithOccupancy(ctx context.Context, r git.Runner, occupancy map[string]string) ([]Branch, error) {
	entries, err := List(ctx, r)
	if err != nil {
		return nil, err
	}
	AttachOccupancy(entries, occupancy)
	return entries, nil
}

func AttachOccupancy(entries []Branch, occupancy map[string]string) {
	for i := range entries {
		if !entries[i].Remote {
			entries[i].OccupiedPath = occupancy[entries[i].Name]
		}
	}
}
func Checkout(ctx context.Context, r git.Runner, name string) (git.Result, error) {
	return r.Run(ctx, "switch", "--", name)
}
func Create(ctx context.Context, r git.Runner, name string) (git.Result, error) {
	if !validName(name) {
		return git.Result{}, ErrInvalidName
	}
	return r.Run(ctx, "switch", "--create", name)
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
