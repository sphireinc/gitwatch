package app

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/rebase"
	"github.com/sphireinc/git-watch/internal/ui/historyview"
	"github.com/sphireinc/git-watch/internal/workspace"
)

func TestHistoricalPatchSelectionBuildsEditPlan(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	commit := history.Commit{SHA: "abcdef1234567890", Short: "abcdef1", Subject: "change", Parents: []string{"parent"}}
	m.HistoryCommits = []history.Commit{commit}
	m.History = historyview.New([]history.Commit{commit})
	m.History.Selected = 0
	m.HistoryInspector = history.Inspector{Commit: commit, Diff: "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,1 +1,1 @@\n-old\n+new\n"}
	m.Workspace.Navigate(workspace.Log, "History")
	updated, _ := m.Update(key("H"))
	m = updated.(Model)
	if m.currentView() != workspace.Hunks || !m.HistoricalPatchMode {
		t.Fatalf("historical hunk route = view=%q mode=%v", m.currentView(), m.HistoricalPatchMode)
	}
	updated, _ = m.Update(key("A"))
	m = updated.(Model)
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.currentView() != workspace.Rebase || len(m.HistoricalPatch) == 0 || m.HistoricalRebaseTarget != commit.SHA {
		t.Fatalf("historical edit plan = view=%q patch=%d target=%q", m.currentView(), len(m.HistoricalPatch), m.HistoricalRebaseTarget)
	}
	entries := m.Rebase.Plan.Entries()
	if len(entries) == 0 || entries[0].Action() != rebase.Edit {
		t.Fatalf("historical plan entries = %#v", entries)
	}
}

func TestHistoricalPatchAppliedOpensAmendComposer(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.HistoricalRebaseAction = rebase.Edit
	m.HistoricalRebaseTarget = "target"
	m.HistoryCommits = []history.Commit{{SHA: "target", Subject: "remove line"}}
	updated, command := m.Update(HistoricalPatchAppliedMsg{Repository: m.repositoryGeneration})
	m = updated.(Model)
	if command == nil || m.currentView() != workspace.Commit || !m.Composer.Draft.Amend || m.Composer.Draft.Subject != "remove line" {
		t.Fatalf("amend route = command nil=%v view=%q amend=%v subject=%q", command == nil, m.currentView(), m.Composer.Draft.Amend, m.Composer.Draft.Subject)
	}
}
