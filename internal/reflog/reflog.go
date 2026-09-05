// Package reflog provides a bounded, locale-independent recovery-point loader.
package reflog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
	MaxBytes     = 512 << 10
)

// Request selects one bounded reflog page. Ref defaults to HEAD; callers may
// provide HEAD, a local branch name, or another Git reflog selector.
type Request struct {
	Ref   string
	Limit int
	Skip  int
}

// Entry is one Git reflog recovery point.
type Entry struct {
	SHA       string
	Selector  string
	Timestamp int64
	Actor     string
	Subject   string
	Action    string
}

// Load reads one bounded reflog page through the typed Git runner.
func Load(ctx context.Context, runner git.Runner, request Request) ([]Entry, error) {
	ref := strings.TrimSpace(request.Ref)
	if ref == "" {
		ref = "HEAD"
	}
	if strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\r\n\x00") {
		return nil, fmt.Errorf("invalid reflog ref")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if request.Skip < 0 {
		return nil, fmt.Errorf("reflog skip must not be negative")
	}
	format := "%H%x00%gD%x00%ct%x00%an%x00%gs%x00"
	result, err := runner.RunBounded(ctx, MaxBytes, "reflog", "show", "--format="+format, "--max-count="+strconv.Itoa(limit), "--skip="+strconv.Itoa(request.Skip), ref)
	if err != nil {
		return nil, err
	}
	return parse(result.Stdout)
}

func parse(output []byte) ([]Entry, error) {
	fields := strings.Split(strings.TrimSuffix(string(output), "\n"), "\x00")
	if len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	if len(fields)%5 != 0 {
		return nil, fmt.Errorf("invalid reflog record framing")
	}
	entries := make([]Entry, 0, len(fields)/5)
	for index := 0; index < len(fields); index += 5 {
		timestamp, err := strconv.ParseInt(fields[index+2], 10, 64)
		if err != nil || timestamp < 0 {
			return nil, fmt.Errorf("invalid reflog timestamp %q", fields[index+2])
		}
		subject := fields[index+4]
		action := subject
		if colon := strings.Index(subject, ": "); colon >= 0 {
			action = subject[:colon]
		}
		entries = append(entries, Entry{
			SHA: fields[index], Selector: fields[index+1], Timestamp: timestamp,
			Actor: fields[index+3], Subject: subject, Action: action,
		})
	}
	return entries, nil
}
