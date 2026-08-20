package git

import (
	"context"
	"github.com/jusanchez/gitwatch/internal/hunks"
)

type PartialPatch struct {
	Patch []byte
	Paths [][]byte
}

func (r Runner) ApplyCachedPatch(ctx context.Context, p PartialPatch) (Result, error) {
	if result, err := r.RunInput(ctx, p.Patch, "apply", "--cached", "--check", "--whitespace=nowarn", "-"); err != nil {
		return result, err
	}
	return r.RunInput(ctx, p.Patch, "apply", "--cached", "--whitespace=nowarn", "-")
}
func (r Runner) ApplyReversePatch(ctx context.Context, p PartialPatch) (Result, error) {
	if result, err := r.RunInput(ctx, p.Patch, "apply", "--reverse", "--check", "--whitespace=nowarn", "-"); err != nil {
		return result, err
	}
	return r.RunInput(ctx, p.Patch, "apply", "--reverse", "--whitespace=nowarn", "-")
}
func SelectedCount(s hunks.Selection) int { return s.Count() }
