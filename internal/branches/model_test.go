package branches

import (
	"errors"
	"testing"

	"github.com/jusanchez/gitwatch/internal/git"
)

func TestParse(t *testing.T) {
	v := Parse([]byte("main\x00abc\x00origin/main\x00*\x00>\x001700000000\x00initial commit\nfeature\x00def\x00\x00 \x00=\x001600000000\x00feature commit\n"))
	if len(v) != 2 || !v[0].Current || v[0].Ahead != 1 || v[0].LastCommitUnix != 1700000000 || v[0].Subject != "initial commit" || v[1].Remote {
		t.Fatal(v)
	}
}

func TestDeleteGuardsCurrentBranchAndExactConfirmation(t *testing.T) {
	branch := Branch{Name: "feature", Current: false}
	if _, err := Delete(nil, git.Runner{}, branch, DeletePrompt("feature", false), "wrong"); !errors.Is(err, ErrConfirmation) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	branch.Current = true
	if _, err := Delete(nil, git.Runner{}, branch, DeletePrompt("feature", false), "feature"); !errors.Is(err, ErrCurrentBranch) {
		t.Fatalf("expected current branch error, got %v", err)
	}
}

func TestListWithOccupancy(t *testing.T) {
	// The helper preserves branch parsing while attaching only local worktree occupancy.
	entries := []Branch{{Name: "main"}, {Name: "remotes/origin/main", Remote: true}}
	occupancy := map[string]string{"main": "/tmp/main"}
	AttachOccupancy(entries, occupancy)
	if entries[0].OccupiedPath != "/tmp/main" || entries[1].OccupiedPath != "" {
		t.Fatalf("occupancy = %#v", entries)
	}
}
