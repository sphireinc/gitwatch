package worktrees

import (
	"context"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestParse(t *testing.T) {
	v := Parse([]byte("worktree /tmp/main\nHEAD abc\nbranch refs/heads/main\n\nworktree /tmp/other\nHEAD def\ndetached\n"))
	if len(v) != 2 || v[1].Branch != "" || !v[1].Detached {
		t.Fatal(v)
	}
}

func TestOccupancyAndTargetValidation(t *testing.T) {
	entries := Parse([]byte("worktree /tmp/main\nHEAD abc\nbranch refs/heads/main\n"))
	if got := Occupancy(entries)["main"]; got != "/tmp/main" {
		t.Fatalf("unexpected occupancy: %#v", Occupancy(entries))
	}
	if _, err := Add(context.Background(), git.Runner{}, "-bad", ""); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected target validation, got %v", err)
	}
}

func TestAddWithCommitRejectsUnsafeArguments(t *testing.T) {
	if _, err := AddWithCommit(context.Background(), git.Runner{}, "/tmp/tree", "feature", "bad\nref"); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected invalid commit, got %v", err)
	}
}
