package app

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/history"
	"github.com/jusanchez/gitwatch/internal/remotes"
	"github.com/jusanchez/gitwatch/internal/repo"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/ui/branchview"
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
	updated, cmd = m.Update(key("p"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.Status != "pushing" {
		t.Fatalf("push command/state = %v/%v/%q", cmd == nil, m.State, m.Status)
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
}

func contains(s, want string) bool {
	for i := 0; i+len(want) <= len(s); i++ {
		if s[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
