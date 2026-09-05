package reflogview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/reflog"
)

func TestViewShowsBoundedRecoveryPointsAndPaging(t *testing.T) {
	m := New("HEAD")
	m.SetPage([]reflog.Entry{{SHA: "abcdef1234567890", Timestamp: 1, Actor: "A\nB", Subject: "commit: restore"}}, true)
	view := m.View()
	for _, want := range []string{"Reflog · HEAD", "abcdef123456", "restore", "load more"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "A\nB") {
		t.Fatalf("unsanitized actor in view: %q", view)
	}
}

func TestMoveAndSelectedEntry(t *testing.T) {
	m := New("main")
	m.SetPage([]reflog.Entry{{SHA: "one"}, {SHA: "two"}}, false)
	m.Move(1)
	entry, ok := m.SelectedEntry()
	if !ok || entry.SHA != "two" {
		t.Fatalf("selected entry = %#v, %v", entry, ok)
	}
}
