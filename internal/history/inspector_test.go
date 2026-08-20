package history

import (
	"errors"
	"testing"

	"github.com/jusanchez/gitwatch/internal/git"
)

func TestParseStats(t *testing.T) {
	stats := parseStats([]byte("3\t1\tinternal/app/app.go\n-\t-\tassets/logo.bin\n"))
	if len(stats) != 2 || stats[0].Added != 3 || stats[0].Deleted != 1 || !stats[1].Binary {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if got := statPaths([]byte("3\t1\tfile.txt\n")); len(got) != 1 || got[0] != "file.txt" {
		t.Fatalf("unexpected paths: %#v", got)
	}
}

func TestResolveRefRejectsOptionLikeInput(t *testing.T) {
	if _, err := ResolveRef(nil, git.Runner{}, "-bad"); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("expected invalid ref, got %v", err)
	}
}
