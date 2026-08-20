package git

import (
	"bytes"
	"context"
)

type Diff struct {
	Path   []byte
	Staged bool
	Text   []byte
	Binary bool
	Result Result
}

func (r Runner) Diff(ctx context.Context, path []byte, staged bool) (Diff, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", string(path))
	result, err := r.Run(ctx, args...)
	d := Diff{Path: append([]byte(nil), path...), Staged: staged, Text: result.Stdout, Result: result}
	d.Binary = bytes.Contains(result.Stdout, []byte("Binary files")) || bytes.IndexByte(result.Stdout, 0) >= 0
	return d, err
}
