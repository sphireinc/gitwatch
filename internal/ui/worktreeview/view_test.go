package worktreeview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/worktrees"
)

func TestViewAndSelectionPreservation(t *testing.T) {
	m := New([]worktrees.Entry{{Path: "/one", HEAD: "aaa", Branch: "refs/heads/main"}, {Path: "/two", HEAD: "bbb", Locked: true, LockNote: "busy"}})
	m.Move(1)
	m.SetEntries([]worktrees.Entry{{Path: "/two", HEAD: "bbb", Locked: true, LockNote: "busy"}, {Path: "/one", HEAD: "aaa", Branch: "refs/heads/main"}})
	if m.Selected != 0 {
		t.Fatalf("selected = %d", m.Selected)
	}
	view := m.View()
	for _, want := range []string{"/two", "locked", "lock: busy", "/one", "refs/heads/main"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q: %s", want, view)
		}
	}
}
