package history

import (
	"github.com/sphireinc/git-watch/internal/repo"
	"testing"
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
