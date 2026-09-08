package app

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/pathhistory"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/workspace"
)

func TestPathHistoryFromStatusPreservesStatusContext(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path([]byte("dir/file name.txt")), Untracked: true}})
	m.Files.Selected = 0
	updated, command := m.Update(key("h"))
	m = updated.(Model)
	if command == nil || m.currentView() != workspace.PathHistory || m.PathHistory.Path != "dir/file name.txt" {
		t.Fatalf("path-history route = command nil=%v view=%q path=%q", command == nil, m.currentView(), m.PathHistory.Path)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.currentView() != workspace.Status || m.Files.SelectedPath() != "dir/file name.txt" {
		t.Fatalf("status context was not preserved: view=%q path=%q", m.currentView(), m.Files.SelectedPath())
	}
}

func TestPathHistoryReadyAndActions(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.openPathHistory("tracked.txt", false)
	m.PathHistoryRequest = 1
	m.PathHistoryGeneration = m.repositoryGeneration
	updated, _ := m.Update(PathHistoryReadyMsg{
		Path: "tracked.txt", Request: 1, Generation: m.repositoryGeneration,
		Entries: []pathhistory.Entry{{Commit: history.Commit{SHA: "abc", Short: "abc", Subject: "change", Parents: []string{"parent"}}, Kind: "M", OldPath: "tracked.txt", NewPath: "tracked.txt"}},
	})
	m = updated.(Model)
	if m.PathHistoryLoading || len(m.PathHistory.Entries) != 1 {
		t.Fatalf("history result = loading=%v entries=%d", m.PathHistoryLoading, len(m.PathHistory.Entries))
	}
	updated, command := m.Update(key("Y"))
	m = updated.(Model)
	if command == nil || m.currentView() != workspace.Compare || m.CompareLeft != "abc" || m.CompareRight != "HEAD" {
		t.Fatalf("compare route = command nil=%v view=%q refs=%q/%q", command == nil, m.currentView(), m.CompareLeft, m.CompareRight)
	}
	if !strings.Contains(m.PathHistory.View(), "change") {
		t.Fatal("path-history entry was not retained")
	}
}
