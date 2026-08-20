package worktrees

import (
	"context"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

type Entry struct {
	Path     string
	HEAD     string
	Branch   string
	LockNote string
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool
}

func Parse(lines []byte) []Entry {
	var out []Entry
	var entry Entry
	flush := func() {
		if entry.Path != "" {
			out = append(out, entry)
		}
	}
	for _, line := range strings.Split(string(lines), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			entry = Entry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			entry.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			entry.Branch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			entry.Bare = true
		case line == "detached":
			entry.Detached = true
		case line == "locked":
			entry.Locked = true
		case strings.HasPrefix(line, "locked "):
			entry.Locked = true
			entry.LockNote = strings.TrimPrefix(line, "locked ")
		case line == "prunable":
			entry.Prunable = true
		case strings.HasPrefix(line, "prunable "):
			entry.Prunable = true
		}
	}
	flush()
	return out
}

func List(ctx context.Context, runner git.Runner) ([]Entry, error) {
	result, err := runner.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return Parse(result.Stdout), nil
}
