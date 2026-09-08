package pathhistoryview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/pathhistory"
)

func TestSelectionAndPaging(t *testing.T) {
	m := Model{}
	m.SetPage("dir/file name.txt", []pathhistory.Entry{
		{Commit: history.Commit{Short: "abc1234", Author: "A", Subject: "first"}, Kind: "M", OldPath: "dir/file name.txt", NewPath: "dir/file name.txt"},
	}, true)
	if entry, ok := m.SelectedEntry(); !ok || entry.Commit.Short != "abc1234" {
		t.Fatalf("selected entry = %#v, %v", entry, ok)
	}
	m.AppendPage([]pathhistory.Entry{{Commit: history.Commit{Short: "def5678", Subject: "second"}, Kind: "D", OldPath: "dir/file name.txt"}}, false)
	m.Move(1)
	if m.Selected != 1 || m.HasMore {
		t.Fatalf("selection/page state = %d/%v", m.Selected, m.HasMore)
	}
	if got := m.View(); !strings.Contains(got, "def5678") || !strings.Contains(got, "dir/file name.txt") {
		t.Fatalf("view omitted history details:\n%s", got)
	}
}

func TestFollowModeIsVisible(t *testing.T) {
	m := Model{Path: "old name.txt", Follow: true}
	if got := m.View(); !strings.Contains(got, "rename-following") {
		t.Fatalf("follow mode not rendered: %s", got)
	}
}
