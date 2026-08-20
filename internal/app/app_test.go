package app

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/history"
	"github.com/jusanchez/gitwatch/internal/registry"
	"github.com/jusanchez/gitwatch/internal/remotes"
	"github.com/jusanchez/gitwatch/internal/repo"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/ui/branchview"
	"github.com/jusanchez/gitwatch/internal/ui/historyview"
	"github.com/jusanchez/gitwatch/internal/ui/remoteview"
	"github.com/jusanchez/gitwatch/internal/ui/stashview"
	"github.com/jusanchez/gitwatch/internal/ui/worktreeview"
	"github.com/jusanchez/gitwatch/internal/workspace"
	"github.com/jusanchez/gitwatch/internal/worktrees"
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
	updated, cmd = m.Update(RepositoryOpenedMsg{Path: "/repo", Discovery: git.Discovery{Root: "/repo"}})
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Status || m.Discovery.Root != "/repo" {
		t.Fatalf("repository opened = cmdnil=%v view=%q root=%q", cmd == nil, m.currentView(), m.Discovery.Root)
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
	updated, _ = m.Update(StashPreviewReadyMsg{Ref: "stash@{0}", Text: "-old\n+new"})
	m = updated.(Model)
	if got := m.View().Content; !contains(got, "Preview stash@{0}") || !contains(got, "+new") {
		t.Fatalf("stash preview missing: %q", got)
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
