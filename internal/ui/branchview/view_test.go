package branchview

import (
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/branches"
)

func TestView(t *testing.T) {
	m := New([]branches.Branch{{Name: "main", Current: true}})
	if !strings.Contains(m.View(), "main *") {
		t.Fatal(m.View())
	}
}

func TestViewShowsWorktreeOccupancy(t *testing.T) {
	m := New([]branches.Branch{{Name: "feature", OccupiedPath: "/tmp/linked"}})
	if !strings.Contains(m.View(), "worktree: /tmp/linked") {
		t.Fatal(m.View())
	}
}

func TestFilterSortAndPreserveSelection(t *testing.T) {
	m := New([]branches.Branch{
		{Name: "zeta", Upstream: "origin/zeta", Ahead: 1, LastCommitUnix: 10},
		{Name: "alpha", Upstream: "origin/alpha", Ahead: 3, LastCommitUnix: 20},
	})
	m.SetFilter("alp")
	if len(m.Entries) != 1 || m.Entries[0].Name != "alpha" {
		t.Fatalf("filtered entries = %#v", m.Entries)
	}
	m.SetFilter("")
	m.CycleSort()
	if m.Sort != SortAhead || m.Entries[0].Name != "zeta" {
		t.Fatalf("sorted entries = sort=%s entries=%#v", m.Sort, m.Entries)
	}
	m.Selected = 1
	m.SetEntries([]branches.Branch{{Name: "alpha", Ahead: 3}, {Name: "zeta", Ahead: 1}})
	if m.Selected != 1 || m.Entries[m.Selected].Name != "alpha" {
		t.Fatalf("selection was not preserved: selected=%d entries=%#v", m.Selected, m.Entries)
	}
}

func TestViewShowsBranchMetadata(t *testing.T) {
	m := New([]branches.Branch{{Name: "feature", Merged: true, LastCommitUnix: 1700000000, Ahead: 2, Behind: 1, Upstream: "origin/feature"}})
	view := m.View()
	for _, want := range []string{"sort: name", "ahead 2/behind 1", "merged", "last commit 2023-11-14"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q: %s", want, view)
		}
	}
}
