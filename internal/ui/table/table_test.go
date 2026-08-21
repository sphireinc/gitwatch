package table

import (
	"fmt"
	"github.com/sphireinc/git-watch/internal/repo"
	"testing"
)

func TestTableFilteringSortingAndStableSelection(t *testing.T) {
	entries := make([]repo.Entry, 10000)
	for i := range entries {
		entries[i] = repo.Entry{Path: repo.Path(fmt.Sprintf("dir/%05d file.go", i)), Unstaged: i%2 == 0}
	}
	m := New(entries)
	m.SetFilter("9999")
	if len(m.Visible) != 1 {
		t.Fatal(len(m.Visible))
	}
	if m.SelectedPath() != "dir/09999 file.go" {
		t.Fatal(m.SelectedPath())
	}
	m.SetFilter("")
	m.Selected = 5
	selected := m.SelectedPath()
	m.SetEntries(entries)
	if m.SelectedPath() != selected {
		t.Fatalf("selection changed from %q to %q", selected, m.SelectedPath())
	}
}

func TestRowHitTesting(t *testing.T) {
	m := New([]repo.Entry{{Path: repo.Path("a")}, {Path: repo.Path("b")}})
	if e, ok := m.RowAt(4, 3, 5); !ok || string(e.Path) != "b" {
		t.Fatal(e, ok)
	}
}

func TestConflictFilter(t *testing.T) {
	m := New([]repo.Entry{{Path: repo.Path("clean")}, {Path: repo.Path("conflict"), Conflicted: true}})
	m.SetConflictFilter(true)
	if len(m.Visible) != 1 || string(m.Entries[m.Visible[0]].Path) != "conflict" {
		t.Fatal(m.Visible)
	}
}
