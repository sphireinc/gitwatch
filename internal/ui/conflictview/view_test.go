package conflictview

import (
	"strings"
	"testing"
	"time"

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

func TestViewShowsBoundedContentMetadata(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindMerge, "main", []conflicts.Conflict{{Path: []byte("file")}})
	if !m.SetDetail(Detail{Path: []byte("file"), Ours: Content{Text: "ours"}, Theirs: Content{Binary: true}, Result: Content{Truncated: true}}) {
		t.Fatal("detail for selected conflict was rejected")
	}
	view := m.View(80, 24)
	for _, want := range []string{"Ours: ours", "Theirs: binary content", "Result: content exceeds display limit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q:\n%s", want, view)
		}
	}
}

func TestWideViewShowsOperationAndSideColumns(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindMerge, "feature", []conflicts.Conflict{{Path: []byte("file"), Ours: conflicts.Stage{OID: "ours"}, Theirs: conflicts.Stage{OID: "theirs"}}})
	view := m.View(140, 30)
	for _, want := range []string{"Operation: merge", "Target: feature", "Ours                 Theirs               Result", "ours", "theirs"} {
		if !strings.Contains(view, want) {
			t.Fatalf("wide view missing %q:\n%s", want, view)
		}
	}
}

func TestCherryPickViewShowsRepositoryScopedProgress(t *testing.T) {
	m := New()
	state, err := sequencer.NewState("repo", 4, sequencer.KindCherryPick, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithHistory("before", sequencer.Recovery{}, time.Time{})
	state = state.WithObservation("after", "current", 1, 1, []string{"file"}, time.Now())
	state, err = state.WithDetails(sequencer.Details{CherryPick: &sequencer.CherryPickDetails{Commits: []string{"first", "current"}, CurrentIndex: 1}})
	if err != nil {
		t.Fatal(err)
	}
	m.SetSnapshot(sequencer.KindCherryPick, "feature", []conflicts.Conflict{{Path: []byte("file")}})
	m.SetOperationState(&state)
	view := m.View(120, 30)
	for _, want := range []string{"Cherry-pick progress", "Original HEAD: before", "Progress: 1 completed · 1 remaining", "completed first", "conflicted current", "[s] skip"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestRecoveryFooterOnlyShowsSupportedActions(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindMerge, "main", nil)
	view := m.View(80, 24)
	if strings.Contains(view, "[s] skip") {
		t.Fatalf("merge footer exposed unsupported skip:\n%s", view)
	}
	m.SetSnapshot(sequencer.KindRebase, "main", nil)
	view = m.View(80, 24)
	if !strings.Contains(view, "[s] skip") {
		t.Fatalf("rebase footer omitted skip:\n%s", view)
	}
}

func TestRecoveryCoordinatorGatesContinueOnAuthoritativeState(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindCherryPick, "main", []conflicts.Conflict{{Path: []byte("file"), Resolution: "unmerged"}})
	m.SetStagedCount(1)
	if got := m.RecoveryActions(); got.Continue || !got.Skip || !got.Abort {
		t.Fatalf("unresolved recovery = %+v", got)
	}
	m.SetSnapshot(sequencer.KindCherryPick, "main", nil)
	m.SetStagedCount(0)
	if got := m.RecoveryActions(); got.Continue {
		t.Fatalf("clean index exposed continue = %+v", got)
	}
	m.SetStagedCount(1)
	if got := m.RecoveryActions(); !got.Continue || !got.Skip || !got.Abort {
		t.Fatalf("resolved recovery = %+v", got)
	}
	m.SetSnapshot(sequencer.KindMerge, "main", nil)
	if got := m.RecoveryActions(); !got.Continue || got.Skip || !got.Abort {
		t.Fatalf("merge recovery = %+v", got)
	}
}

func TestRecoveryCoordinatorSharesLifecycleRulesAcrossSequencers(t *testing.T) {
	for _, test := range []struct {
		kind     sequencer.Kind
		wantSkip bool
	}{
		{sequencer.KindRebase, true},
		{sequencer.KindCherryPick, true},
		{sequencer.KindRevert, true},
		{sequencer.KindMerge, false},
	} {
		t.Run(test.kind.String(), func(t *testing.T) {
			m := New()
			m.SetSnapshot(test.kind, "target", nil)
			m.SetStagedCount(1)
			recovery := m.RecoveryActions()
			if !recovery.Continue || recovery.Skip != test.wantSkip || !recovery.Abort {
				t.Fatalf("%s recovery = %+v", test.kind, recovery)
			}
		})
	}
}

func TestRevertViewUsesCommonProgressAndRecoveryPresentation(t *testing.T) {
	m := New()
	m.SetSnapshot(sequencer.KindRevert, "main", nil)
	m.SetStagedCount(1)
	state, err := sequencer.NewState("repo", 2, sequencer.KindRevert, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithHistory("before", sequencer.Recovery{}, time.Time{}).WithObservation("after", "current", 1, 1, nil, time.Now())
	state, err = state.WithDetails(sequencer.Details{Revert: &sequencer.RevertDetails{Commits: []string{"first", "current"}, Completed: []string{"first"}, CurrentIndex: 1}})
	if err != nil {
		t.Fatal(err)
	}
	m.SetOperationState(&state)
	view := m.View(120, 30)
	for _, want := range []string{"Revert progress", "Progress: 1 completed · 1 remaining", "completed first", "current current", "[s] skip"} {
		if !strings.Contains(view, want) {
			t.Fatalf("revert view missing %q:\n%s", want, view)
		}
	}
}
