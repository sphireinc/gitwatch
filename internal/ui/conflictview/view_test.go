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
