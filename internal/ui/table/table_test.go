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

func TestRefreshAndFilterKeepScrolledSelectionInViewport(t *testing.T) {
	entries := make([]repo.Entry, 10000)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("file/%05d.txt", index))}
	}
	m := New(entries)
	m.Move(len(entries)-1, 20)
	selected := m.SelectedPath()
	if m.Selected < m.Offset || m.Selected >= m.Offset+m.viewportRows {
		t.Fatalf("initial selection %d is outside viewport [%d,%d)", m.Selected, m.Offset, m.Offset+m.viewportRows)
	}

	updated := append([]repo.Entry{{Path: repo.Path("aaa.txt")}}, entries...)
	m.SetEntries(updated)
	if got := m.SelectedPath(); got != selected {
		t.Fatalf("refresh changed selected path from %q to %q", selected, got)
	}
	if m.Selected < m.Offset || m.Selected >= m.Offset+m.viewportRows {
		t.Fatalf("refreshed selection %d is outside viewport [%d,%d)", m.Selected, m.Offset, m.Offset+m.viewportRows)
	}

	m.SetFilter("file/00000")
	if m.Selected < m.Offset || m.Selected >= m.Offset+m.viewportRows {
		t.Fatalf("filtered selection %d is outside viewport [%d,%d)", m.Selected, m.Offset, m.Offset+m.viewportRows)
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
