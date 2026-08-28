package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	// DefaultUnpushedCommits is the default number of commits shown in the pane.
	DefaultUnpushedCommits = 100
	// MaxUnpushedCommits bounds the work requested from Git for this view.
	MaxUnpushedCommits = 1000
	// MaxUnpushedBytes bounds the rendered Git output retained in memory.
	MaxUnpushedBytes = 256 << 10
)

// UnpushedCommits is the bounded presentation of commits ahead of upstream.
type UnpushedCommits struct {
	Head, Upstream string
	Count          int
	Lines          []string
}

// LoadUnpushed loads commits reachable from HEAD but not from upstream.
func LoadUnpushed(ctx context.Context, runner Runner, upstream string, limit int) (UnpushedCommits, error) {
	if strings.TrimSpace(upstream) == "" {
		return UnpushedCommits{Upstream: ""}, nil
	}
	if strings.HasPrefix(upstream, "-") || strings.ContainsAny(upstream, "\r\n\x00") {
		return UnpushedCommits{}, fmt.Errorf("invalid upstream ref")
	}
	if limit <= 0 {
		limit = DefaultUnpushedCommits
	}
	if limit > MaxUnpushedCommits {
		limit = MaxUnpushedCommits
	}
	head, err := runner.Run(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return UnpushedCommits{Upstream: upstream}, err
	}
	countResult, err := runner.Run(ctx, "rev-list", "--count", "--end-of-options", upstream+"..HEAD")
	if err != nil {
		return UnpushedCommits{Head: strings.TrimSpace(string(head.Stdout)), Upstream: upstream}, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(countResult.Stdout)))
	if err != nil || count < 0 {
		return UnpushedCommits{}, fmt.Errorf("invalid unpushed commit count")
	}
	result, err := runner.Run(ctx, "log", "--graph", "--decorate", "--oneline", "--no-color", "--max-count", strconv.Itoa(limit), upstream+"..HEAD")
	if err != nil {
		return UnpushedCommits{Head: strings.TrimSpace(string(head.Stdout)), Upstream: upstream, Count: count}, err
	}
	if len(result.Stdout) > MaxUnpushedBytes {
		return UnpushedCommits{}, fmt.Errorf("unpushed commit output exceeds %d bytes", MaxUnpushedBytes)
	}
	text := strings.TrimSuffix(string(result.Stdout), "\n")
	lines := []string{}
	if text != "" {
		lines = strings.Split(text, "\n")
	}
	if len(lines) > limit*4+8 {
		lines = lines[:limit*4+8]
	}
	return UnpushedCommits{Head: strings.TrimSpace(string(head.Stdout)), Upstream: upstream, Count: count, Lines: lines}, nil
}
