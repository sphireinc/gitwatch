package history

import (
	"fmt"
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestBoundedLogAndDiff(t *testing.T) {
	l := New(2)
	l.Add(Event{Kind: FileModified})
	l.Add(Event{Kind: FileStaged})
	l.Add(Event{Kind: OperationSuccess})
	if len(l.All()) != 2 {
		t.Fatal(len(l.All()))
	}
	old := repo.Snapshot{Branch: repo.Branch{Name: "main"}, Entries: []repo.Entry{{Path: repo.Path("a")}}}
	next := repo.Snapshot{Branch: repo.Branch{Name: "feature"}, Entries: []repo.Entry{{Path: repo.Path("a"), Staged: true}}}
	if len(Diff(old, next)) < 2 {
		t.Fatal(Diff(old, next))
	}
}

func TestDiffCoalescesLargeRefreshes(t *testing.T) {
	entries := make([]repo.Entry, 10_000)
	for i := range entries {
		entries[i].Path = repo.Path(fmt.Sprintf("file-%05d", i))
	}
	events := Diff(repo.Snapshot{}, repo.Snapshot{Entries: entries})
	if len(events) != maxDiffEvents {
		t.Fatalf("events = %d, want %d", len(events), maxDiffEvents)
	}
	last := events[len(events)-1]
	if last.Kind != ChangesCoalesced || last.Message != "9901 additional changes" {
		t.Fatalf("last event = %#v", last)
	}
}

func TestDiffDetectsRemovalsInLinearLookup(t *testing.T) {
	old := repo.Snapshot{Entries: []repo.Entry{{Path: repo.Path("keep")}, {Path: repo.Path("remove")}}}
	next := repo.Snapshot{Entries: []repo.Entry{{Path: repo.Path("keep")}}}
	events := Diff(old, next)
	if len(events) != 1 || events[0].Kind != FileRemoved || events[0].Path != "remove" {
		t.Fatalf("events = %#v", events)
	}
}
