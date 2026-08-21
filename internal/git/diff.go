package git

import (
	"bytes"
	"context"
	"strconv"
)

type Diff struct {
	Path   []byte
	Staged bool
	Text   []byte
	Binary bool
	Result Result
}

func (r Runner) Diff(ctx context.Context, path []byte, staged bool) (Diff, error) {
	return r.DiffWithContext(ctx, path, staged, 3)
}

// DiffWithContext returns a unified diff with the requested number of context lines.
func (r Runner) DiffWithContext(ctx context.Context, path []byte, staged bool, contextLines int) (Diff, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	if contextLines > 0 {
		args = append(args, "--unified="+strconv.Itoa(contextLines))
	}
	args = append(args, "--", string(path))
	result, err := r.Run(ctx, args...)
	d := Diff{Path: append([]byte(nil), path...), Staged: staged, Text: result.Stdout, Result: result}
	d.Binary = bytes.Contains(result.Stdout, []byte("Binary files")) || bytes.IndexByte(result.Stdout, 0) >= 0
	return d, err
}
