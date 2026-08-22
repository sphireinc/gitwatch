package git

import (
	"context"
	"fmt"
	"strings"
)

// CommitOptions describes the commit flags and message to pass to Git.
type CommitOptions struct {
	Message                      []byte
	Amend, NoEdit, Signoff, Sign bool
	Author                       string
}

// CommitResult contains the command result and the new commit ID.
type CommitResult struct {
	SHA    string
	Result Result
}

// CommitConfig contains the user's Git identity and signing preferences.
type CommitConfig struct {
	UserName    string
	UserEmail   string
	SignEnabled bool
	SignFormat  string
}

// CommitConfig reads optional commit identity and signing settings from Git.
func (r Runner) CommitConfig(ctx context.Context) CommitConfig {
	config := CommitConfig{}
	read := func(key string) string {
		result, err := r.Run(ctx, "config", "--get", key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(result.Stdout))
	}
	config.UserName = read("user.name")
	config.UserEmail = read("user.email")
	sign := read("commit.gpgsign")
	config.SignEnabled = strings.EqualFold(sign, "true") || sign == "1"
	config.SignFormat = read("gpg.format")
	return config
}

// Commit creates a commit using opts and returns its resulting object ID.
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
		if strings.ContainsAny(opts.Author, "\r\n") {
			return CommitResult{}, fmt.Errorf("author must be one line")
		}
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
	return CommitResult{SHA: strings.TrimSpace(string(sha.Stdout)), Result: result}, nil
}
