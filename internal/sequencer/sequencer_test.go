package sequencer

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewStateRequiresRepositoryAndSupportsAllOperationKinds(t *testing.T) {
	for _, kind := range []Kind{KindUnknown, KindRebase, KindCherryPick, KindRevert, KindMerge, KindBisect} {
		state, err := NewState("repo-a", 7, kind, PhaseUnknown)
		if err != nil {
			t.Fatalf("kind %s: %v", kind, err)
		}
		if state.RepositoryID() != "repo-a" || state.Generation() != 7 || state.Kind() != kind {
			t.Fatalf("kind %s state = %#v", kind, state)
		}
	}
	if _, err := NewState("", 1, KindRebase, PhaseUnknown); err == nil {
		t.Fatal("empty repository ID was accepted")
	}
	if _, err := NewState("repo-a", 1, Kind(99), PhaseUnknown); err == nil {
		t.Fatal("invalid kind was accepted")
	}
}

func TestTypedDetailsAreValidatedAndCopied(t *testing.T) {
	state, err := NewState("repo-a", 2, KindCherryPick, PhaseActive)
	if err != nil {
		t.Fatal(err)
	}
	commits := []string{"a", "b"}
	state, err = state.WithDetails(Details{CherryPick: &CherryPickDetails{Commits: commits}})
	if err != nil {
		t.Fatal(err)
	}
	commits[0] = "changed"
	got := state.Details()
	if got.CherryPick == nil || got.CherryPick.Commits[0] != "a" {
		t.Fatalf("details were not copied: %#v", got)
	}
	got.CherryPick.Commits[0] = "mutated"
	if state.Details().CherryPick.Commits[0] != "a" {
		t.Fatal("details accessor exposed mutable state")
	}
	if _, err := state.WithDetails(Details{Rebase: &RebaseDetails{}}); err == nil {
		t.Fatal("wrong typed details were accepted")
	}
}

func TestWithObservationNormalizesConflictPathsAndCounters(t *testing.T) {
	state, err := NewState("repo-a", 3, KindMerge, PhaseActive)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(20, 0)
	state = state.WithObservation("head-2", "commit-2", -1, 4, []string{"z", "a", "z", ""}, at)
	if state.Remaining() != 0 || state.Completed() != 4 || state.HeadCurrent() != "head-2" || !state.UpdatedAt().Equal(at) {
		t.Fatalf("observation = %#v", state)
	}
	paths := state.ConflictPaths()
	if len(paths) != 2 || paths[0] != "a" || paths[1] != "z" {
		t.Fatalf("conflict paths = %#v", paths)
	}
	paths[0] = "mutated"
	if state.ConflictPaths()[0] != "a" {
		t.Fatal("conflict accessor exposed mutable state")
	}
}

func TestTransitionRules(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		event Event
		want  Phase
	}{
		{"start", PhaseUnknown, EventObserveActive, PhaseActive},
		{"pause", PhaseActive, EventPause, PhasePaused},
		{"resume", PhasePaused, EventResume, PhaseActive},
		{"complete", PhaseActive, EventComplete, PhaseCompleted},
		{"abort", PhasePaused, EventAbort, PhaseAborted},
		{"fail", PhaseActive, EventFail, PhaseFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, err := NewState("repo-a", 1, KindRebase, test.phase)
			if err != nil {
				t.Fatal(err)
			}
			next, err := Transition(state, test.event, time.Unix(30, 0))
			if err != nil {
				t.Fatal(err)
			}
			if next.Phase() != test.want || !next.UpdatedAt().Equal(time.Unix(30, 0)) {
				t.Fatalf("next phase=%s updated=%v", next.Phase(), next.UpdatedAt())
			}
			if state.Phase() != test.phase {
				t.Fatal("transition mutated the input state")
			}
		})
	}
	for _, test := range []struct {
		phase Phase
		event Event
	}{
		{PhaseUnknown, EventPause},
		{PhasePaused, EventComplete},
		{PhaseCompleted, EventObserveActive},
		{PhaseAborted, EventResume},
		{PhaseFailed, EventAbort},
	} {
		state, err := NewState("repo-a", 1, KindRebase, test.phase)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Transition(state, test.event, time.Time{}); err == nil || !strings.Contains(err.Error(), "invalid sequencer transition") {
			t.Fatalf("%s -> %s accepted or wrong error: %v", test.phase, test.event, err)
		}
	}
}

func TestApplyMessageRejectsStaleOrCrossRepositoryResults(t *testing.T) {
	state, err := NewState("repo-a", 4, KindRevert, PhaseActive)
	if err != nil {
		t.Fatal(err)
	}
	message := Message{RepositoryID: "repo-a", Generation: 4, State: state}
	if got, err := ApplyMessage(State{}, message, "repo-a", 4); err != nil || got.RepositoryID() != state.RepositoryID() || got.Generation() != state.Generation() || got.Kind() != state.Kind() || got.Phase() != state.Phase() {
		t.Fatalf("valid message result=%#v err=%v", got, err)
	}
	for _, invalid := range []struct {
		name             string
		message          Message
		activeRepository RepositoryID
		activeGeneration uint64
	}{
		{"other repository", Message{RepositoryID: "repo-b", Generation: 4, State: state}, "repo-a", 4},
		{"stale generation", message, "repo-a", 5},
		{"message scope", Message{RepositoryID: "repo-a", Generation: 5, State: state}, "repo-a", 4},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			if _, err := ApplyMessage(State{}, invalid.message, invalid.activeRepository, invalid.activeGeneration); err == nil {
				t.Fatal("invalid message was accepted")
			}
		})
	}
}

func TestApplyMessageInterleavesRepositoryGenerations(t *testing.T) {
	states := make([]State, 2)
	for i, repository := range []RepositoryID{"repo-a", "repo-b"} {
		state, err := NewState(repository, 1, KindMerge, PhaseActive)
		if err != nil {
			t.Fatal(err)
		}
		states[i] = state
	}
	var wait sync.WaitGroup
	for i, repository := range []RepositoryID{"repo-a", "repo-b"} {
		i, repository := i, repository
		wait.Add(1)
		go func() {
			defer wait.Done()
			for generation := uint64(1); generation <= 100; generation++ {
				state := states[i].WithObservation("head", "commit", 2, 1, nil, time.Unix(int64(generation), 0))
				state = state.WithObservation("head", "commit", int(generation), int(generation), nil, time.Unix(int64(generation), 0))
				message := Message{RepositoryID: repository, Generation: 1, State: state}
				if _, err := ApplyMessage(State{}, message, repository, 1); err != nil {
					t.Errorf("repository %s generation %d: %v", repository, generation, err)
				}
			}
		}()
	}
	wait.Wait()
}
