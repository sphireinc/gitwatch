package app

import (
	"strings"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestStatusHealthDetailsDistinguishesUnloadedAndObservedEmptyLists(t *testing.T) {
	m := New()
	m.Snapshot = repo.Snapshot{
		Branch:     repo.Branch{Ahead: 3, Behind: 2},
		Counts:     repo.Counts{Staged: 1, Untracked: 2, Conflicted: 1},
		ObservedAt: time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC),
	}
	text := strings.Join(m.statusHealthDetails(), "\n")
	for _, want := range []string{"Dirty: 3 · conflicts: 1", "Ahead: 3 · behind: 2 · unpushed: 3", "operation: none", "Source: git status · observed: 2026-09-28T04:00:00Z", "Stashes: unknown · source: not loaded", "Worktrees: unknown · source: not loaded"} {
		if !strings.Contains(text, want) {
			t.Fatalf("health details missing %q:\n%s", want, text)
		}
	}
	m.StashesObservedAt = m.Snapshot.ObservedAt.Add(-time.Hour)
	m.WorktreesObservedAt = m.Snapshot.ObservedAt.Add(-2 * time.Hour)
	text = strings.Join(m.statusHealthDetails(), "\n")
	for _, want := range []string{"Stashes: 0 · source: cached Git list · observed: 2026-09-28T03:00:00Z", "Worktrees: 0 · source: cached Git list · observed: 2026-09-28T02:00:00Z"} {
		if !strings.Contains(text, want) {
			t.Fatalf("observed empty list missing %q:\n%s", want, text)
		}
	}
}

func TestCleanStatusDetailsIncludesHealthWithoutLoadingLists(t *testing.T) {
	m := New()
	text := strings.Join(m.statusDetailsLines(160, 30), "\n")
	for _, want := range []string{"Repository health", "observed: unknown", "Stashes: unknown", "Worktrees: unknown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("clean details missing %q:\n%s", want, text)
		}
	}
	if !m.StashesObservedAt.IsZero() || !m.WorktreesObservedAt.IsZero() {
		t.Fatal("rendering must not fabricate cache observations")
	}
}
