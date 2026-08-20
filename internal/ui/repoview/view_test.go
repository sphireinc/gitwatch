package repoview

import (
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/registry"
)

func TestViewPreservesSelectionAndRendersState(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "one", Path: "/one"}, Branch: "main", State: "ready"}, {Repository: registry.Repository{Name: "two", Path: "/two"}, State: "inactive", Dirty: 2, Ahead: 1}})
	m.Move(1)
	m.SetRows([]registry.Row{{Repository: registry.Repository{Name: "two", Path: "/two"}, State: "inactive", Dirty: 2, Ahead: 1}})
	if m.Selected != 0 || !strings.Contains(m.View(), "two") || !strings.Contains(m.View(), "inactive") {
		t.Fatalf("repository view = selected=%d text=%q", m.Selected, m.View())
	}
}
