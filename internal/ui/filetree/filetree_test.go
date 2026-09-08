package filetree

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestTreeBuildsAggregatesAndPreservesPathSelection(t *testing.T) {
	entries := []repo.Entry{
		{Path: repo.Path("cmd/main.go"), Staged: true},
		{Path: repo.Path("cmd/test.go"), Unstaged: true},
		{Path: repo.Path("notes.md"), Untracked: true},
	}
	m := New(entries, []int{0, 1, 2}, "cmd/test.go")
	if len(m.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(m.Rows))
	}
	if m.Rows[0].Path != "cmd" || !m.Rows[0].Directory {
		t.Fatalf("first row = %#v, want cmd directory", m.Rows[0])
	}
	if got := m.Rows[0].Counts; got.Staged != 1 || got.Unstaged != 1 {
		t.Fatalf("directory counts = %#v, want staged=1 unstaged=1", got)
	}
	index, ok := m.SelectedEntryIndex()
	if !ok || index != 1 {
		t.Fatalf("selected entry = %d, %v, want index 1", index, ok)
	}
}

func TestTreeCollapseExpandAndNavigation(t *testing.T) {
	entries := []repo.Entry{
		{Path: repo.Path("a/one.txt")},
		{Path: repo.Path("a/two.txt")},
		{Path: repo.Path("b/three.txt")},
	}
	m := New(entries, []int{0, 1, 2}, "a/one.txt")
	m.Move(-1, 3)
	if !m.ToggleSelected() {
		t.Fatal("ToggleSelected() = false, want directory toggle")
	}
	if len(m.Rows) != 3 || m.Rows[1].Path != "b" {
		t.Fatalf("collapsed rows = %#v, want a, b, b/three.txt", m.Rows)
	}
	m.ExpandAll()
	if len(m.Rows) != 5 {
		t.Fatalf("expanded rows = %d, want 5", len(m.Rows))
	}
	m.Home(2)
	m.Move(1, 2)
	if m.Offset != 0 || m.Selected != 1 {
		t.Fatalf("selection = (%d,%d), want (1,0)", m.Selected, m.Offset)
	}
	m.End(2)
	if m.Selected != 4 || m.Offset != 3 {
		t.Fatalf("end selection = (%d,%d), want (4,3)", m.Selected, m.Offset)
	}
}

func TestTreeFiltersByCallerProvidedIndexes(t *testing.T) {
	entries := []repo.Entry{
		{Path: repo.Path("a/hidden.txt")},
		{Path: repo.Path("b/visible.txt")},
	}
	m := New(entries, []int{1}, "b/visible.txt")
	if len(m.Rows) != 2 || m.Rows[0].Path != "b" || m.Rows[1].Path != "b/visible.txt" {
		t.Fatalf("filtered rows = %#v", m.Rows)
	}
}
