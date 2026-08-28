package git

import (
	"context"
	"fmt"
	"strings"
)

// CommitTreeMaxCommits is the largest history request accepted by the UI.
const CommitTreeMaxCommits = 1000

// CommitTreeMaxBytes bounds graph output before it enters the Bubble Tea model.
const CommitTreeMaxBytes = 256 << 10

// CommitTree contains presentation lines and the HEAD identity used for refresh coalescing.
type CommitTree struct {
	Head      string
	Lines     []string
	Colorized bool
}

// LoadCommitTree loads a bounded, human-readable graph without mutating Git state.
func LoadCommitTree(ctx context.Context, runner Runner, limit int) (CommitTree, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > CommitTreeMaxCommits {
		limit = CommitTreeMaxCommits
	}
	head, err := runner.Run(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		// Unborn repositories have no HEAD yet and simply show an empty tree.
		if strings.Contains(string(head.Stderr), "Needed a single revision") || strings.Contains(string(head.Stderr), "does not have any commits") {
			return CommitTree{}, nil
		}
		return CommitTree{}, err
	}
	format := "%Cred%h%Creset -%C(auto)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset"
	result, err := runner.Run(ctx, "--no-pager", "log", "--color=always", "--graph", "--all", "--decorate", "--pretty=format:"+format, "-n", fmt.Sprint(limit))
	if err != nil {
		return CommitTree{}, err
	}
	if len(result.Stdout) > CommitTreeMaxBytes {
		return CommitTree{}, fmt.Errorf("commit tree output exceeds %d bytes", CommitTreeMaxBytes)
	}
	text := strings.TrimSuffix(string(result.Stdout), "\n")
	if text == "" {
		return CommitTree{Head: strings.TrimSpace(string(head.Stdout)), Colorized: true}, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > limit*4+8 {
		lines = lines[:limit*4+8]
	}
	return CommitTree{Head: strings.TrimSpace(string(head.Stdout)), Lines: lines, Colorized: true}, nil
}
