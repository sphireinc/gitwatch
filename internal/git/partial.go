package git

import (
	"context"
	"github.com/sphireinc/git-watch/internal/hunks"
)

// PartialPatch contains an applyable hunk patch and the paths it represents.
type PartialPatch struct {
	Patch []byte
	Paths [][]byte
}

// ApplyCachedPatch applies a partial patch to the index after checking it.
func (r Runner) ApplyCachedPatch(ctx context.Context, p PartialPatch) (Result, error) {
	if result, err := r.RunInput(ctx, p.Patch, "apply", "--cached", "--check", "--whitespace=nowarn", "-"); err != nil {
		return result, err
	}
	return r.RunInput(ctx, p.Patch, "apply", "--cached", "--whitespace=nowarn", "-")
}

// ApplyReverseCachedPatch reverses a partial patch in the index after checking it.
func (r Runner) ApplyReverseCachedPatch(ctx context.Context, p PartialPatch) (Result, error) {
	if result, err := r.RunInput(ctx, p.Patch, "apply", "--cached", "--reverse", "--check", "--whitespace=nowarn", "-"); err != nil {
		return result, err
	}
	return r.RunInput(ctx, p.Patch, "apply", "--cached", "--reverse", "--whitespace=nowarn", "-")
}

// ApplyReversePatch reverses a partial patch in the worktree after checking it.
func (r Runner) ApplyReversePatch(ctx context.Context, p PartialPatch) (Result, error) {
	if result, err := r.RunInput(ctx, p.Patch, "apply", "--reverse", "--check", "--whitespace=nowarn", "-"); err != nil {
		return result, err
	}
	return r.RunInput(ctx, p.Patch, "apply", "--reverse", "--whitespace=nowarn", "-")
}

// SelectedCount returns the number of selected hunks in a patch selection.
func SelectedCount(s hunks.Selection) int { return s.Count() }
