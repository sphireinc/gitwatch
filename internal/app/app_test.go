package app

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/workspace"
)

func TestStateTransitions(t *testing.T) {
	m := New()
	updated, _ := m.Update(RefreshStartedMsg{})
	m = updated.(Model)
	if m.State != StateRefreshing {
		t.Fatal(m.State)
	}
	updated, _ = m.Update(RefreshFinishedMsg{})
	m = updated.(Model)
	if m.State != StateReady {
		t.Fatal(m.State)
	}
	updated, _ = m.Update(OperationStartedMsg{Name: "stage"})
	m = updated.(Model)
	if m.State != StateOperationPending {
		t.Fatal(m.State)
	}
	updated, _ = m.Update(OperationFinishedMsg{Name: "stage", Err: errors.New("failed")})
	m = updated.(Model)
	if m.State != StateError {
		t.Fatal(m.State)
	}
	updated, _ = m.Update(ModalMsg{Open: true, Name: "help"})
	m = updated.(Model)
	if m.State != StateModal || m.Modal != "help" {
		t.Fatal(m)
	}
}

func key(text string) tea.KeyPressMsg {
	if text == "esc" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func TestWorkspaceRoutesLoadAndRenderFeatureViews(t *testing.T) {
	m := New()
	updated, cmd := m.Update(key("b"))
	m = updated.(Model)
	if m.currentView() != workspace.Branches || cmd == nil {
		t.Fatalf("branch route = %q, cmd nil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(BranchesReadyMsg{Entries: []branches.Branch{{Name: "main", Current: true}}})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "main") {
		t.Fatalf("branch view missing entry: %q", got)
	}
	updated, cmd = m.Update(key("s"))
	m = updated.(Model)
	if m.currentView() != workspace.Stashes || cmd == nil {
		t.Fatalf("stash route = %q, cmd nil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(StashesReadyMsg{Entries: []stash.Entry{{Ref: "stash@{0}", Message: "work"}}})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "work") {
		t.Fatalf("stash view missing entry: %q", got)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.currentView() != workspace.Branches {
		t.Fatalf("escape route = %q", m.currentView())
	}
	updated, _ = m.Update(key("1"))
	m = updated.(Model)
	if m.currentView() != workspace.Status {
		t.Fatalf("status route = %q", m.currentView())
	}
}

func contains(s, want string) bool {
	for i := 0; i+len(want) <= len(s); i++ {
		if s[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
