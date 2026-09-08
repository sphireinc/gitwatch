package compareview

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/compare"
)

func TestViewRendersResolvedSidesRenameAndBudgetNotices(t *testing.T) {
	m := Model{}
	m.SetResult(compare.Result{
		Left: compare.Revision{Ref: "HEAD~1", SHA: "left"}, Right: compare.Revision{Ref: "origin/main", SHA: "right"},
		Changes:        []compare.Change{{Status: "R100", OldPath: "old", NewPath: "new", Added: 1, Removed: 2}},
		FilesTruncated: true, Patch: "diff --git a/old b/new", PatchTruncated: true,
	})
	view := m.View()
	for _, want := range []string{"HEAD~1", "origin/main", "old -> new", "NOTICE: changed-file list truncated", "Patch truncated"} {
		if !contains(view, want) {
			t.Fatalf("comparison view missing %q: %s", want, view)
		}
	}
}

func TestMoveClampsSelection(t *testing.T) {
	m := Model{}
	m.SetResult(compare.Result{Changes: []compare.Change{{NewPath: "a"}, {NewPath: "b"}}})
	m.Move(10)
	if m.Selected != 1 {
		t.Fatalf("selected after down = %d", m.Selected)
	}
	m.Move(-10)
	if m.Selected != 0 {
		t.Fatalf("selected after up = %d", m.Selected)
	}
}

func contains(value, want string) bool {
	for index := 0; index+len(want) <= len(value); index++ {
		if value[index:index+len(want)] == want {
			return true
		}
	}
	return false
}
