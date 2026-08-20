package branchview

import (
	"github.com/jusanchez/gitwatch/internal/branches"
	"strings"
	"testing"
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
