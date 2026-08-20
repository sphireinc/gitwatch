package git

import (
	"context"
	"fmt"
)

type CommitOptions struct {
	Message                      []byte
	Amend, NoEdit, Signoff, Sign bool
	Author                       string
}
type CommitResult struct {
	SHA    string
	Result Result
}

func (r Runner) Commit(ctx context.Context, opts CommitOptions) (CommitResult, error) {
	if len(opts.Message) == 0 && !opts.NoEdit {
		return CommitResult{}, fmt.Errorf("commit message is empty")
	}
	args := []string{"commit"}
	if opts.Amend {
		args = append(args, "--amend")
	}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	} else {
		args = append(args, "-F", "-")
	}
	if opts.Signoff {
		args = append(args, "--signoff")
	}
	if opts.Sign {
		args = append(args, "-S")
	}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	result, err := r.RunInput(ctx, opts.Message, args...)
	if err != nil {
		return CommitResult{Result: result}, err
	}
	sha, err := r.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return CommitResult{Result: result}, err
	}
	return CommitResult{SHA: string(sha.Stdout), Result: result}, nil
}
