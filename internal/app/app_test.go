package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/branches"
	"github.com/sphireinc/git-watch/internal/commands"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/notifications"
	"github.com/sphireinc/git-watch/internal/patch"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/rebase"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/remotes"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/stash"
	"github.com/sphireinc/git-watch/internal/ui/branchview"
	"github.com/sphireinc/git-watch/internal/ui/gitignoreview"
	"github.com/sphireinc/git-watch/internal/ui/historyview"
	"github.com/sphireinc/git-watch/internal/ui/hunkview"
	"github.com/sphireinc/git-watch/internal/ui/pluginview"
	"github.com/sphireinc/git-watch/internal/ui/remoteview"
	"github.com/sphireinc/git-watch/internal/ui/repoview"
	"github.com/sphireinc/git-watch/internal/ui/stashview"
	"github.com/sphireinc/git-watch/internal/ui/theme"
	"github.com/sphireinc/git-watch/internal/ui/worktreeview"
	"github.com/sphireinc/git-watch/internal/watch"
	"github.com/sphireinc/git-watch/internal/workspace"
	"github.com/sphireinc/git-watch/internal/worktrees"
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

func TestGitignoreNoFileCreationFlowRefreshesAuthoritativeStatus(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	updated, cmd := m.Update(key("I"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Gitignore {
		t.Fatalf("gitignore route = view=%q cmdnil=%v", m.currentView(), cmd == nil)
	}
	ready := cmd()
	updated, _ = m.Update(ready)
	m = updated.(Model)
	if !m.GitignoreMissing {
		t.Fatal("missing .gitignore was not recognized")
	}
	m.Gitignore.SetQuery("go")
	if len(m.Gitignore.Entries) == 0 {
		t.Fatal("Go template not available in creation browser")
	}
	m.Gitignore.Toggle()
	updated, cmd = m.Update(key("a"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("creation preview command missing")
	}
	preview := cmd()
	updated, _ = m.Update(preview)
	m = updated.(Model)
	if !m.GitignoreCreateConfirm || !strings.Contains(m.Gitignore.PreviewText, "after") {
		t.Fatalf("creation preview state = confirm=%v preview=%q", m.GitignoreCreateConfirm, m.Gitignore.PreviewText)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Status {
		t.Fatalf("creation confirmation = view=%q cmdnil=%v", m.currentView(), cmd == nil)
	}
	finished := cmd()
	updated, refresh := m.Update(finished)
	m = updated.(Model)
	if refresh == nil {
		t.Fatal("successful creation did not emit authoritative refresh command")
	}
	if _, err := os.Stat(filepath.Join(m.Discovery.Root, ".gitignore")); err != nil {
		t.Fatal(err)
	}
}

func TestGitignoreLoaderIgnoresStaleRepositoryGeneration(t *testing.T) {
	m := New()
	m.repositoryGeneration = 2
	m.GitignoreMissing = true
	updated, cmd := m.Update(GitignoreReadyMsg{Model: gitignoreview.RepositoryModel{RepositoryID: "old-repo"}, Generation: 1})
	if cmd != nil || updated.(Model).GitignoreMissing != true {
		t.Fatal("stale gitignore result replaced current state")
	}
}

func TestConflictWorkspaceRouteAndResolutionIntent(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}, Conflicts: []conflicts.Conflict{{Path: []byte("one")}, {Path: []byte("two")}}})
	updated, cmd := m.Update(key("C"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Conflict || len(m.Conflict.Conflicts) != 2 {
		t.Fatalf("conflict route = cmdnil=%v view=%q conflicts=%d", cmd == nil, m.currentView(), len(m.Conflict.Conflicts))
	}
	updated, cmd = m.Update(key("o"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("conflict resolution intent = cmdnil=%v state=%v", cmd == nil, m.State)
	}
}

func TestHistoricalRebaseEntryBuildsExplicitEditPlan(t *testing.T) {
	m := New()
	m.Snapshot.Branch.Name = "feature"
	commits := []history.Commit{
		{SHA: "new", Subject: "new", Parents: []string{"old"}},
		{SHA: "old", Subject: "old message", Parents: []string{"base"}},
	}
	m.HistoryCommits = commits
	m.History = historyview.New(commits)
	m.History.Selected = 1
	if cmd := m.openHistoricalRebase(rebase.Reword); cmd != nil {
		t.Fatal("historical plan unexpectedly scheduled work")
	}
	if m.currentView() != workspace.Rebase || m.Rebase.Base.Ref != "base" || m.HistoricalRebaseTarget != "old" {
		t.Fatalf("historical rebase state = view=%q base=%#v target=%q", m.currentView(), m.Rebase.Base, m.HistoricalRebaseTarget)
	}
	entries := m.Rebase.Plan.Entries()
	if len(entries) != 2 || entries[0].SHA() != "old" || entries[0].Action() != rebase.Reword {
		t.Fatalf("historical plan entries = %#v", entries)
	}
}

func TestWorktreeRouteLoadsAndRendersState(t *testing.T) {
	m := New()
	updated, cmd := m.Update(key("w"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Worktrees {
		t.Fatalf("worktree route = %q cmdnil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(WorktreesReadyMsg{Entries: []worktrees.Entry{{Path: "/linked", HEAD: "abc", Branch: "refs/heads/feature", Locked: true, Prunable: true}}})
	m = updated.(Model)
	for _, want := range []string{"/linked", "refs/heads/feature", "locked", "prunable"} {
		if !contains(m.View().Content, want) {
			t.Fatalf("worktree view missing %q: %s", want, m.View().Content)
		}
	}
}

func TestWorktreeMutationRouting(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Worktrees, "Worktrees")
	m.Worktrees = worktreeview.New([]worktrees.Entry{{Path: "/linked", HEAD: "abc"}})
	updated, cmd := m.Update(key("D"))
	m = updated.(Model)
	if cmd != nil || m.WorktreeConfirmAction != "remove" || m.WorktreeConfirmTarget != "/linked" {
		t.Fatalf("remove confirmation = cmdnil=%v action=%q target=%q", cmd == nil, m.WorktreeConfirmAction, m.WorktreeConfirmTarget)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.WorktreeConfirmAction != "" || !contains(m.Status, "cancelled") {
		t.Fatalf("remove cancellation = cmdnil=%v action=%q status=%q", cmd == nil, m.WorktreeConfirmAction, m.Status)
	}
	updated, cmd = m.Update(key("A"))
	m = updated.(Model)
	if cmd != nil || !m.WorktreeAddMode {
		t.Fatalf("add mode = cmdnil=%v mode=%v", cmd == nil, m.WorktreeAddMode)
	}
	for _, r := range "/tmp/new-tree" {
		updated, _ = m.Update(key(string(r)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.WorktreeAddMode || m.State != StateOperationPending {
		t.Fatalf("add execution = cmdnil=%v mode=%v state=%v", cmd == nil, m.WorktreeAddMode, m.State)
	}
	updated, cmd = m.Update(WorktreeOperationFinishedMsg{Operation: "added worktree", Target: "/tmp/new-tree"})
	m = updated.(Model)
	if cmd == nil || m.State != StateReady || !contains(m.Status, "complete") {
		t.Fatalf("add completion = cmdnil=%v state=%v status=%q", cmd == nil, m.State, m.Status)
	}
}

func TestHistoryLoadCancelsWhenLeavingView(t *testing.T) {
	m := New()
	updated, cmd := m.Update(key("l"))
	m = updated.(Model)
	if cmd == nil || m.HistoryCancel == nil || m.currentView() != workspace.Log {
		t.Fatalf("history load = cmdnil=%v cancel=%v view=%q", cmd == nil, m.HistoryCancel == nil, m.currentView())
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.HistoryCancel != nil || m.currentView() != workspace.Status {
		t.Fatalf("history cancellation = cancel=%v view=%q", m.HistoryCancel != nil, m.currentView())
	}
}

func TestHistoryRefJumpRouting(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{{SHA: "abc", Short: "abc", Subject: "target"}})
	updated, cmd := m.Update(key("g"))
	m = updated.(Model)
	if cmd != nil || !m.HistoryRefMode {
		t.Fatalf("ref mode = cmdnil=%v mode=%v", cmd == nil, m.HistoryRefMode)
	}
	for _, r := range "main" {
		updated, _ = m.Update(key(string(r)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.HistoryRefMode || m.State != StateOperationPending {
		t.Fatalf("ref resolve = cmdnil=%v mode=%v state=%v", cmd == nil, m.HistoryRefMode, m.State)
	}
	updated, _ = m.Update(HistoryRefReadyMsg{Ref: "main", SHA: "abc"})
	m = updated.(Model)
	if m.History.Selected != 0 || !contains(m.Status, "jumped to main") {
		t.Fatalf("ref result = selected=%d status=%q", m.History.Selected, m.Status)
	}
}

func TestCommandPaletteSearchAndExecution(t *testing.T) {
	m := New()
	m.Discovery.Root = "/repo"
	m.openPalette()
	if !m.PaletteMode || len(m.PaletteResults) == 0 {
		t.Fatalf("palette open = mode=%v results=%d", m.PaletteMode, len(m.PaletteResults))
	}
	updated, _ := m.Update(key("w"))
	m = updated.(Model)
	if !m.PaletteMode || !contains(m.PaletteQuery, "w") {
		t.Fatalf("palette query = mode=%v query=%q", m.PaletteMode, m.PaletteQuery)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.PaletteMode || m.currentView() != workspace.Worktrees {
		t.Fatalf("palette execution = mode=%v view=%q", m.PaletteMode, m.currentView())
	}
}

func TestOperationNotificationsAndToast(t *testing.T) {
	m := New()
	updated, _ := m.Update(RemoteOperationFinishedMsg{Operation: "push", Remote: "origin", Err: errors.New("rejected")})
	m = updated.(Model)
	if m.Notifications == nil || len(m.Notifications.Items()) != 1 || m.Toast.Text == "" || !m.Toast.Error {
		t.Fatalf("notification state = model=%v toast=%#v items=%#v", m.Notifications, m.Toast, m.Notifications.Items())
	}
	if !contains(m.View().Content, "NOTICE: push: rejected") {
		t.Fatalf("toast missing from view: %q", m.View().Content)
	}
}

func TestFeatureViewRendersToast(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Toast = ToastMsg{Text: "branch warning", Error: true}
	if got := m.View().Content; !contains(got, "NOTICE: branch warning") {
		t.Fatalf("feature view omitted toast: %q", got)
	}
}

func TestNotificationsClassifyConflictAndHookFailures(t *testing.T) {
	m := New()
	updated, _ := m.Update(SnapshotMsg{Snapshot: repo.Snapshot{Counts: repo.Counts{Conflicted: 2}}})
	m = updated.(Model)
	items := m.Notifications.Items()
	if len(items) != 1 || items[0].Kind != notifications.Conflict {
		t.Fatalf("conflict notification = %#v", items)
	}
	updated, _ = m.Update(CommitFinishedMsg{Err: errors.New("pre-commit failed")})
	m = updated.(Model)
	items = m.Notifications.Items()
	if len(items) != 2 || items[1].Kind != notifications.HookFailure {
		t.Fatalf("hook notification = %#v", items)
	}
}

func TestNotificationAttentionBadgeAndDismissal(t *testing.T) {
	m := New()
	m.notify(notifications.Conflict, notifications.Error, "conflict", "resolve", true)
	if !contains(m.View().Content, "[!] 1 attention") {
		t.Fatalf("attention badge missing: %q", m.View().Content)
	}
	updated, _ := m.Update(key("ctrl+n"))
	m = updated.(Model)
	if m.Notifications.Attention() != 0 || !contains(m.Status, "dismissed notification") {
		t.Fatalf("notification dismissal = attention=%d status=%q", m.Notifications.Attention(), m.Status)
	}
}

func TestViewSanitizesHostileRepositoryText(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("evil\x1b[31m.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	m.DiffPath, m.DiffText = "evil\x1b[31m.txt", "-old\x1b[2J\n+new token=secret"
	m.Theme = theme.New(theme.Dark, true)
	m.Width, m.Height = 120, 20
	view := m.View().Content
	if strings.Contains(view, "\x1b") || strings.Contains(view, "secret") || !strings.Contains(view, "�") {
		t.Fatalf("hostile text reached view: %q", view)
	}
}

func TestStatusViewAppliesConfiguredColorPolicy(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := New()
	m.Width, m.Height = 120, 20
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	if view := m.View().Content; !strings.Contains(view, "\x1b[") {
		t.Fatalf("the configured theme did not render terminal styles: %q", view)
	}

	t.Setenv("NO_COLOR", "1")
	m.Theme = theme.New(theme.Dark, false)
	if view := m.View().Content; strings.Contains(view, "\x1b[") {
		t.Fatalf("NO_COLOR view contains terminal styles: %q", view)
	}
}

func TestStatusDiffUsesWideRightPaneAndNarrowOverlay(t *testing.T) {
	m := New()
	m.Theme = theme.New(theme.Dark, true)
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	m.DiffPath, m.DiffText, m.DiffAdded, m.DiffDeleted = "notes.txt", "diff --git a/notes.txt b/notes.txt\n-old\n+new", 1, 1
	m.Width, m.Height = 160, 20
	wide := strings.Split(m.View().Content, "\n")
	if len(wide) != m.Height || !strings.Contains(wide[4], "│ Diff (unstaged) · notes.txt") {
		t.Fatalf("wide diff is not right-aligned: %#v", wide)
	}
	m.Width = 80
	narrow := m.View().Content
	if strings.Contains(narrow, "clean worktree") || !strings.Contains(narrow, "Diff (unstaged) · notes.txt") || !strings.Contains(narrow, "[esc] close") {
		t.Fatalf("narrow diff overlay = %q", narrow)
	}
}

func TestStatusPanelsWrapLongContent(t *testing.T) {
	m := New()
	m.Theme = theme.New(theme.Dark, true)
	m.Width, m.Height = 160, 20
	longPath := repo.Path("a/very/long/path/" + strings.Repeat("nested/", 16) + "filename-with-a-visible-tail.txt")
	m.Snapshot.Entries = []repo.Entry{{Path: longPath, Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	m.DiffPath = string(longPath)
	m.DiffText = "+" + strings.Repeat("right-panel-content ", 12) + "visible-tail"
	view := m.View().Content
	if !strings.Contains(view, "filename-with-a-visible-tail.txt") || !strings.Contains(view, "visible-tail") {
		t.Fatalf("wrapped panel content was truncated: %q", view)
	}
}

func TestDiffSearchAndBudget(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 20
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	m.DiffPath = "notes.txt"
	m.DiffText = "first line\nneedle here\nlast line"
	m.DiffMaxBytes, m.DiffMaxLines = 1<<20, 2
	if text, truncated := limitDiffText(m.DiffText, m.DiffMaxBytes, m.DiffMaxLines); !truncated || strings.Contains(text, "last line") {
		t.Fatalf("diff budget = %q truncated=%v", text, truncated)
	}
	m.DiffSearchInput = "needle"
	if !m.seekDiffMatch(0) || m.DiffOffset != 1 {
		t.Fatalf("diff search offset = %d", m.DiffOffset)
	}
}

func TestDiffStatIgnoresPatchHeaders(t *testing.T) {
	added, deleted := diffStat("diff --git a/a b/a\n--- a/a\n+++ b/a\n context\n-old\n+new\n++literal\n--literal\n")
	if added != 2 || deleted != 2 {
		t.Fatalf("diffstat = +%d -%d", added, deleted)
	}
}

func TestStatusEnterOpensDiffAndEscapeCancelsIt(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || !m.DiffLoading || m.DiffPath != "notes.txt" || m.DiffCancel == nil {
		t.Fatalf("open diff = commandnil=%v loading=%v path=%q", command == nil, m.DiffLoading, m.DiffPath)
	}
	request := m.DiffRequest
	updated, command = m.Update(key("esc"))
	m = updated.(Model)
	if command != nil || m.DiffPath != "" || m.DiffLoading || m.DiffRequest <= request {
		t.Fatalf("close diff = commandnil=%v loading=%v path=%q request=%d", command == nil, m.DiffLoading, m.DiffPath, m.DiffRequest)
	}
	updated, _ = m.Update(DiffReadyMsg{Path: "notes.txt", Text: "stale", Request: request})
	m = updated.(Model)
	if m.DiffPath != "" || m.DiffText != "" {
		t.Fatal("cancelled diff result was applied")
	}
}

func TestSnapshotClosesDiffWhenPathLeavesStatus(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	t.Cleanup(func() { _ = m.Close() })
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	updated, _ := m.Update(key("enter"))
	m = updated.(Model)
	if m.DiffPath != "notes.txt" || !m.DiffLoading {
		t.Fatalf("diff was not opened: path=%q loading=%v", m.DiffPath, m.DiffLoading)
	}
	request := m.DiffRequest
	m.applySnapshot(repo.Snapshot{Entries: []repo.Entry{{Path: repo.Path("other.txt"), Unstaged: true}}})
	if m.DiffPath != "" || m.DiffText != "" || m.DiffLoading || m.DiffRequest <= request {
		t.Fatalf("stale diff remained after path disappeared: path=%q text=%q loading=%v request=%d", m.DiffPath, m.DiffText, m.DiffLoading, m.DiffRequest)
	}
}

func TestStatusMouseClickOpensSelectedFileDiff(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	t.Cleanup(func() { _ = m.Close() })
	m.Width, m.Height = 160, 20
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("first.txt"), Unstaged: true}, {Path: repo.Path("second.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	updated, command := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 10, Y: 5})
	m = updated.(Model)
	if command == nil || m.Files.Selected != 1 || m.DiffPath != "second.txt" || !m.DiffLoading {
		t.Fatalf("mouse diff = commandnil=%v selected=%d path=%q loading=%v", command == nil, m.Files.Selected, m.DiffPath, m.DiffLoading)
	}
}

func TestStatusFilterSortConflictAndDetailsActivity(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 20
	m.Snapshot = repo.Snapshot{ObservedAt: time.Now(), Entries: []repo.Entry{
		{Path: repo.Path("zeta.txt"), Unstaged: true},
		{Path: repo.Path("beta.txt"), Staged: true, Conflicted: true},
	}}
	m.Files.SetEntries(m.Snapshot.Entries)
	updated, _ := m.Update(key("/"))
	m = updated.(Model)
	for _, character := range "bt" {
		updated, _ = m.Update(key(string(character)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.FileFilterMode || len(m.Files.Visible) != 1 || m.Files.SelectedPath() != "beta.txt" {
		t.Fatalf("filter = mode=%v visible=%v selected=%q", m.FileFilterMode, m.Files.Visible, m.Files.SelectedPath())
	}
	updated, _ = m.Update(key("S"))
	m = updated.(Model)
	if m.Files.Sort == 0 {
		t.Fatal("status sort did not advance")
	}
	updated, _ = m.Update(key("!"))
	m = updated.(Model)
	if !m.FileConflictOnly || len(m.Files.Visible) != 1 || m.Files.SelectedPath() != "beta.txt" {
		t.Fatalf("conflict filter = enabled=%v visible=%v", m.FileConflictOnly, m.Files.Visible)
	}
	m.recordActivity(history.OperationSuccess, "beta.txt", "stage complete")
	view := m.View().Content
	for _, expected := range []string{"Selected file details", "Path: beta.txt", "activity: operation success", "stage complete"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("status details missing %q: %q", expected, view)
		}
	}
}

func TestOpenDiffFollowsKeyboardSelection(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("a.txt"), Unstaged: true}, {Path: repo.Path("b.txt"), Unstaged: true}}
	m.Files.SetEntries(m.Snapshot.Entries)
	updated, first := m.Update(key("d"))
	m = updated.(Model)
	if first == nil || m.DiffPath != "a.txt" {
		t.Fatalf("first diff = commandnil=%v path=%q", first == nil, m.DiffPath)
	}
	firstRequest := m.DiffRequest
	updated, second := m.Update(key("j"))
	m = updated.(Model)
	if second == nil || m.Files.Selected != 1 || m.DiffPath != "b.txt" || m.DiffRequest <= firstRequest {
		t.Fatalf("selection diff = commandnil=%v selected=%d path=%q request=%d", second == nil, m.Files.Selected, m.DiffPath, m.DiffRequest)
	}
	m.closeDiff()
}

func TestRepositoriesRouteLoadsRows(t *testing.T) {
	m := New()
	m.Discovery.Root = "/repo"
	updated, cmd := m.Update(key("v"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Repositories {
		t.Fatalf("repository route = view=%q cmdnil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(RepositoriesReadyMsg{Rows: []registry.Row{{Repository: registry.Repository{Name: "repo", Path: "/repo"}, Branch: "main", State: "ready", Dirty: 1}}})
	m = updated.(Model)
	if !contains(m.View().Content, "repo") || !contains(m.View().Content, "dirty:1") {
		t.Fatalf("repository view missing row: %q", m.View().Content)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("repository open = cmdnil=%v state=%v", cmd == nil, m.State)
	}
	updated, cmd = m.Update(RepositoryOpenedMsg{Path: "/repo", Discovery: git.Discovery{Root: "/repo"}, PersistenceErr: errors.New("read-only registry")})
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Status || m.Discovery.Root != "/repo" {
		t.Fatalf("repository opened = cmdnil=%v view=%q root=%q", cmd == nil, m.currentView(), m.Discovery.Root)
	}
	if !m.Toast.Error || !contains(m.Toast.Text, "read-only registry") {
		t.Fatalf("repository persistence failure was hidden: %#v", m.Toast)
	}
}

func TestCommitTreeStatusPaneIsBoundedAndScrollable(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 30
	m.CommitTreeEnabled = true
	m.CommitTreeMaxCommits = 100
	m.CommitTreeLines = []string{"* abc123 first", "| * def456 second", "|/", "* 789abc third"}
	view := m.statusView()
	if !strings.Contains(view, "Commit tree") || !strings.Contains(view, "abc123") {
		t.Fatalf("commit tree missing from status view: %q", view)
	}
	lines := m.statusCommitTreeLines(40, 8)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != strings.Repeat("─", 39) {
		t.Fatalf("commit tree separator = %#v", lines)
	}
	if strings.TrimSpace(lines[1]) != "Commit tree · last 100" {
		t.Fatalf("commit tree heading moved unexpectedly = %#v", lines)
	}
	m.CommitTreeFocused = true
	m.scrollCommitTree(2)
	if m.CommitTreeOffset < 0 || m.CommitTreeOffset > len(m.CommitTreeLines) {
		t.Fatalf("invalid tree offset: %d", m.CommitTreeOffset)
	}
}

func TestCommitTreeRenderingUsesSafeThemeSegments(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := New()
	m.Width, m.Height = 160, 30
	m.CommitTreeEnabled = true
	m.CommitTreeLines = []string{"* \x1b[31mabc123\x1b[0m - subject \x1b[32m(2 days ago)\x1b[0m \x1b[1;34m<author>\x1b[0m\x1b]8;;https://evil.example\a"}
	m.Theme = theme.New(theme.Dark, true)
	view := m.statusView()
	if strings.Contains(view, "\x1b") || !strings.Contains(view, "abc123") || !strings.Contains(view, "author") {
		t.Fatalf("unsafe or missing colorless commit tree: %q", view)
	}
	m.Theme = theme.New(theme.Dark, false)
	view = m.statusView()
	if !strings.Contains(view, "\x1b[") {
		t.Fatal("colored theme did not render semantic commit-tree styles")
	}
}

func TestCommitTreeInitKeepsAsyncRequestOnLiveModel(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: t.TempDir()}, config.Config{
		ShowCommitTree: true,
		CommitTree:     config.CommitTreeConfig{MaxCommits: 100},
	})
	cmd := m.Init()
	if cmd == nil || m.CommitTreeRequest != 0 {
		t.Fatalf("init tree request state: cmdnil=%v request=%d", cmd == nil, m.CommitTreeRequest)
	}
	updated, _ := m.Update(CommitTreeReadyMsg{
		Generation: m.repositoryGeneration,
		Request:    0,
		Tree:       git.CommitTree{Head: "abc123", Lines: []string{"* abc123 commit"}},
	})
	m = updated.(Model)
	if len(m.CommitTreeLines) != 1 || m.CommitTreeHead != "abc123" {
		t.Fatalf("init tree result was discarded: %#v", m.CommitTreeLines)
	}
}

func TestStatusContextPaneShortcutsSelectUnpushedAndBranches(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 30
	m.Discovery.Root = t.TempDir()
	m.Snapshot.Branch = repo.Branch{Name: "main", Upstream: "origin/main"}
	m.UnpushedLines = []string{"* abc123 local work"}
	m.UnpushedCount = 1
	updated, command := m.Update(key("P"))
	m = updated.(Model)
	if command == nil || m.LowerPane != "unpushed" || !strings.Contains(m.statusView(), "Unpushed commits") {
		t.Fatalf("unpushed shortcut: commandnil=%v pane=%q", command == nil, m.LowerPane)
	}
	updated, _ = m.Update(key("B"))
	m = updated.(Model)
	if m.LowerPane != "branches" || !strings.Contains(m.statusView(), "Branches") {
		t.Fatalf("branch summary shortcut: pane=%q", m.LowerPane)
	}
}

func TestCommitTreeShortcutEnablesOnDemand(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 30
	m.Discovery.Root = t.TempDir()
	updated, command := m.Update(key("T"))
	m = updated.(Model)
	if command == nil || !m.CommitTreeEnabled || m.LowerPane != "commit-tree" || !m.CommitTreeLoading || m.CommitTreeRequest != 1 {
		t.Fatalf("on-demand commit tree: commandnil=%v enabled=%v pane=%q loading=%v request=%d", command == nil, m.CommitTreeEnabled, m.LowerPane, m.CommitTreeLoading, m.CommitTreeRequest)
	}
	if strings.Contains(m.Status, "disabled") {
		t.Fatalf("on-demand shortcut reported disabled: %q", m.Status)
	}
}

func TestStatusCommitInspectionPopulatesHistoricalFiles(t *testing.T) {
	m := New()
	m.Width, m.Height = 160, 30
	m.Discovery.Root = t.TempDir()
	m.StatusCommitRequest = 1
	updated, _ := m.Update(StatusCommitInspectorReadyMsg{
		Generation: 0,
		Request:    1,
		Inspector: history.Inspector{
			Commit: history.Commit{SHA: "abcdef1234567890", Short: "abcdef1"},
			Stats:  []history.FileStat{{Path: "historical.txt", Added: 2, Deleted: 1}},
		},
	})
	m = updated.(Model)
	if !m.StatusCommitActive || m.StatusCommitSHA != "abcdef1234567890" || m.Files.SelectedPath() != "historical.txt" {
		t.Fatalf("historical inspection = active=%v sha=%q path=%q", m.StatusCommitActive, m.StatusCommitSHA, m.Files.SelectedPath())
	}
	m.applySnapshot(repo.Snapshot{Branch: repo.Branch{Name: "main"}})
	if m.Files.SelectedPath() != "historical.txt" {
		t.Fatal("authoritative worktree refresh replaced historical file list")
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.StatusCommitActive {
		t.Fatal("escape did not return from historical inspection")
	}
}

func TestStatusCommitInspectionLoadsRealCommitFiles(t *testing.T) {
	root := t.TempDir()
	runner := git.NewRunner(root)
	ctx := context.Background()
	if _, err := runner.Run(ctx, "init", "-b", "main", "--", root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "historical.txt"), []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "add", "--", "historical.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "-c", "user.name=gitwatch", "-c", "user.email=gitwatch@example.com", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	shortResult, err := runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepository(git.Discovery{Root: root})
	m.Width, m.Height = 160, 30
	m.CommitTreeEnabled, m.LowerPane, m.CommitTreeFocused = true, "commit-tree", true
	m.CommitTreeLines = []string{"* " + strings.TrimSpace(string(shortResult.Stdout)) + " - initial"}
	m.StatusCommitSelectedLine = 0
	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command == nil {
		t.Fatal("commit inspection command was nil")
	}
	updated, _ = m.Update(command())
	m = updated.(Model)
	if !m.StatusCommitActive || m.Files.SelectedPath() != "historical.txt" {
		t.Fatalf("real inspection = active=%v path=%q err=%v", m.StatusCommitActive, m.Files.SelectedPath(), m.StatusCommitErr)
	}
}

func key(text string) tea.KeyPressMsg {
	if text == "esc" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	}
	if text == "tab" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	}
	if text == "enter" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	}
	if text == "ctrl+s" {
		return tea.KeyPressMsg(tea.Key{Text: "s", Code: 's', Mod: tea.ModCtrl})
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func TestCommitWorkspaceEditsDraftAndPreservesItOnFailure(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("file.txt"), Staged: true}}
	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	if m.currentView() != workspace.Commit || m.Composer.Focus != "subject" {
		t.Fatalf("composer route = %q focus=%q", m.currentView(), m.Composer.Focus)
	}
	for _, ch := range "add" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("tab"))
	m = updated.(Model)
	for _, ch := range "details" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	if m.Composer.Draft.Subject != "add" || m.Composer.Draft.Body != "details" || !m.Composer.Ready() {
		t.Fatalf("draft = %#v", m.Composer.Draft)
	}
	updated, _ = m.Update(CommitFinishedMsg{Err: errors.New("hook failed")})
	m = updated.(Model)
	if m.currentView() != workspace.Commit || m.Composer.Draft.Subject != "add" || m.State != StateError {
		t.Fatalf("failed commit changed workspace: view=%q draft=%#v state=%v", m.currentView(), m.Composer.Draft, m.State)
	}
}

func TestCommitComposerOptionsAndAmendConfirmation(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("file.txt"), Staged: true}}
	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	for _, option := range []string{"A", "N", "o", "S"} {
		updated, _ = m.Update(key(option))
		m = updated.(Model)
	}
	if !m.Composer.Draft.Amend || !m.Composer.Draft.NoEdit || !m.Composer.Draft.Signoff || !m.Composer.Draft.Sign {
		t.Fatalf("commit options = %#v", m.Composer.Draft)
	}
	updated, _ = m.Update(key("@"))
	m = updated.(Model)
	for _, ch := range "Ada Lovelace <ada@example.com>" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	if m.Composer.Draft.Author != "Ada Lovelace <ada@example.com>" || !m.CommitAuthorMode {
		t.Fatalf("author mode/draft = %q/%t", m.Composer.Draft.Author, m.CommitAuthorMode)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	updated, cmd := m.Update(key("ctrl+s"))
	m = updated.(Model)
	if cmd != nil || !m.CommitAmendConfirm || m.State == StateOperationPending {
		t.Fatalf("amend confirmation = cmdnil=%v confirm=%v state=%v view=%q draft=%#v validation=%#v key=%q", cmd == nil, m.CommitAmendConfirm, m.State, m.currentView(), m.Composer.Draft, m.Composer.Draft.Validate(), key("ctrl+s").String())
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.CommitAmendConfirm || m.State == StateOperationPending {
		t.Fatalf("amend cancellation = cmdnil=%v confirm=%v state=%v", cmd == nil, m.CommitAmendConfirm, m.State)
	}
}

func TestCommitFailureShowsHookOutputAndPreservesDraft(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("file.txt"), Staged: true}}
	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	m.Composer.SetSubject("keep this draft")
	updated, _ = m.Update(CommitFinishedMsg{Err: errors.New("hook failed"), HookOutput: "lint failed"})
	m = updated.(Model)
	if m.Composer.Draft.Subject != "keep this draft" || !contains(m.Status, "hook output:\nlint failed") {
		t.Fatalf("commit failure = draft=%q status=%q", m.Composer.Draft.Subject, m.Status)
	}
}

func TestCommitConfigIsShownInComposer(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("file.txt"), Staged: true}}
	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	updated, _ = m.Update(CommitConfigReadyMsg{Config: git.CommitConfig{UserName: "Ada", UserEmail: "ada@example.com", SignEnabled: true, SignFormat: "ssh"}})
	m = updated.(Model)
	if !m.CommitConfigReady || !strings.Contains(m.Composer.View(), "Ada <ada@example.com>") || !strings.Contains(m.Composer.View(), "configured signing: ssh") {
		t.Fatalf("composer config summary missing: %s", m.Composer.View())
	}
}

func TestCommitComposerMouseSelectsEditor(t *testing.T) {
	m := New()
	m.Snapshot.Entries = []repo.Entry{{Path: repo.Path("file.txt"), Staged: true}}
	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, Y: 7})
	m = updated.(Model)
	if m.Composer.Focus != "subject" {
		t.Fatalf("subject click focus = %q", m.Composer.Focus)
	}
	updated, _ = m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, Y: 9})
	m = updated.(Model)
	if m.Composer.Focus != "body" {
		t.Fatalf("body click focus = %q", m.Composer.Focus)
	}
}

func TestRemoteOperationTracksActiveJobAndCancellation(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, cmd := m.Update(key("f"))
	m = updated.(Model)
	if cmd == nil || m.RemoteCancel == nil || len(m.Remotes.Dashboard.ActiveJobs()) != 1 {
		t.Fatalf("remote job start = cmdnil=%v cancelnil=%v jobs=%#v", cmd == nil, m.RemoteCancel == nil, m.Remotes.Dashboard.Jobs)
	}
	updated, cmd = m.Update(key("esc"))
	m = updated.(Model)
	if cmd != nil || m.RemoteCancel != nil || m.Status != "remote operation cancellation requested" {
		t.Fatalf("remote cancellation = cmdnil=%v cancelnil=%v status=%q", cmd == nil, m.RemoteCancel == nil, m.Status)
	}
}

func TestPaletteAcceptsProviderOrPluginActions(t *testing.T) {
	m := New()
	run := false
	m.RegisterPaletteAction(commands.Action{ID: "provider-pr", Label: "Open pull request", Enabled: true}, func() tea.Cmd {
		run = true
		return nil
	})
	m.openPalette()
	found := false
	for _, result := range m.PaletteResults {
		if result.ID == "provider-pr" {
			found = true
		}
	}
	if !found {
		t.Fatal("registered palette action missing")
	}
	if cmd := m.executePaletteAction("provider-pr"); cmd != nil || !run {
		t.Fatalf("registered palette action execution = cmdnil=%v run=%v", cmd == nil, run)
	}
}

func TestPaletteAddsRepositoryAttentionJumpTarget(t *testing.T) {
	m := New()
	m.Repositories = repoview.New([]registry.Row{
		{Repository: registry.Repository{Name: "healthy", Path: "/healthy"}},
		{Repository: registry.Repository{Name: "needs-attention", Path: "/needs-attention"}, Operation: "cherry-pick", Attention: "cherry-pick"},
	})
	m.openPalette()
	found := ""
	for _, result := range m.PaletteResults {
		if strings.Contains(result.Label, "needs-attention") {
			found = result.ID
			break
		}
	}
	if found == "" {
		t.Fatal("repository attention palette target missing")
	}
	if cmd := m.executePaletteAction(found); m.currentView() != workspace.Repositories || m.Repositories.Selected != 1 {
		t.Fatalf("attention target execution = cmdnil=%v view=%q selected=%d", cmd == nil, m.currentView(), m.Repositories.Selected)
	}
}

func TestSelectedWorktreeOpensThroughRepositoryDiscovery(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Worktrees, "Worktrees")
	m.Worktrees = worktreeview.New([]worktrees.Entry{{Path: "/tmp/worktree", Branch: "main"}})
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.Status != "opening worktree" {
		t.Fatalf("worktree open = cmdnil=%v state=%v status=%q", cmd == nil, m.State, m.Status)
	}
}

func TestHistoryPulseRespectsMotionPolicy(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.Motion = MotionOff
	updated, _ := m.Update(TickMsg{})
	m = updated.(Model)
	if m.HistoryPulse != 0 {
		t.Fatalf("motion-off pulse = %d", m.HistoryPulse)
	}
	m.Motion = MotionFull
	updated, _ = m.Update(TickMsg{})
	m = updated.(Model)
	if m.HistoryPulse != 1 {
		t.Fatalf("motion-full pulse = %d", m.HistoryPulse)
	}
}

func TestConfiguredKeymapDispatchesCanonicalActions(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{}, config.Config{Keymap: map[string]string{"quit": "x", "help": "h"}})
	updated, cmd := m.Update(key("x"))
	m = updated.(Model)
	if cmd == nil || m.State != StateShutdown {
		t.Fatalf("remapped quit = cmdnil=%v state=%v", cmd == nil, m.State)
	}
}

func TestConfiguredPanelSplitReachesStatusLayout(t *testing.T) {
	c := config.Defaults()
	c.Layout.FilesPercent, c.Layout.DetailsPercent = 50, 50
	m := NewRepositoryWithConfig(git.Discovery{}, c)
	m.Width, m.Height = 200, 40
	status := m.statusLayout()
	if status.Files.Width != 100 || status.Details.Width != 100 {
		t.Fatalf("configured panel split = %#v", status)
	}
}

func TestRefreshCoordinatorFeedsAuthoritativeSnapshotToModel(t *testing.T) {
	root := t.TempDir()
	runner := git.NewRunner(root)
	if _, err := runner.Run(context.Background(), "init", "--", root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepositoryWithConfig(discovery, config.Defaults())
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	wait := waitForRefresh(m.RefreshCoordinator)
	updated, _ := m.Update(m.refresh()())
	m = updated.(Model)
	if m.State != StateRefreshing {
		t.Fatalf("state = %v", m.State)
	}
	updated, next := m.Update(wait())
	m = updated.(Model)
	if next == nil || m.State != StateReady || m.Snapshot.Generation != 1 || len(m.Snapshot.Entries) != 1 {
		t.Fatalf("state=%v generation=%d entries=%#v", m.State, m.Snapshot.Generation, m.Snapshot.Entries)
	}
}

func TestRefreshRequestCannotFinishBeforeRefreshingState(t *testing.T) {
	started := make(chan struct{})
	coordinator := git.NewRefreshCoordinator(func(context.Context, uint64) (repo.Snapshot, error) {
		close(started)
		return repo.Snapshot{}, nil
	})
	t.Cleanup(coordinator.Close)
	m := New()
	t.Cleanup(func() { _ = m.Close() })
	m.RefreshCoordinator = coordinator
	request := m.refresh()()
	select {
	case <-started:
		t.Fatal("refresh started before Bubble Tea applied the refreshing state")
	default:
	}
	updated, _ := m.Update(request)
	m = updated.(Model)
	if m.State != StateRefreshing {
		t.Fatalf("state = %v", m.State)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start after request was applied")
	}
}

func TestMutationCompletionAlwaysRequestsAuthoritativeRefresh(t *testing.T) {
	tests := []tea.Msg{
		OperationFinishedMsg{Name: "stage", Err: errors.New("failed")},
		PartialOperationFinishedMsg{Name: "partial stage", Err: errors.New("failed")},
		StashOperationFinishedMsg{Operation: "pop", Err: errors.New("failed")},
		BranchOperationFinishedMsg{Operation: "checkout", Err: errors.New("failed")},
		WorktreeOperationFinishedMsg{Operation: "remove", Err: errors.New("failed")},
		RemoteOperationFinishedMsg{Operation: "pull merge", Err: errors.New("failed")},
	}
	for _, message := range tests {
		m := New()
		updated, command := m.Update(message)
		m = updated.(Model)
		if command == nil || m.State != StateError {
			t.Errorf("%T: commandnil=%v state=%v", message, command == nil, m.State)
		}
		_ = m.Close()
	}
}

func TestStaleRepositoryMutationCompletionIsIgnored(t *testing.T) {
	m := New()
	m.repositoryGeneration = 2
	m.State, m.Status = StateReady, "current repository"
	updated, command := m.Update(OperationFinishedMsg{Name: "stage", Repository: 1, Err: errors.New("stale failure")})
	m = updated.(Model)
	if command != nil || m.State != StateReady || m.Status != "current repository" {
		t.Fatalf("stale mutation changed model: commandnil=%v state=%v status=%q", command == nil, m.State, m.Status)
	}
	_ = m.Close()
}

func TestFilesystemWatcherIsConnectedToModel(t *testing.T) {
	root := t.TempDir()
	settings := config.Defaults()
	settings.Interval = time.Hour
	settings.Reconciliation = time.Hour
	settings.Debounce = 5 * time.Millisecond
	m := NewRepositoryWithConfig(git.Discovery{Root: root}, settings)
	started, ok := m.startWatcher()().(watcherStartedMsg)
	if !ok || started.Manager == nil || started.Warning != nil || started.Generation != m.repositoryGeneration {
		t.Fatalf("watcher start = %#v", started)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	updated, wait := m.Update(started)
	m = updated.(Model)
	if wait == nil || m.WatchMode != watch.ModeFS || !contains(m.View().Content, "watch:fs") {
		t.Fatalf("mode=%s waitnil=%v view=%q", m.WatchMode, wait == nil, m.View().Content)
	}
	if err := os.WriteFile(filepath.Join(root, "changed.txt"), []byte("change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	event, ok := wait().(watcherEventMsg)
	if !ok || !event.Open || event.Event.Mode != watch.ModeFS {
		t.Fatalf("watcher event = %#v", event)
	}
	updated, command := m.Update(event)
	m = updated.(Model)
	if command == nil {
		t.Fatal("watcher event did not request refresh and resubscription")
	}
}

func TestGitHubWorkspaceLoadsAsynchronouslyWhenEnabled(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true, TokenEnv: "GITHUB_TOKEN"}})
	updated, cmd := m.Update(key("G"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.GitHub || m.State != StateLoading {
		t.Fatalf("GitHub route = cmdnil=%v view=%q state=%v", cmd == nil, m.currentView(), m.State)
	}
	updated, _ = m.Update(GitHubReadyMsg{Repository: provider.Repository{Owner: "octo", Name: "repo"}, Branch: "main", Pull: provider.PullRequest{Number: 1, Title: "Improve", State: "open"}, Checks: provider.ChecksSnapshot{Passing: 1}})
	m = updated.(Model)
	if !m.GitHub.Ready || m.State != StateReady || !strings.Contains(m.GitHub.View(), "PR #1") {
		t.Fatalf("GitHub result = ready=%v state=%v view=%s", m.GitHub.Ready, m.State, m.GitHub.View())
	}
}

func TestPluginWorkspaceTogglesSelectedEntry(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{}, config.Config{Plugins: config.PluginConfig{Enabled: true}})
	m.PluginStatePath = filepath.Join(t.TempDir(), "plugins.json")
	m.Workspace.Navigate(workspace.Plugins, "Plugins")
	m.Plugins = pluginview.New([]plugins.Entry{{Manifest: plugins.Manifest{ID: "one", Name: "One"}, Enabled: true, Healthy: true}})
	updated, cmd := m.Update(key("space"))
	m = updated.(Model)
	if cmd == nil || m.Plugins.Entries[0].Enabled || m.Status != "plugin one disabled" {
		t.Fatalf("plugin toggle = cmdnil=%v enabled=%v status=%q", cmd == nil, m.Plugins.Entries[0].Enabled, m.Status)
	}
	if _, ok := cmd().(PluginStateSavedMsg); !ok {
		t.Fatalf("plugin state command returned %T", cmd())
	}
}

func TestHunkWorkspaceSelectionAndDiscardConfirmation(t *testing.T) {
	m := New()
	m.DiffText = "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,2 @@\n keep\n-old\n+new\n"
	m.Workspace.Navigate(workspace.Status, "Status")
	updated, _ := m.Update(key("H"))
	m = updated.(Model)
	if m.currentView() != workspace.Hunks || len(m.Hunks.Files) != 1 {
		t.Fatalf("hunk route = view=%q files=%d", m.currentView(), len(m.Hunks.Files))
	}
	updated, _ = m.Update(key("a"))
	m = updated.(Model)
	if m.Hunks.Selection.Count() != 2 {
		t.Fatalf("hunk selection count = %d", m.Hunks.Selection.Count())
	}
	updated, cmd := m.Update(key("d"))
	m = updated.(Model)
	if cmd != nil || !m.HunkDiscardConfirm {
		t.Fatalf("discard confirmation = cmdnil=%v confirm=%v", cmd == nil, m.HunkDiscardConfirm)
	}
	for _, ch := range "discard" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.HunkDiscardConfirm || m.State != StateOperationPending {
		t.Fatalf("discard execution = cmdnil=%v confirm=%v state=%v", cmd == nil, m.HunkDiscardConfirm, m.State)
	}
}

func TestRemoteSetUpstreamAndTagPushControls(t *testing.T) {
	m := New()
	m.Snapshot.Branch.Name = "main"
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, _ := m.Update(key("u"))
	m = updated.(Model)
	if !m.RemotePushConfirm || !m.RemoteSetUpstream {
		t.Fatalf("upstream confirmation = %#v", m)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.RemoteSetUpstream != true || len(m.Remotes.Dashboard.ActiveJobs()) != 1 {
		t.Fatalf("upstream push = cmdnil=%v setup=%v jobs=%#v", cmd == nil, m.RemoteSetUpstream, m.Remotes.Dashboard.Jobs)
	}

	m = New()
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, _ = m.Update(key("T"))
	m = updated.(Model)
	for _, ch := range "v1.2.3" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.RemotePushConfirm || m.RemoteTag != "v1.2.3" {
		t.Fatalf("tag confirmation = confirm=%v tag=%q", m.RemotePushConfirm, m.RemoteTag)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || len(m.Remotes.Dashboard.ActiveJobs()) != 1 {
		t.Fatalf("tag push = cmdnil=%v jobs=%#v", cmd == nil, m.Remotes.Dashboard.Jobs)
	}
}

func TestRemoteDashboardMouseSelectsRemote(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}, {Name: "backup"}}})
	updated, cmd := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 2, Y: 7})
	m = updated.(Model)
	if cmd != nil || m.Remotes.Selected != 1 {
		t.Fatalf("remote mouse selection = cmdnil=%v selected=%d", cmd == nil, m.Remotes.Selected)
	}
}

func TestBranchCheckoutRejectsRemoteEntries(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true}})
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("checkout command/state = %v/%v", cmd == nil, m.State)
	}
	msg := cmd()
	finished, ok := msg.(BranchOperationFinishedMsg)
	if !ok || finished.Err == nil {
		t.Fatalf("remote checkout result = %#v", msg)
	}
}

func TestHistoryBranchCreationUsesExplicitNameAndTarget(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{{SHA: "abc123", Short: "abc123", Subject: "commit"}})
	updated, _ := m.Update(key("B"))
	m = updated.(Model)
	if !m.HistoryBranchCreating || m.HistoryBranchTarget != "abc123" {
		t.Fatalf("branch mode = %v target=%q", m.HistoryBranchCreating, m.HistoryBranchTarget)
	}
	for _, ch := range "feature" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.HistoryBranchCreating {
		t.Fatalf("branch command/state = cmdnil=%v state=%v creating=%v", cmd == nil, m.State, m.HistoryBranchCreating)
	}
	updated, _ = m.Update(HistoryActionFinishedMsg{Action: "created branch feature", Target: "abc123"})
	m = updated.(Model)
	if m.currentView() != workspace.Status || !contains(m.Status, "created branch feature") {
		t.Fatalf("branch completion = view=%q status=%q", m.currentView(), m.Status)
	}
}

func TestHistoryRevertRequiresExactSHA(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{{SHA: "abc123", Short: "abc123", Subject: "commit"}})
	updated, _ := m.Update(key("R"))
	m = updated.(Model)
	if !m.HistoryRevertConfirm || m.HistoryRevertTarget != "abc123" {
		t.Fatalf("revert mode = %v target=%q", m.HistoryRevertConfirm, m.HistoryRevertTarget)
	}
	for _, ch := range "abc12x" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.HistoryRevertConfirm || !contains(m.Status, "exact SHA") {
		t.Fatalf("wrong SHA accepted: confirm=%v status=%q", m.HistoryRevertConfirm, m.Status)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("R"))
	m = updated.(Model)
	for _, ch := range "abc123" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.HistoryRevertConfirm || m.State != StateOperationPending {
		t.Fatalf("exact SHA revert = cmdnil=%v confirm=%v state=%v", cmd == nil, m.HistoryRevertConfirm, m.State)
	}
}

func TestStashMutationRoutingAndConfirmation(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Stashes, "Stashes")
	m.Stashes = stashview.New([]stash.Entry{{Ref: "stash@{0}", Message: "work"}})
	updated, _ := m.Update(key("C"))
	m = updated.(Model)
	if !m.StashCreateMode {
		t.Fatal("stash create mode did not start")
	}
	if !m.StashIncludeUntracked {
		t.Fatal("stash create should include untracked by default")
	}
	updated, _ = m.Update(key("u"))
	m = updated.(Model)
	if m.StashIncludeUntracked {
		t.Fatal("stash include-untracked toggle did not turn off")
	}
	updated, _ = m.Update(key("u"))
	m = updated.(Model)
	for _, ch := range "save work" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.StashCreateMode {
		t.Fatalf("stash create = cmdnil=%v state=%v mode=%v", cmd == nil, m.State, m.StashCreateMode)
	}
	updated, _ = m.Update(StashOperationFinishedMsg{Operation: "created stash", Ref: "save work"})
	m = updated.(Model)
	if m.State != StateReady || !contains(m.Status, "created stash") {
		t.Fatalf("stash create completion = state=%v status=%q", m.State, m.Status)
	}
	updated, _ = m.Update(key("a"))
	m = updated.(Model)
	if m.StashConfirmAction != "apply" || m.StashConfirmRef != "stash@{0}" {
		t.Fatalf("stash apply confirmation = %q/%q", m.StashConfirmAction, m.StashConfirmRef)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	if m.StashConfirmAction != "" || !contains(m.Status, "cancelled") {
		t.Fatalf("stash cancellation = %q/%q", m.StashConfirmAction, m.Status)
	}
	updated, _ = m.Update(key("p"))
	m = updated.(Model)
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.StashConfirmAction != "" {
		t.Fatalf("stash pop execution = cmdnil=%v state=%v action=%q", cmd == nil, m.State, m.StashConfirmAction)
	}
	updated, _ = m.Update(StashOperationFinishedMsg{Operation: "pop", Ref: "stash@{0}", Err: errors.New("would clobber local changes")})
	m = updated.(Model)
	if m.State != StateError || !contains(m.Status, "would clobber") {
		t.Fatalf("stash conflict = state=%v status=%q", m.State, m.Status)
	}
	updated, _ = m.Update(key("B"))
	m = updated.(Model)
	if !m.StashBranchMode || m.StashBranchRef != "stash@{0}" {
		t.Fatalf("stash branch mode = %v/%q", m.StashBranchMode, m.StashBranchRef)
	}
	for _, ch := range "from-stash" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.StashBranchMode {
		t.Fatalf("stash branch execution = cmdnil=%v state=%v mode=%v", cmd == nil, m.State, m.StashBranchMode)
	}
}

func TestStashMouseSelectsAndPreviewsRow(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Stashes, "Stashes")
	m.Stashes = stashview.New([]stash.Entry{{Ref: "stash@{0}"}, {Ref: "stash@{1}"}})
	updated, cmd := m.Update(tea.MouseClickMsg{X: 2, Y: 4, Button: tea.MouseLeft})
	m = updated.(Model)
	if m.Stashes.Selected != 1 || cmd == nil {
		t.Fatalf("stash mouse selection = %d, cmd nil=%v", m.Stashes.Selected, cmd == nil)
	}
}

func TestHunkMouseSelectsChangedLine(t *testing.T) {
	files, err := patch.Parse("diff --git a/a b/a\n@@ -1 +1 @@\n-old\n+new\n")
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	m.Workspace.Navigate(workspace.Hunks, "Hunks")
	m.Hunks = hunkview.New(files)
	updated, cmd := m.Update(tea.MouseClickMsg{X: 2, Y: 4, Button: tea.MouseLeft})
	m = updated.(Model)
	if cmd != nil || m.Hunks.Selection.Count() != 1 || m.Hunks.Line != 1 {
		t.Fatalf("hunk mouse selection = cmdnil=%v line=%d selected=%d", cmd == nil, m.Hunks.Line, m.Hunks.Selection.Count())
	}
}

func TestRepositoryMouseSelectsRow(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	m.Repositories = repoview.New([]registry.Row{
		{Repository: registry.Repository{Name: "one", Path: "/one"}},
		{Repository: registry.Repository{Name: "two", Path: "/two"}},
	})
	updated, cmd := m.Update(tea.MouseClickMsg{X: 2, Y: 5, Button: tea.MouseLeft})
	m = updated.(Model)
	if cmd != nil || m.Repositories.Selected != 1 {
		t.Fatalf("repository mouse selection = cmdnil=%v selected=%d", cmd == nil, m.Repositories.Selected)
	}
}

func TestRepositoryDashboardFiltersAndSortsFromKeyboard(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	m.Repositories = repoview.New([]registry.Row{
		{Repository: registry.Repository{Name: "one", Path: "/one"}, Dirty: 1},
		{Repository: registry.Repository{Name: "two", Path: "/two"}, Dirty: 3},
	})
	updated, _ := m.Update(key("/"))
	m = updated.(Model)
	for _, character := range "two" {
		updated, _ = m.Update(key(string(character)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.RepositorySearching || len(m.Repositories.Rows) != 1 || m.Repositories.Rows[0].Repository.Name != "two" {
		t.Fatalf("repository filter = searching=%v rows=%#v", m.RepositorySearching, m.Repositories.Rows)
	}
	updated, _ = m.Update(key("s"))
	m = updated.(Model)
	if m.Repositories.Sort != registry.SortDirty {
		t.Fatalf("repository sort = %q", m.Repositories.Sort)
	}
}

func TestBranchMutationModesGuardAndBuildCommands(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{
		{Name: "main", Current: true},
		{Name: "feature", Upstream: "origin/feature"},
	})
	for i, branch := range m.Branches.Entries {
		if branch.Name == "main" {
			m.Branches.Selected = i
		}
	}
	updated, _ := m.Update(key("D"))
	m = updated.(Model)
	if m.BranchDeleteMode || !contains(m.Status, "cannot delete") {
		t.Fatalf("current branch delete guard = mode=%v status=%q", m.BranchDeleteMode, m.Status)
	}
	m.Branches.Selected = 1
	updated, _ = m.Update(key("c"))
	m = updated.(Model)
	for _, ch := range "new" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	if !m.BranchCreateMode || m.BranchMutationInput != "new" {
		t.Fatalf("create mode = %#v", m)
	}
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.BranchCreateMode || m.State != StateOperationPending {
		t.Fatalf("create submit = cmdnil=%v mode=%v state=%v", cmd == nil, m.BranchCreateMode, m.State)
	}
	updated, _ = m.Update(BranchOperationFinishedMsg{Operation: "created", Name: "new"})
	m = updated.(Model)
	if m.State != StateReady || !contains(m.Status, "created new") {
		t.Fatalf("create completion = state=%v status=%q", m.State, m.Status)
	}
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
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
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
	updated, _ = m.Update(StashPreviewReadyMsg{Ref: "stash@{0}", Text: "-old\n+new"})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "Preview stash@{0}") || !contains(got, "+new") {
		t.Fatalf("stash preview missing: %q", got)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.currentView() != workspace.Status {
		t.Fatalf("escape route = %q", m.currentView())
	}
	updated, _ = m.Update(key("1"))
	m = updated.(Model)
	if m.currentView() != workspace.Status {
		t.Fatalf("status route = %q", m.currentView())
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if m.currentView() != workspace.Remotes || cmd == nil {
		t.Fatalf("remote route = %q, cmd nil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(RemotesReadyMsg{Dashboard: remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin", FetchURL: "https://example.test/repo.git", PushURL: "https://example.test/repo.git", Reachable: true}}}})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "origin") || !contains(got, "reachable") {
		t.Fatalf("remote view missing data: %q", got)
	}
	updated, cmd = m.Update(key("f"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("fetch command/state = %v/%v", cmd == nil, m.State)
	}
	updated, _ = m.Update(RemoteOperationFinishedMsg{Operation: "fetch", Remote: "origin", Err: errors.New("network unavailable")})
	m = updated.(Model)
	if m.State != StateError || !contains(m.Status, "network unavailable") {
		t.Fatalf("fetch failure state/status = %v/%q", m.State, m.Status)
	}
	if len(m.Remotes.Dashboard.Activity) != 1 || m.Remotes.Dashboard.Activity[0].Success {
		t.Fatalf("remote failure activity = %#v", m.Remotes.Dashboard.Activity)
	}
	updated, _ = m.Update(RemoteOperationFinishedMsg{Operation: "pull merge", Remote: "origin", Err: errors.New("CONFLICT (content): merge conflict")})
	m = updated.(Model)
	if m.State != StateError || !contains(m.Status, "resolve conflicts") {
		t.Fatalf("conflict status = %v/%q", m.State, m.Status)
	}
	updated, cmd = m.Update(key("m"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || !contains(m.Status, "pulling merge") {
		t.Fatalf("pull command/state = %v/%v/%q", cmd == nil, m.State, m.Status)
	}
	updated, _ = m.Update(RemoteOperationFinishedMsg{Operation: "pull merge", Remote: "origin"})
	m = updated.(Model)
	if m.State != StateReady || !contains(m.Status, "pull merge complete") {
		t.Fatalf("pull completion state/status = %v/%q", m.State, m.Status)
	}
	m.Snapshot.Branch.Name = "main"
	updated, cmd = m.Update(key("p"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.Status != "preparing push preview" {
		t.Fatalf("push command/state = %v/%v/%q", cmd == nil, m.State, m.Status)
	}
	updated, _ = m.Update(PushPreviewReadyMsg{Preview: remotes.RefMovement{Remote: "origin", Branch: "main", LocalSHA: "local", RemoteSHA: "remote"}})
	m = updated.(Model)
	if !m.RemotePushConfirm || !contains(m.Status, "remote -> local") {
		t.Fatalf("push preview = %v/%q", m.RemotePushConfirm, m.Status)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.RemotePushConfirm || m.State != StateOperationPending || m.Status != "pushing" {
		t.Fatalf("push confirmation = cmdnil=%v confirm=%v state=%v status=%q", cmd == nil, m.RemotePushConfirm, m.State, m.Status)
	}
	updated, _ = m.Update(key("P"))
	m = updated.(Model)
	if !m.RemoteForceConfirm || !contains(m.Status, "force-with-lease") {
		t.Fatalf("force confirmation = %v/%q", m.RemoteForceConfirm, m.Status)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.RemoteForceConfirm || m.State != StateOperationPending || m.Status != "force pushing" {
		t.Fatalf("force confirmation acceptance = cmdnil=%v confirm=%v state=%v status=%q", cmd == nil, m.RemoteForceConfirm, m.State, m.Status)
	}
	updated, cmd = m.Update(key("l"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Log {
		t.Fatalf("history route = %q, cmd nil=%v", m.currentView(), cmd == nil)
	}
	updated, _ = m.Update(HistoryReadyMsg{Commits: []history.Commit{{SHA: "one", Short: "one", Subject: "first"}}, Skip: 0, HasMore: true})
	m = updated.(Model)
	if !m.HistoryHasMore || len(m.History.Rows) != 1 {
		t.Fatalf("history page state = more=%v rows=%d", m.HistoryHasMore, len(m.History.Rows))
	}
	updated, cmd = m.Update(key("]"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("history next-page command/state = %v/%v", cmd == nil, m.State)
	}
	updated, _ = m.Update(HistoryReadyMsg{Commits: []history.Commit{{SHA: "two", Short: "two", Subject: "second"}}, Skip: 1, HasMore: false})
	m = updated.(Model)
	if m.HistoryHasMore || len(m.History.Rows) != 2 {
		t.Fatalf("history append state = more=%v rows=%d", m.HistoryHasMore, len(m.History.Rows))
	}
	updated, _ = m.Update(HistoryInspectorReadyMsg{Inspector: history.Inspector{
		Commit: history.Commit{SHA: "two", Short: "two", Author: "Alice", Subject: "second", Parents: []string{"parent-one", "parent-two"}},
		Stats:  []history.FileStat{{Path: "file.txt", Added: 2, Deleted: 1}}, Diff: "+new",
	}})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "Selected commit: two") || !contains(got, "Parents: parent-one, parent-two") || !contains(got, "file.txt +2 -1") || !contains(got, "+new") {
		t.Fatalf("history inspector missing: %q", got)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || !contains(m.Status, "copied two") {
		t.Fatalf("copy SHA = cmdnil=%v status=%q", cmd == nil, m.Status)
	}
	updated, cmd = m.Update(key("M"))
	m = updated.(Model)
	if cmd == nil || m.HistoryInspectorParent != "parent-one" || m.State != StateOperationPending {
		t.Fatalf("parent inspection = cmdnil=%v parent=%q state=%v", cmd == nil, m.HistoryInspectorParent, m.State)
	}
	updated, _ = m.Update(key("f"))
	m = updated.(Model)
	if !m.HistoryInspectorPathMode {
		t.Fatal("path filter mode did not open")
	}
	for _, r := range "file.txt" {
		updated, _ = m.Update(key(string(r)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.HistoryInspectorPathMode || m.State != StateOperationPending {
		t.Fatalf("path inspection = cmdnil=%v mode=%v state=%v", cmd == nil, m.HistoryInspectorPathMode, m.State)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if !m.HistoryActionConfirm || !contains(m.Status, "checkout commit one") {
		t.Fatalf("history action confirmation = %v/%q", m.HistoryActionConfirm, m.Status)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	if m.HistoryActionConfirm || !contains(m.Status, "cancelled") {
		t.Fatalf("history action cancellation = %v/%q", m.HistoryActionConfirm, m.Status)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.HistoryActionConfirm || m.State != StateOperationPending {
		t.Fatalf("history action acceptance = cmdnil=%v confirm=%v state=%v", cmd == nil, m.HistoryActionConfirm, m.State)
	}
	updated, cmd = m.Update(key("t"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("tag loading command/state = %v/%v", cmd == nil, m.State)
	}
	updated, _ = m.Update(HistoryTagsReadyMsg{Tags: []history.Ref{{Name: "v1.0.0", OID: "abc123", Kind: "tag"}}})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "v1.0.0") || !contains(got, "abc123") {
		t.Fatalf("tag view missing: %q", got)
	}
	updated, _ = m.Update(key("/"))
	m = updated.(Model)
	if !m.HistorySearching {
		t.Fatal("history search did not start")
	}
	for _, ch := range "second" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	if m.History.Filter != "second" || len(m.History.Rows) != 1 || m.History.Rows[0].Commit.SHA != "two" {
		t.Fatalf("history filter = %q rows=%#v", m.History.Filter, m.History.Rows)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.HistorySearching || m.currentView() != workspace.Log {
		t.Fatalf("history search escape state = searching=%v view=%q", m.HistorySearching, m.currentView())
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
