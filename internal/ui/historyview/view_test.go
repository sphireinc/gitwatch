package historyview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/history"
)

func TestSelectionPersistsAcrossRefreshAndFilter(t *testing.T) {
	commits := []history.Commit{{SHA: "one", Short: "one", Subject: "first"}, {SHA: "two", Short: "two", Subject: "second"}}
	m := New(commits)
	m.Move(1)
	m.SetCommits([]history.Commit{{SHA: "one", Short: "one", Subject: "first"}, {SHA: "two", Short: "two", Subject: "updated"}})
	if m.Selected != 1 {
		t.Fatalf("selected index = %d", m.Selected)
	}
	m.SetFilter("updated", []history.Commit{{SHA: "one", Short: "one", Subject: "first"}, {SHA: "two", Short: "two", Subject: "updated"}})
	if len(m.Rows) != 1 || m.Rows[0].Commit.SHA != "two" {
		t.Fatalf("filtered rows = %#v", m.Rows)
	}
	if !strings.Contains(m.View(), "updated") {
		t.Fatal("render omitted selected commit")
	}
}

func TestViewSanitizesCommitText(t *testing.T) {
	m := New([]history.Commit{{Short: "abc", Subject: "unsafe\x1b[31m"}})
	view := m.View()
	if strings.Contains(view, "\x1b") {
		t.Fatalf("escape sequence rendered: %q", view)
	}
}

func TestPulseOnlyChangesGraphMarker(t *testing.T) {
	m := New([]history.Commit{{SHA: "abcdef1", Short: "abcdef1", Subject: "commit"}})
	base := m.View()
	m.SetPulse(1)
	pulsed := m.View()
	if base == pulsed || !strings.Contains(pulsed, "◉") {
		t.Fatalf("pulse did not update marker: base=%q pulsed=%q", base, pulsed)
	}
}

func TestBasketSurvivesPaginationAndClearsOnScopeSwitch(t *testing.T) {
	m := New([]history.Commit{{SHA: "one", Short: "one", Subject: "one"}, {SHA: "two", Short: "two", Subject: "two"}})
	if err := m.SetScope("repo-a", "main", 1); err != nil {
		t.Fatal(err)
	}
	m.Selected = 1
	if err := m.ToggleBasket(); err != nil {
		t.Fatal(err)
	}
	m.SetCommits([]history.Commit{{SHA: "one", Short: "one", Subject: "one"}, {SHA: "two", Short: "two", Subject: "updated"}, {SHA: "three", Short: "three", Subject: "three"}})
	if m.Basket.Count() != 1 || m.Basket.SHAs()[0] != "two" || !strings.Contains(m.View(), "Basket: 1") {
		t.Fatalf("basket after pagination = %#v view=%q", m.Basket, m.View())
	}
	if err := m.SetScope("repo-b", "main", 2); err != nil {
		t.Fatal(err)
	}
	if m.Basket.Count() != 0 {
		t.Fatal("basket crossed repository scope")
	}
}
