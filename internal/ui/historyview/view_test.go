package historyview

import (
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/history"
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
