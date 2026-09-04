package conflictview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

func TestSetSnapshotKeepsSelectionAcrossAuthoritativeRefresh(t *testing.T) {
	m := New()
	values := []conflicts.Conflict{{Path: []byte("a"), Resolution: "unmerged"}, {Path: []byte("b"), Resolution: "resolved"}}
	m.SetSnapshot(sequencer.KindMerge, "main", values)
	m.Move(1)
	m.SetSnapshot(sequencer.KindMerge, "main", []conflicts.Conflict{values[1], values[0]})
	if m.Selected != 0 || m.ResolvedCount() != 1 {
		t.Fatalf("selection/count after refresh: %+v", m)
	}
}

func TestViewUsesTextLabelsAndStacksAtNarrowWidth(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindCherryPick, "target", []conflicts.Conflict{{Path: []byte("file"), Ours: conflicts.Stage{OID: "ours"}, Theirs: conflicts.Stage{OID: "theirs"}, Resolution: "unmerged"}})
	view := m.View(80, 24)
	for _, want := range []string{"Operation: cherry-pick", "Ours:", "Theirs:", "Result:", "[j/k] conflict"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q:\n%s", want, view)
		}
	}
}

func TestKeyMapsResolutionAndNavigationIntents(t *testing.T) {
	for key, want := range map[string]Action{"o": ActionChooseOurs, "t": ActionChooseTheirs, "b": ActionChooseBoth, "c": ActionContinue, "x": ActionAbort, "1": ActionStatus} {
		if got := Key(key); got != want {
			t.Fatalf("Key(%q)=%v, want %v", key, got, want)
		}
	}
}

func TestClickRequiresExplicitActionZone(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindMerge, "main", []conflicts.Conflict{{Path: []byte("file")}})
	if action, index := m.Click(2, 9, 80, 24); action != MouseSelectConflict || index != 0 {
		t.Fatalf("list click = %v, %d", action, index)
	}
	if action, _ := m.Click(2, 23, 80, 24); action != MouseChooseOurs {
		t.Fatalf("ours footer click = %v", action)
	}
	if action, _ := m.Click(70, 23, 80, 24); action != MouseStatus {
		t.Fatalf("status footer click = %v", action)
	}
}
