package app

import (
	"errors"
	"testing"
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
