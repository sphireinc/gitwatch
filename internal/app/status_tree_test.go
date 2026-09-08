package app

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestStatusTreeModeUsesFilteredEntriesAndDirectoryNavigation(t *testing.T) {
	m := New()
	m.Files.SetEntries([]repo.Entry{
		{Path: repo.Path("internal/app.go"), Staged: true},
		{Path: repo.Path("internal/view.go"), Unstaged: true},
		{Path: repo.Path("README.md"), Untracked: true},
	})
	m.Files.Selected = 2
	m.rebuildStatusFileTree()
	m.StatusTreeMode = true
	m.rebuildStatusFileTree()
	if len(m.FileTree.Rows) != 4 {
		t.Fatalf("tree rows = %d, want 4", len(m.FileTree.Rows))
	}
	if got := m.Files.SelectedPath(); string(got) != "internal/view.go" {
		t.Fatalf("selected path = %q, want internal/view.go", got)
	}
	for index, row := range m.FileTree.Rows {
		if row.Directory && row.Path == "internal" {
			m.FileTree.Selected = index
			break
		}
	}
	if !m.FileTree.ToggleSelected() {
		t.Fatal("directory was not toggleable")
	}
	if len(m.FileTree.Rows) != 2 {
		t.Fatalf("collapsed tree rows = %d, want 2", len(m.FileTree.Rows))
	}
	if m.FileTree.Rows[0].Path != "README.md" || m.FileTree.Rows[1].Path != "internal" {
		t.Fatalf("collapsed rows = %#v", m.FileTree.Rows)
	}
}

func TestStatusTreeToggleKeyDoesNotChangeAuthoritativeEntries(t *testing.T) {
	m := New()
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("a/file.txt")}})
	m.rebuildStatusFileTree()
	updated, _ := m.Update(key("O"))
	m = updated.(Model)
	if !m.StatusTreeMode || len(m.Files.Entries) != 1 || string(m.Files.Entries[0].Path) != "a/file.txt" {
		t.Fatalf("tree toggle changed source entries: mode=%v entries=%#v", m.StatusTreeMode, m.Files.Entries)
	}
}
