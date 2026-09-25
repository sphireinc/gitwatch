package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sphireinc/git-watch/internal/bisect"
	"github.com/sphireinc/git-watch/internal/branches"
	"github.com/sphireinc/git-watch/internal/commands"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/customcmd"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/history"
	mergeops "github.com/sphireinc/git-watch/internal/merge"
	"github.com/sphireinc/git-watch/internal/multirepo"
	"github.com/sphireinc/git-watch/internal/notifications"
	"github.com/sphireinc/git-watch/internal/operations"
	"github.com/sphireinc/git-watch/internal/patch"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/rebase"
	"github.com/sphireinc/git-watch/internal/reflog"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/remotes"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
	"github.com/sphireinc/git-watch/internal/stash"
	"github.com/sphireinc/git-watch/internal/submodules"
	"github.com/sphireinc/git-watch/internal/tags"
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
	publicplugin "github.com/sphireinc/git-watch/pkg/plugin"
)

func TestCustomCommandPromptFormBlocksExecutionUntilSubmit(t *testing.T) {
	m := New()
	defer func() { _ = m.Close() }()
	m.Discovery = git.Discovery{Root: t.TempDir()}
	m.CustomCommands = []customcmd.Definition{{
		Name:       "ticket",
		Executable: "tool",
		Prompts:    []customcmd.Prompt{{ID: "ticket", Label: "Ticket", Kind: customcmd.PromptText, Required: true}},
		Args:       []string{"--ticket={prompt:ticket}"},
	}}
	if command := m.runCustomCommand("ticket"); command != nil || m.CustomCommandForm == nil {
		t.Fatalf("prompt start = command nil %v form=%v", command == nil, m.CustomCommandForm != nil)
	}
	if command := m.updateCustomCommandForm("4"); command != nil || m.CustomCommandForm == nil {
		t.Fatalf("prompt edit = command nil %v form=%v", command == nil, m.CustomCommandForm != nil)
	}
	command := m.updateCustomCommandForm("enter")
	if command == nil || m.CustomCommandForm != nil {
		t.Fatalf("prompt submit = command nil %v form=%v", command == nil, m.CustomCommandForm != nil)
	}
}

func TestCustomCommandConfirmationCanBeCancelled(t *testing.T) {
	m := New()
	defer func() { _ = m.Close() }()
	m.Discovery = git.Discovery{Root: t.TempDir()}
	m.CustomCommands = []customcmd.Definition{{Name: "mutate", Executable: "tool", Confirm: true, Mutates: true}}
	if command := m.runCustomCommand("mutate"); command != nil || m.CustomCommandForm == nil {
		t.Fatalf("confirmation start = command nil %v form=%v", command == nil, m.CustomCommandForm != nil)
	}
	if command := m.updateCustomCommandForm("n"); command != nil || m.CustomCommandForm != nil || m.Status != "custom command cancelled" {
		t.Fatalf("confirmation cancel = command nil %v form=%v status=%q", command == nil, m.CustomCommandForm != nil, m.Status)
	}
}

func TestCustomCommandFormMouseSelectsAndAccepts(t *testing.T) {
	form, err := customcmd.NewForm([]customcmd.Prompt{
		{ID: "branch", Label: "Branch", Kind: customcmd.PromptSelect, Options: []string{"main", "feature"}},
		{ID: "targets", Label: "Targets", Kind: customcmd.PromptMultiSelect, Options: []string{"ui", "api"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	defer func() { _ = m.Close() }()
	m.CustomCommandForm, m.CustomCommandPending = &form, "not-found"

	updated, _ := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 0, Y: 4})
	m = updated.(Model)
	prompt, ok := m.CustomCommandForm.Current()
	if !ok || prompt.ID != "targets" {
		t.Fatalf("clicked select did not advance to next prompt: prompt=%#v", prompt)
	}
	updated, _ = m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 0, Y: 4})
	m = updated.(Model)
	if got := m.CustomCommandForm.SelectedOptions(); len(got) != 1 || got[0] != "api" {
		t.Fatalf("clicked multi-select option = %#v", got)
	}
	updated, cmd := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 0, Y: 6})
	m = updated.(Model)
	if cmd != nil || m.CustomCommandForm != nil || m.CustomCommandPromptValues["branch"] != "feature" || m.CustomCommandPromptValues["targets"] != "api" {
		t.Fatalf("clicked form accept = cmdnil=%v form=%v values=%#v", cmd == nil, m.CustomCommandForm != nil, m.CustomCommandPromptValues)
	}
}

func TestCustomCommandFormMouseCanCancelConfirmation(t *testing.T) {
	form, err := customcmd.NewForm([]customcmd.Prompt{{ID: "confirm", Label: "Confirm", Kind: customcmd.PromptConfirm}})
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	defer func() { _ = m.Close() }()
	m.CustomCommandForm, m.CustomCommandPending = &form, "not-found"
	updated, cmd := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 12, Y: 3})
	m = updated.(Model)
	if cmd != nil || m.CustomCommandForm != nil || m.Status != "custom command cancelled" {
		t.Fatalf("clicked confirmation cancel = cmdnil=%v form=%v status=%q", cmd == nil, m.CustomCommandForm != nil, m.Status)
	}
}

func TestCustomCommandFormMouseCanAcceptConfirmationAndText(t *testing.T) {
	t.Run("confirm", func(t *testing.T) {
		form, err := customcmd.NewForm([]customcmd.Prompt{{ID: "confirm", Label: "Confirm", Kind: customcmd.PromptConfirm}})
		if err != nil {
			t.Fatal(err)
		}
		m := New()
		defer func() { _ = m.Close() }()
		m.CustomCommandForm, m.CustomCommandPending = &form, "not-found"
		updated, cmd := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 1, Y: 3})
		m = updated.(Model)
		if cmd != nil || m.CustomCommandPromptValues["confirm"] != "true" {
			t.Fatalf("clicked confirmation accept = cmdnil=%v values=%#v", cmd == nil, m.CustomCommandPromptValues)
		}
	})
	t.Run("text", func(t *testing.T) {
		form, err := customcmd.NewForm([]customcmd.Prompt{{ID: "ticket", Label: "Ticket", Kind: customcmd.PromptText}})
		if err != nil {
			t.Fatal(err)
		}
		m := New()
		defer func() { _ = m.Close() }()
		m.CustomCommandForm, m.CustomCommandPending = &form, "not-found"
		if _, err := m.CustomCommandForm.Handle("A"); err != nil {
			t.Fatal(err)
		}
		updated, cmd := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 0, Y: 6})
		m = updated.(Model)
		if cmd != nil || m.CustomCommandPromptValues["ticket"] != "A" {
			t.Fatalf("clicked text accept = cmdnil=%v values=%#v", cmd == nil, m.CustomCommandPromptValues)
		}
	})
}

func TestCustomCommandSecretOutputIsHiddenFromStatus(t *testing.T) {
	const secret = "private-token-value"
	m := New()
	defer func() { _ = m.Close() }()
	updated, _ := m.Update(CustomCommandFinishedMsg{
		Name: "secret-test", Repository: m.repositoryGeneration, Output: customcmd.Output{Suppressed: true}, Err: errors.New(secret),
	})
	m = updated.(Model)
	if strings.Contains(m.Status, secret) || !strings.Contains(m.Status, "output hidden because a secret prompt was used") {
		t.Fatalf("secret completion status = %q", m.Status)
	}
}

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

func TestGitignoreWatcherInvalidatesOpenPreview(t *testing.T) {
	root := t.TempDir()
	m := NewRepositoryWithConfig(git.Discovery{Root: root}, config.Defaults())
	m.Workspace.Navigate(workspace.Gitignore, "Gitignore catalog")
	manager := &watch.Manager{}
	m.WatchManager = manager
	m.GitignoreCreateConfirm = true
	m.GitignoreCreatePlan = domain.MutationPlan{Root: root, Path: ".gitignore"}
	m.Gitignore.SetPreview("preview")
	updated, cmd := m.Update(watcherEventMsg{Manager: manager, Open: true, Event: watch.Event{Mode: watch.ModeFS, Path: filepath.Join(root, ".gitignore"), Operation: "WRITE"}})
	got := updated.(Model)
	if got.GitignoreCreateConfirm || got.GitignoreCreatePlan.Path != "" || got.Gitignore.PreviewText != "" {
		t.Fatalf("external edit left stale preview: confirm=%v plan=%#v preview=%q", got.GitignoreCreateConfirm, got.GitignoreCreatePlan, got.Gitignore.PreviewText)
	}
	if got.Status != "file changed externally; refresh preview" || cmd == nil {
		t.Fatalf("external edit handling: status=%q cmdnil=%v", got.Status, cmd == nil)
	}
}

func TestGitignoreCreationReturnsToStatusAfterMutation(t *testing.T) {
	m := New()
	m.GitignoreReturnToStatus = true
	m.Workspace.Navigate(workspace.Status, "Status")
	updated, cmd := m.Update(GitignoreMutationFinishedMsg{Action: "add", Repository: m.repositoryGeneration})
	got := updated.(Model)
	if got.currentView() != workspace.Status || got.GitignoreReturnToStatus || cmd == nil {
		t.Fatalf("creation completion: view=%q return=%v cmdnil=%v", got.currentView(), got.GitignoreReturnToStatus, cmd == nil)
	}
}

func TestGitignoreLoaderMakesOversizedDocumentReadOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewRepositoryWithConfig(git.Discovery{Root: root}, config.Config{GitignoreMaxBytes: 4})
	updated, cmd := m.Update(key("I"))
	m = updated.(Model)
	msg := cmd()
	updated, _ = m.Update(msg)
	got := updated.(Model)
	if !got.GitignoreReadOnly || got.GitignoreCreateConfirm || !strings.Contains(got.Status, "read-only") {
		t.Fatalf("oversized gitignore state: readOnly=%v confirm=%v status=%q", got.GitignoreReadOnly, got.GitignoreCreateConfirm, got.Status)
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

func TestActiveRebaseWithoutConflictsHasRecoveryRoute(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	state, err := sequencer.NewState("repo", 1, sequencer.KindRebase, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithObservation("head", "current-commit", 2, 3, nil, time.Now())
	state, err = state.WithDetails(sequencer.Details{Rebase: &sequencer.RebaseDetails{Interactive: true, EditStopped: true, TodoRemaining: 2, TodoCompleted: 3}})
	if err != nil {
		t.Fatal(err)
	}
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "feature"}, Operation: &state})
	m.Width = 200
	status := ansi.Strip(m.statusView())
	for _, want := range []string{"REBASE paused (edit-stop)", "current: current-commit"} {
		if !strings.Contains(status, want) {
			t.Fatalf("status view missing %q:\n%s", want, status)
		}
	}
	updated, cmd := m.Update(key("C"))
	m = updated.(Model)
	if cmd != nil || m.currentView() != workspace.Conflict || m.Conflict.Operation != sequencer.KindRebase {
		t.Fatalf("rebase recovery route = cmdnil=%v view=%q operation=%s", cmd != nil, m.currentView(), m.Conflict.Operation)
	}
	view := m.View().Content
	for _, want := range []string{"Rebase recovery", "[c] continue", "[x] abort"} {
		if !strings.Contains(view, want) {
			t.Fatalf("rebase recovery view missing %q:\n%s", want, view)
		}
	}
}

func TestRebaseRecoveryRecordsResultHeadAndRewrittenCount(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "base")
	runner := git.NewRunner(root)
	gitMustRunAppTest(t, ctx, runner, "switch", "-c", "feature")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("feature\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", "conflicting feature")
	if err := os.WriteFile(filepath.Join(root, "later.txt"), []byte("later\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "later.txt")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", "later feature")
	originalResult, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	original := strings.TrimSpace(string(originalResult.Stdout))
	gitMustRunAppTest(t, ctx, runner, "switch", "main")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", "conflicting main")
	gitMustRunAppTest(t, ctx, runner, "switch", "feature")
	if _, err := runner.Run(ctx, "rebase", "main"); err == nil {
		t.Fatal("expected conflict while rebasing")
	}
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil || snapshot.Operation == nil || snapshot.Operation.Kind() != sequencer.KindRebase {
		t.Fatalf("paused snapshot = %#v, err=%v", snapshot.Operation, err)
	}
	m := NewRepository(discovery)
	defer func() { _ = m.Close() }()
	m.repositoryGeneration = 1
	m.applySnapshot(snapshot)
	m.Workspace.Navigate(workspace.Conflict, "Rebase recovery")
	command := m.updateConflictKey("s")
	if command == nil {
		t.Fatal("skip was not scheduled")
	}
	message, ok := command().(OperationFinishedMsg)
	if !ok || message.Err != nil || message.Snapshot == nil || message.Snapshot.Operation != nil || message.Operation == nil {
		t.Fatalf("skip result = %#v", message)
	}
	if message.Operation.OldHead != original || message.Operation.NewHead != message.Snapshot.Branch.OID || !message.Operation.HasRewrittenCount || message.Operation.RewrittenCount != 1 {
		t.Fatalf("rebase completion record = %#v", message.Operation)
	}
	updated, refresh := m.Update(message)
	m = updated.(Model)
	if refresh == nil || m.currentView() != workspace.Status || !strings.Contains(m.Status, "1 rewritten commits") {
		t.Fatalf("rebase completion view = %q, workspace=%s, refreshnil=%v", m.Status, m.currentView(), refresh == nil)
	}
	events := m.ActivityLog.All()
	if len(events) == 0 || events[len(events)-1].Operation == nil || events[len(events)-1].Operation.RewrittenCount != 1 {
		t.Fatalf("rebase journal = %#v", events)
	}
	if !strings.Contains(journalEventDetails(&events[len(events)-1]), "rewritten commits: 1") {
		t.Fatalf("rebase journal details = %q", journalEventDetails(&events[len(events)-1]))
	}
}

func TestRebaseRecoveryKeepsWorkspaceWhenNextCommitConflicts(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "base")
	runner := git.NewRunner(root)
	gitMustRunAppTest(t, ctx, runner, "switch", "-c", "feature")
	for _, value := range []string{"feature one", "feature two"} {
		if err := os.WriteFile(filepath.Join(root, "README"), []byte(value+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
		gitMustRunAppTest(t, ctx, runner, "commit", "-m", value)
	}
	gitMustRunAppTest(t, ctx, runner, "switch", "main")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", "main")
	gitMustRunAppTest(t, ctx, runner, "switch", "feature")
	if _, err := runner.Run(ctx, "rebase", "main"); err == nil {
		t.Fatal("expected first conflict")
	}
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepository(discovery)
	defer func() { _ = m.Close() }()
	m.repositoryGeneration = 1
	m.applySnapshot(snapshot)
	m.Workspace.Navigate(workspace.Conflict, "Rebase recovery")
	command := m.updateConflictKey("s")
	if command == nil {
		t.Fatal("skip was not scheduled")
	}
	message, ok := command().(OperationFinishedMsg)
	if !ok || message.Snapshot == nil || message.Snapshot.Operation == nil || message.Operation == nil {
		t.Fatalf("next conflict result = %#v", message)
	}
	if message.Operation.HasRewrittenCount {
		t.Fatalf("intermediate action claimed a final rewrite count: %#v", message.Operation)
	}
	updated, refresh := m.Update(message)
	m = updated.(Model)
	if refresh == nil || m.currentView() != workspace.Conflict || strings.Contains(m.Status, "rebase complete:") || m.Snapshot.Operation == nil {
		t.Fatalf("intermediate recovery = %q, workspace=%s, operation=%#v", m.Status, m.currentView(), m.Snapshot.Operation)
	}
}

func TestStatusViewShowsLocalHealthAndSigningSource(t *testing.T) {
	m := New()
	m.Width = 120
	m.Height = 30
	m.Discovery.Root = t.TempDir()
	m.CommitConfigReady = true
	m.CommitConfig.SignEnabled = true
	m.CommitConfig.SignFormat = ""
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}, Counts: repo.Counts{Untracked: 2}, ObservedAt: time.Date(2026, 9, 22, 12, 34, 56, 0, time.UTC)})
	status := ansi.Strip(m.statusView())
	for _, want := range []string{"HEALTH warning", "source:git status", "attention:signing format unset", "observed:12:34:56"} {
		if !strings.Contains(status, want) {
			t.Fatalf("status health missing %q:\n%s", want, status)
		}
	}
}

func TestActiveCherryPickCanReopenProgressFromPalette(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	state, err := sequencer.NewState("repo", 1, sequencer.KindCherryPick, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithObservation("head", "current", 1, 1, nil, time.Now())
	state, err = state.WithDetails(sequencer.Details{CherryPick: &sequencer.CherryPickDetails{Commits: []string{"first", "current"}, CurrentIndex: 1}})
	if err != nil {
		t.Fatal(err)
	}
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "feature"}, Operation: &state})
	updated, cmd := m.Update(key("ctrl+p"))
	m = updated.(Model)
	if cmd != nil || !m.PaletteMode {
		t.Fatalf("palette open = mode=%v cmdnil=%v", m.PaletteMode, cmd != nil)
	}
	if !contains(m.paletteView().Content, "Reopen active cherry-pick") {
		t.Fatalf("palette missing cherry-pick recovery: %s", m.paletteView().Content)
	}
	cmd = m.executePaletteAction("cherry_pick_recovery")
	if cmd != nil || m.currentView() != workspace.CherryPick || m.Conflict.Operation != sequencer.KindCherryPick {
		t.Fatalf("palette recovery route = view=%q operation=%s cmdnil=%v", m.currentView(), m.Conflict.Operation, cmd != nil)
	}
}

func TestRevertSequencerRefreshRoutesToConflictWorkspace(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.HistoryRevertRunning = true
	state, err := sequencer.NewState("repo", 1, sequencer.KindRevert, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	updated, cmd := m.Update(SnapshotMsg{Snapshot: repo.Snapshot{
		Root:      m.Discovery.Root,
		Branch:    repo.Branch{Name: "main"},
		Operation: &state,
		Conflicts: []conflicts.Conflict{{Path: []byte("file")}},
	}})
	got := updated.(Model)
	if cmd != nil || got.currentView() != workspace.Conflict || got.Conflict.Operation != sequencer.KindRevert {
		t.Fatalf("revert recovery route = cmdnil=%v view=%q operation=%s", cmd == nil, got.currentView(), got.Conflict.Operation)
	}
	if got.Status != "revert paused for conflict recovery" {
		t.Fatalf("revert recovery status = %q", got.Status)
	}
}

func TestSubmoduleHealthLoadsOutsideAuthoritativeSnapshotAndRendersSummary(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.repositoryGeneration = 1
	updated, command := m.Update(SnapshotMsg{Generation: 1, Snapshot: repo.Snapshot{
		Root:       m.Discovery.Root,
		Generation: 1,
		Branch:     repo.Branch{Name: "main"},
	}})
	m = updated.(Model)
	if command == nil || !m.SubmodulesLoading {
		t.Fatalf("submodule load = cmdnil=%v loading=%v", command == nil, m.SubmodulesLoading)
	}
	updated, _ = m.Update(SubmodulesReadyMsg{Generation: 1, Snapshot: submodules.Snapshot{
		Repository: m.Discovery.Root,
		Modules: []submodules.Module{
			{Path: "clean", State: submodules.StateClean},
			{Path: "dirty", State: submodules.StateDirty},
		},
	}})
	m = updated.(Model)
	status := ansi.Strip(m.statusView())
	if m.SubmodulesLoading || m.SubmodulesErr != nil || !strings.Contains(status, "SUBMODULES 1 clean  1 attention") {
		t.Fatalf("submodule summary = loading=%v err=%v view=%q", m.SubmodulesLoading, m.SubmodulesErr, m.statusView())
	}
}

func TestTagsWorkspaceLoadsFiltersAndSortsBoundedSnapshot(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Status, "Status")
	updated, command := m.Update(key("t"))
	m = updated.(Model)
	if command == nil || m.currentView() != workspace.Tags || !m.TagsLoading {
		t.Fatalf("tags navigation = cmdnil=%v view=%q loading=%v", command == nil, m.currentView(), m.TagsLoading)
	}
	updated, _ = m.Update(TagsReadyMsg{Generation: 1, Snapshot: tags.Snapshot{Repository: m.Discovery.Root, Tags: []tags.Tag{
		{Name: "v2", TargetID: "bbb", Kind: tags.Lightweight, RemotePresence: tags.RemoteAbsent},
		{Name: "v1", TargetID: "aaa", Kind: tags.Annotated, RemotePresence: tags.RemotePresent, RemoteNames: []string{"origin"}},
	}}})
	m = updated.(Model)
	if m.TagsLoading || m.TagsErr != nil || !strings.Contains(m.View().Content, "v1") {
		t.Fatalf("tags loaded = loading=%v err=%v view=%q", m.TagsLoading, m.TagsErr, m.View().Content)
	}
	updated, _ = m.Update(key("s"))
	m = updated.(Model)
	if m.TagsSort != "target" {
		t.Fatalf("tag sort = %q", m.TagsSort)
	}
	updated, _ = m.Update(key("/"))
	m = updated.(Model)
	updated, _ = m.Update(key("v"))
	m = updated.(Model)
	updated, _ = m.Update(key("1"))
	m = updated.(Model)
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.TagsFilterMode || m.TagsFilter != "v1" || !strings.Contains(m.Status, "1 match") {
		t.Fatalf("tag filter = mode=%v filter=%q status=%q", m.TagsFilterMode, m.TagsFilter, m.Status)
	}
	updated, command = m.Update(key("V"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.TagSignatureChecking != "v1" {
		t.Fatalf("tag verify dispatch = cmdnil=%v state=%v checking=%q", command == nil, m.State, m.TagSignatureChecking)
	}
	updated, _ = m.Update(TagSignatureReadyMsg{Generation: 1, Name: "v1", State: tags.SignatureInvalid, Err: errors.New("bad signature")})
	m = updated.(Model)
	if m.TagSignatureChecking != "" || m.TagSnapshot.Tags[1].Signature != tags.SignatureInvalid {
		t.Fatalf("tag verify result = checking=%q tags=%+v", m.TagSignatureChecking, m.TagSnapshot.Tags)
	}
	updated, command = m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending {
		t.Fatalf("tag inspect dispatch = cmdnil=%v state=%v", command == nil, m.State)
	}
	updated, command = m.Update(key("d"))
	m = updated.(Model)
	if command == nil || !m.TagCompareLoading {
		t.Fatalf("tag compare dispatch = cmdnil=%v loading=%v", command == nil, m.TagCompareLoading)
	}
	updated, _ = m.Update(TagCompareReadyMsg{Generation: 1, Name: "v1", Text: "diff --git a/file b/file"})
	m = updated.(Model)
	if m.TagCompareLoading || !strings.Contains(m.TagCompare, "diff --git") {
		t.Fatalf("tag compare result = loading=%v compare=%q", m.TagCompareLoading, m.TagCompare)
	}
	updated, _ = m.Update(key("w"))
	m = updated.(Model)
	if !m.TagWorktreeMode {
		t.Fatalf("tag worktree mode = %v", m.TagWorktreeMode)
	}
	m.TagWorktreePath = "/tmp/tag-worktree"
	updated, command = m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.TagWorktreeMode {
		t.Fatalf("tag worktree dispatch = cmdnil=%v state=%v mode=%v", command == nil, m.State, m.TagWorktreeMode)
	}
	updated, command = m.Update(TagWorktreeFinishedMsg{Generation: 1, Name: "v1", Path: "/tmp/tag-worktree"})
	m = updated.(Model)
	if command == nil || !strings.Contains(m.Status, "created worktree") {
		t.Fatalf("tag worktree result = cmdnil=%v status=%q", command == nil, m.Status)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if !m.TagCheckoutConfirm || !strings.Contains(m.Status, "detached checkout") {
		t.Fatalf("tag checkout confirmation = confirm=%v status=%q", m.TagCheckoutConfirm, m.Status)
	}
	updated, command = m.Update(key("y"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.TagCheckoutConfirm {
		t.Fatalf("tag checkout dispatch = cmdnil=%v state=%v confirm=%v", command == nil, m.State, m.TagCheckoutConfirm)
	}
	updated, command = m.Update(TagCheckoutFinishedMsg{Generation: 1, Name: "v1"})
	m = updated.(Model)
	if command == nil || m.currentView() != workspace.Status || !strings.Contains(m.Status, "checked out tag") {
		t.Fatalf("tag checkout result = cmdnil=%v view=%q status=%q", command == nil, m.currentView(), m.Status)
	}
}

func TestTagMutationControlsRequireExplicitInputsAndRefresh(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Tags, "Tags")
	m.TagSnapshot = tags.Snapshot{Repository: m.Discovery.Root, Tags: []tags.Tag{{Name: "v1", TargetID: "abc123", Kind: tags.Lightweight}}}

	updated, _ := m.Update(key("c"))
	m = updated.(Model)
	if m.TagCreateMode != "name" || m.TagCreateKind != tags.CreateLightweight {
		t.Fatalf("lightweight tag mode = mode=%q kind=%q", m.TagCreateMode, m.TagCreateKind)
	}
	for _, ch := range "v2" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.TagCreateMode != "target" || m.TagCreateInput != "HEAD" {
		t.Fatalf("tag target prompt = mode=%q input=%q", m.TagCreateMode, m.TagCreateInput)
	}
	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.TagCreateMode != "" {
		t.Fatalf("tag create submit = cmdnil=%v state=%v mode=%q", command == nil, m.State, m.TagCreateMode)
	}
	updated, command = m.Update(TagMutationFinishedMsg{Generation: 1, Operation: "created", Name: "v2"})
	m = updated.(Model)
	if command == nil || m.State != StateReady || !strings.Contains(m.Status, "created tag v2") {
		t.Fatalf("tag create completion = cmdnil=%v state=%v status=%q", command == nil, m.State, m.Status)
	}

	m.TagsSelected = 0
	updated, _ = m.Update(key("D"))
	m = updated.(Model)
	if !m.TagDeleteMode || m.TagDeleteTarget != "v1" {
		t.Fatalf("tag delete mode = mode=%v target=%q", m.TagDeleteMode, m.TagDeleteTarget)
	}
	for _, ch := range "v1" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, command = m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.TagDeleteMode {
		t.Fatalf("tag delete submit = cmdnil=%v state=%v mode=%v", command == nil, m.State, m.TagDeleteMode)
	}
}

func TestSubmoduleLifecycleMenuRequiresExactRemovalConfirmation(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.repositoryGeneration = 1
	m.Submodules = submodules.Snapshot{Modules: []submodules.Module{{Path: "nested path", State: submodules.StateUninitialized}}}
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("nested path"), ModeWork: "160000", Submodule: "-.."}})
	m.Workspace.Navigate(workspace.Status, "Status")
	updated, _ := m.Update(key("M"))
	m = updated.(Model)
	if m.SubmoduleAction != "menu" || m.SubmodulePath != "nested path" {
		t.Fatalf("submodule menu = action=%q path=%q", m.SubmoduleAction, m.SubmodulePath)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if m.SubmoduleAction != "confirm-remove" || !strings.Contains(m.Status, "nested path") {
		t.Fatalf("remove confirmation = action=%q status=%q", m.SubmoduleAction, m.Status)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.SubmoduleAction != "remove" {
		t.Fatalf("remove dispatch = cmdnil=%v state=%v action=%q", cmd == nil, m.State, m.SubmoduleAction)
	}
}

func TestBulkSubmodulePreviewAndFailureRetryRouting(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.repositoryGeneration = 1
	m.Submodules = submodules.Snapshot{Modules: []submodules.Module{
		{Path: "first", State: submodules.StateDirty},
		{Path: "second", State: submodules.StateUninitialized},
	}}
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("first"), ModeWork: "160000", Submodule: "-.."}})
	m.Workspace.Navigate(workspace.Status, "Status")

	updated, _ := m.Update(key("M"))
	m = updated.(Model)
	updated, _ = m.Update(key("U"))
	m = updated.(Model)
	if m.SubmoduleAction != "bulk-confirm" || m.BulkSubmoduleAction != string(submodules.BulkUpdate) || len(m.BulkSubmodulePaths) != 2 {
		t.Fatalf("bulk all preview = action=%q bulk=%q paths=%v", m.SubmoduleAction, m.BulkSubmoduleAction, m.BulkSubmodulePaths)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	if m.SubmoduleAction != "" {
		t.Fatalf("bulk preview cancel = action=%q", m.SubmoduleAction)
	}

	updated, _ = m.Update(key("M"))
	m = updated.(Model)
	updated, _ = m.Update(key("2"))
	m = updated.(Model)
	if m.SubmoduleAction != "bulk-confirm" || len(m.BulkSubmodulePaths) != 1 || m.BulkSubmodulePaths[0] != "first" {
		t.Fatalf("bulk selected preview = action=%q paths=%v", m.SubmoduleAction, m.BulkSubmodulePaths)
	}
	updated, command := m.Update(key("y"))
	m = updated.(Model)
	if command == nil || m.SubmoduleAction != "bulk-running" || m.State != StateOperationPending {
		t.Fatalf("bulk dispatch = cmdnil=%v action=%q state=%v", command == nil, m.SubmoduleAction, m.State)
	}

	updated, command = m.Update(BulkSubmoduleFinishedMsg{Generation: 1, Outcome: submodules.BulkOutcome{
		Repository: m.Discovery.Root,
		Action:     submodules.BulkUpdate,
		Items: []submodules.BulkItem{
			{Path: "first", State: submodules.ItemFailed},
			{Path: "second", State: submodules.ItemSucceeded},
		},
	}})
	m = updated.(Model)
	if command == nil || m.SubmoduleAction != "" || !strings.Contains(m.Status, "1 failed") || m.BulkSubmoduleOutcome == nil {
		t.Fatalf("bulk result = cmdnil=%v action=%q status=%q outcome=%v", command == nil, m.SubmoduleAction, m.Status, m.BulkSubmoduleOutcome)
	}
	updated, _ = m.Update(key("M"))
	m = updated.(Model)
	updated, _ = m.Update(key("r"))
	m = updated.(Model)
	if m.SubmoduleAction != "bulk-confirm" || len(m.BulkSubmodulePaths) != 1 || m.BulkSubmodulePaths[0] != "first" {
		t.Fatalf("bulk retry preview = action=%q paths=%v", m.SubmoduleAction, m.BulkSubmodulePaths)
	}
}

func TestInitializedSubmoduleNavigationKeepsParentBreadcrumbAndReturns(t *testing.T) {
	parent := t.TempDir()
	child := t.TempDir()
	m := NewRepository(git.Discovery{Root: parent})
	m.Submodules = submodules.Snapshot{Repository: parent, Modules: []submodules.Module{{Path: "nested", State: submodules.StateClean}}}
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("nested"), ModeWork: "160000", Submodule: "..."}})
	generation := m.repositoryGeneration
	updated, command := m.Update(SubmoduleOpenedMsg{Generation: generation, Path: "nested", Discovery: git.Discovery{Root: child}})
	m = updated.(Model)
	if command == nil || m.Discovery.Root != child || len(m.repositoryParents) != 1 {
		t.Fatalf("submodule open = cmdnil=%v root=%q parents=%d", command == nil, m.Discovery.Root, len(m.repositoryParents))
	}
	_, breadcrumbs, _, _ := m.Workspace.Snapshot()
	if len(breadcrumbs) != 2 || breadcrumbs[1].Label != "Submodule: nested" {
		t.Fatalf("breadcrumbs = %+v", breadcrumbs)
	}
	updated, command = m.Update(key("esc"))
	m = updated.(Model)
	if command == nil || m.Discovery.Root != parent || len(m.repositoryParents) != 0 || m.currentView() != workspace.Status {
		t.Fatalf("parent return = cmdnil=%v root=%q parents=%d view=%q", command == nil, m.Discovery.Root, len(m.repositoryParents), m.currentView())
	}
}

func TestTwoLevelSubmoduleNavigationUsesRealGitDiscoveryAndBoundedParentStack(t *testing.T) {
	ctx := context.Background()
	grandchild := t.TempDir()
	initCommittedTestRepository(t, ctx, grandchild, "grandchild")
	child := t.TempDir()
	initCommittedTestRepository(t, ctx, child, "child")
	childRunner := git.NewRunner(child)
	childRunner.Env = []string{"GIT_ALLOW_PROTOCOL=file"}
	gitMustRunAppTest(t, ctx, childRunner, "submodule", "add", "file://"+grandchild, "grand path")
	gitMustRunAppTest(t, ctx, childRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "add-grandchild")
	parent := t.TempDir()
	initCommittedTestRepository(t, ctx, parent, "parent")
	parentRunner := git.NewRunner(parent)
	parentRunner.Env = []string{"GIT_ALLOW_PROTOCOL=file"}
	gitMustRunAppTest(t, ctx, parentRunner, "submodule", "add", "file://"+child, "child path")
	gitMustRunAppTest(t, ctx, parentRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "add-child")
	childCheckout := filepath.Join(parent, "child path")
	grandchildCheckout := filepath.Join(childCheckout, "grand path")
	childCheckoutRunner := git.NewRunner(childCheckout)
	childCheckoutRunner.Env = []string{"GIT_ALLOW_PROTOCOL=file"}
	gitMustRunAppTest(t, ctx, childCheckoutRunner, "submodule", "update", "--init", "--", "grand path")
	discovery, err := git.Discover(ctx, parent)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepository(discovery)
	parentSnapshot, err := submodules.Load(ctx, parentRunner, submodules.LoadRequest{Repository: parent})
	if err != nil {
		t.Fatal(err)
	}
	m.Submodules = parentSnapshot
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("child path"), ModeWork: "160000", Submodule: "..."}})
	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command == nil {
		t.Fatal("parent submodule open command was nil")
	}
	updated, _ = m.Update(command())
	m = updated.(Model)
	if !sameTestPath(m.Discovery.Root, childCheckout) || len(m.repositoryParents) != 1 {
		t.Fatalf("child navigation = root=%q parents=%d", m.Discovery.Root, len(m.repositoryParents))
	}
	childSnapshot, err := submodules.Load(ctx, git.NewRunner(childCheckout), submodules.LoadRequest{Repository: childCheckout})
	if err != nil {
		t.Fatal(err)
	}
	m.Submodules = childSnapshot
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("grand path"), ModeWork: "160000", Submodule: "..."}})
	updated, command = m.Update(key("enter"))
	m = updated.(Model)
	if command == nil {
		t.Fatalf("grandchild submodule open command was nil: modules=%+v selected=%q status=%q", m.Submodules.Modules, m.Files.SelectedPath(), m.Status)
	}
	updated, _ = m.Update(command())
	m = updated.(Model)
	if !sameTestPath(m.Discovery.Root, grandchildCheckout) || len(m.repositoryParents) != 2 {
		t.Fatalf("grandchild navigation = root=%q parents=%d", m.Discovery.Root, len(m.repositoryParents))
	}
	updated, command = m.Update(key("esc"))
	m = updated.(Model)
	if command == nil || !sameTestPath(m.Discovery.Root, childCheckout) || len(m.repositoryParents) != 1 {
		t.Fatalf("child return = cmdnil=%v root=%q parents=%d", command == nil, m.Discovery.Root, len(m.repositoryParents))
	}
	updated, command = m.Update(key("esc"))
	m = updated.(Model)
	if command == nil || !sameTestPath(m.Discovery.Root, parent) || len(m.repositoryParents) != 0 {
		t.Fatalf("parent return = cmdnil=%v root=%q parents=%d", command == nil, m.Discovery.Root, len(m.repositoryParents))
	}
}

func initCommittedTestRepository(t *testing.T, ctx context.Context, root, name string) {
	t.Helper()
	runner := git.NewRunner(root)
	for _, args := range [][]string{{"init", "-b", "main", "--", root}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.test"}, {"config", "commit.gpgsign", "false"}} {
		gitMustRunAppTest(t, ctx, runner, args...)
	}
	if err := os.WriteFile(filepath.Join(root, "README"), []byte(name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", name)
}

func gitMustRunAppTest(t *testing.T, ctx context.Context, runner git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(ctx, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

type appRoundTripFunc func(*http.Request) (*http.Response, error)

func (f appRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func sameTestPath(left, right string) bool {
	leftResolved, leftErr := filepath.EvalSymlinks(left)
	rightResolved, rightErr := filepath.EvalSymlinks(right)
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return filepath.Clean(leftResolved) == filepath.Clean(rightResolved)
}

func TestCherryPickProgressCanNavigateToStatusAndBack(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	state, err := sequencer.NewState("repo", 1, sequencer.KindCherryPick, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithObservation("head", "current", 0, 0, nil, time.Now())
	state, err = state.WithDetails(sequencer.Details{CherryPick: &sequencer.CherryPickDetails{Commits: []string{"current"}, CurrentIndex: 0}})
	if err != nil {
		t.Fatal(err)
	}
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "feature"}, Operation: &state})
	updated, cmd := m.Update(key("C"))
	m = updated.(Model)
	if cmd != nil || m.currentView() != workspace.CherryPick {
		t.Fatalf("status recovery route = view=%q cmdnil=%v", m.currentView(), cmd != nil)
	}
	if cmd := m.executePaletteAction("cherry_pick_recovery"); cmd != nil || m.currentView() != workspace.CherryPick {
		t.Fatalf("initial progress route = view=%q cmdnil=%v", m.currentView(), cmd != nil)
	}
	updated, cmd = m.Update(key("1"))
	m = updated.(Model)
	if cmd != nil || m.currentView() != workspace.Status {
		t.Fatalf("status navigation = view=%q cmdnil=%v", m.currentView(), cmd != nil)
	}
	if cmd := m.executePaletteAction("cherry_pick_recovery"); cmd != nil || m.currentView() != workspace.CherryPick {
		t.Fatalf("progress reopen = view=%q cmdnil=%v", m.currentView(), cmd != nil)
	}
}

func TestExternalCherryPickResolutionEnablesContinueFromFreshSnapshot(t *testing.T) {
	m := New()
	m.Width, m.Height = 80, 24
	m.Discovery.Root = t.TempDir()
	m.Workspace.Navigate(workspace.CherryPick, "Cherry-pick progress")
	state, err := sequencer.NewState("repo", 1, sequencer.KindCherryPick, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	state = state.WithObservation("before", "current", 0, 1, []string{"file"}, time.Now())
	state, err = state.WithDetails(sequencer.Details{CherryPick: &sequencer.CherryPickDetails{Commits: []string{"current"}, CurrentIndex: 0}})
	if err != nil {
		t.Fatal(err)
	}
	m.applySnapshot(repo.Snapshot{
		Root:       m.Discovery.Root,
		Branch:     repo.Branch{Name: "feature"},
		Operation:  &state,
		Conflicts:  []conflicts.Conflict{{Path: []byte("file"), Resolution: "unmerged"}},
		Counts:     repo.Counts{Conflicted: 1, Staged: 1},
		Generation: 1,
	})
	if m.Conflict.Target != "feature" {
		t.Fatalf("cherry-pick target branch = %q", m.Conflict.Target)
	}
	if actions := m.Conflict.RecoveryActions(); actions.Continue {
		t.Fatalf("unresolved cherry-pick exposed continue: %+v", actions)
	}
	footer := strings.Split(m.View().Content, "\n")
	if !strings.Contains(footer[len(footer)-1], "[s] skip") || strings.Contains(footer[len(footer)-1], "[c] continue") || len(footer[len(footer)-1]) > 80 {
		t.Fatalf("unresolved narrow footer = %q", footer[len(footer)-1])
	}
	m.Status = "cherry-pick paused for conflict recovery"
	m.Toast.Text = "repository conflicts"
	withNotices := strings.Split(m.View().Content, "\n")
	if len(withNotices) > 24 || !strings.Contains(withNotices[len(withNotices)-1], "[s] skip") {
		t.Fatalf("narrow recovery clipped its footer: %d lines, last=%q", len(withNotices), withNotices[len(withNotices)-1])
	}
	m.Status, m.Toast.Text = "", ""

	// This is the authoritative refresh after an external editor resolved and
	// staged the conflict while the progress workspace stayed open.
	m.applySnapshot(repo.Snapshot{
		Root:       m.Discovery.Root,
		Branch:     repo.Branch{Name: "feature"},
		Operation:  &state,
		Counts:     repo.Counts{Staged: 1},
		Generation: 2,
	})
	if actions := m.Conflict.RecoveryActions(); !actions.Continue {
		t.Fatalf("external resolution did not enable continue: %+v", actions)
	}
	if view := m.Conflict.View(80, 24); !strings.Contains(view, "[c] continue") {
		t.Fatalf("resolved progress footer omitted continue:\n%s", view)
	}
	footer = strings.Split(m.View().Content, "\n")
	if !strings.Contains(footer[len(footer)-1], "[c] continue") || !strings.Contains(footer[len(footer)-1], "[s] skip") || len(footer[len(footer)-1]) > 80 {
		t.Fatalf("resolved narrow footer = %q", footer[len(footer)-1])
	}
	updated, command := m.Update(key("c"))
	if command == nil || updated.(Model).State != StateOperationPending {
		t.Fatalf("continue input after external resolution = state=%v cmdnil=%v", updated.(Model).State, command == nil)
	}
}

func TestActiveSequencerRecoveryPaletteRoutesAllOperationKinds(t *testing.T) {
	for _, test := range []struct {
		kind sequencer.Kind
		view workspace.View
	}{
		{kind: sequencer.KindRebase, view: workspace.Conflict},
		{kind: sequencer.KindCherryPick, view: workspace.CherryPick},
		{kind: sequencer.KindRevert, view: workspace.Conflict},
		{kind: sequencer.KindMerge, view: workspace.Conflict},
	} {
		t.Run(test.kind.String(), func(t *testing.T) {
			m := New()
			m.Discovery.Root = t.TempDir()
			state, err := sequencer.NewState("repo", 1, test.kind, sequencer.PhasePaused)
			if err != nil {
				t.Fatal(err)
			}
			m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}, Operation: &state})
			if command := m.executePaletteAction("operation_recovery"); command != nil || m.currentView() != test.view {
				t.Fatalf("recovery route = command=%v view=%q want=%q", command != nil, m.currentView(), test.view)
			}
		})
	}
}

func TestExternalSequencerCompletionClosesConflictWorkspace(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	state, err := sequencer.NewState("repo", 1, sequencer.KindRevert, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}, Operation: &state})
	m.executePaletteAction("cherry_pick_recovery")
	// Simulate a fresh authoritative refresh after another terminal aborts or
	// completes the operation. The operation marker is gone and no conflicts
	// remain, so the recovery workspace must close safely.
	m.applySnapshot(repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}})
	if m.currentView() != workspace.Status || !strings.Contains(m.Status, "externally") {
		t.Fatalf("external completion route = view=%q status=%q", m.currentView(), m.Status)
	}
}

func TestBranchMergePromptRequiresCleanWorktreeAndExplicitStrategy(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Snapshot = repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}}
	m.Branches = branchview.New([]branches.Branch{{Name: "feature"}})
	m.Workspace.Navigate(workspace.Branches, "Branches")
	updated, cmd := m.Update(key("M"))
	m = updated.(Model)
	if cmd != nil || !m.BranchMergeMode || m.BranchMergeTarget != "feature" {
		t.Fatalf("merge prompt = mode=%v target=%q cmdnil=%v", m.BranchMergeMode, m.BranchMergeTarget, cmd != nil)
	}
	for _, value := range []string{"x", "enter"} {
		updated, cmd = m.Update(key(value))
		m = updated.(Model)
	}
	if cmd != nil || !m.BranchMergeMode || !strings.Contains(m.Status, "merge strategy must be") {
		t.Fatalf("invalid strategy = mode=%v status=%q cmdnil=%v", m.BranchMergeMode, m.Status, cmd != nil)
	}
	updated, cmd = m.Update(key("backspace"))
	m = updated.(Model)
	for _, value := range []string{"f", "f", "-", "o", "n", "l", "y", "enter"} {
		updated, cmd = m.Update(key(value))
		m = updated.(Model)
	}
	if cmd == nil || m.BranchMergeMode {
		t.Fatalf("valid strategy = mode=%v cmdnil=%v status=%q", m.BranchMergeMode, cmd == nil, m.Status)
	}

	m = New()
	m.Discovery.Root = t.TempDir()
	m.Snapshot = repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}, Counts: repo.Counts{Unstaged: 1}}
	m.Branches = branchview.New([]branches.Branch{{Name: "feature"}})
	m.Workspace.Navigate(workspace.Branches, "Branches")
	updated, cmd = m.Update(key("M"))
	m = updated.(Model)
	if cmd != nil || m.BranchMergeMode || !strings.Contains(m.Status, "clean worktree") {
		t.Fatalf("dirty merge guard = mode=%v status=%q cmdnil=%v", m.BranchMergeMode, m.Status, cmd != nil)
	}

	m = New()
	m.Discovery.Root = t.TempDir()
	m.Snapshot = repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}}
	m.Branches = branchview.New([]branches.Branch{{Name: "feature", OccupiedPath: "/tmp/linked-feature"}})
	m.Workspace.Navigate(workspace.Branches, "Branches")
	updated, cmd = m.Update(key("M"))
	m = updated.(Model)
	if cmd != nil || m.BranchMergeMode || !strings.Contains(m.Status, "another worktree") {
		t.Fatalf("occupied merge guard = mode=%v status=%q cmdnil=%v", m.BranchMergeMode, m.Status, cmd != nil)
	}
}

func TestRemoteBranchMergeRetainsQualifiedTarget(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Snapshot = repo.Snapshot{Root: m.Discovery.Root, Branch: repo.Branch{Name: "main"}}
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/feature", Remote: true, RemoteName: "origin", RemoteBranch: "feature"}})
	m.Workspace.Navigate(workspace.Branches, "Branches")

	updated, cmd := m.Update(key("M"))
	m = updated.(Model)
	if cmd != nil || !m.BranchMergeMode || m.BranchMergeTarget != "origin/feature" {
		t.Fatalf("remote merge target = cmdnil=%v mode=%v target=%q", cmd == nil, m.BranchMergeMode, m.BranchMergeTarget)
	}
}

func TestBranchFastForwardRequiresExplicitConfirmation(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "main", Current: true, Upstream: "origin/main", Behind: 2}})

	updated, cmd := m.Update(key("F"))
	m = updated.(Model)
	if cmd != nil || !m.BranchRecoveryConfirm || m.BranchRecoveryTarget != "origin/main" {
		t.Fatalf("fast-forward prompt = cmdnil=%v confirm=%v target=%q", cmd == nil, m.BranchRecoveryConfirm, m.BranchRecoveryTarget)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.BranchRecoveryConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("fast-forward cancellation = cmdnil=%v confirm=%v status=%q", cmd == nil, m.BranchRecoveryConfirm, m.Status)
	}
}

func TestBranchResetPromptOnlyAcceptsSoftOrMixedRef(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "main", Current: true}})
	updated, cmd := m.Update(key("z"))
	m = updated.(Model)
	if cmd != nil || !m.BranchResetPrompt {
		t.Fatalf("reset prompt = cmdnil=%v prompt=%v", cmd == nil, m.BranchResetPrompt)
	}
	for _, ch := range "soft HEAD~1" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.BranchResetPrompt || m.State != StateOperationPending {
		t.Fatalf("reset submit = cmdnil=%v prompt=%v state=%v", cmd == nil, m.BranchResetPrompt, m.State)
	}
}

func TestSelectedBranchRebaseUsesQualifiedBaseAndShowsPreview(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Snapshot = repo.Snapshot{Branch: repo.Branch{Name: "main", Ahead: 3, Behind: 2}}
	m.HistoryCommits = []history.Commit{{SHA: "abc", Subject: "change"}}
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true, RemoteName: "origin", RemoteBranch: "main"}})

	updated, cmd := m.Update(key("I"))
	m = updated.(Model)
	if cmd != nil || m.currentView() != workspace.Rebase || m.Rebase.Base.Ref != "origin/main" {
		t.Fatalf("branch rebase = cmdnil=%v view=%q base=%q", cmd == nil, m.currentView(), m.Rebase.Base.Ref)
	}
	view := m.View().Content
	if !contains(view, "Divergence: 3 ahead") || !contains(view, "Commits to rewrite: 3") {
		t.Fatalf("rebase preview missing: %s", view)
	}
}

func TestCompareAssignsSelectedRevisionsFromDifferentWorkspaces(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{{SHA: "commit-a", Subject: "A"}})
	updated, cmd := m.Update(key("Y"))
	m = updated.(Model)
	if cmd != nil || m.CompareLeft != "commit-a" || !strings.Contains(m.Status, "select B") {
		t.Fatalf("comparison A assignment = cmdnil=%v left=%q status=%q", cmd == nil, m.CompareLeft, m.Status)
	}
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true, RemoteName: "origin", RemoteBranch: "main"}})
	updated, cmd = m.Update(key("Y"))
	m = updated.(Model)
	if cmd == nil || m.CompareRight != "origin/main" || m.currentView() != workspace.Compare || !m.CompareLoading {
		t.Fatalf("comparison B assignment = cmdnil=%v right=%q view=%q loading=%v", cmd == nil, m.CompareRight, m.currentView(), m.CompareLoading)
	}
}

func TestCompareAssignsTagsAndReflogEntries(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Tags, "Tags")
	m.TagSnapshot = tags.Snapshot{Tags: []tags.Tag{{Name: "v1.2.3"}}}
	m.TagsSelected = 0
	updated, cmd := m.Update(key("Y"))
	m = updated.(Model)
	if cmd != nil || m.CompareLeft != "v1.2.3" {
		t.Fatalf("tag comparison assignment = cmdnil=%v left=%q", cmd == nil, m.CompareLeft)
	}
	m.Workspace.Navigate(workspace.Reflog, "Reflog")
	m.Reflog.SetPage([]reflog.Entry{{SHA: "reflog-sha"}}, false)
	updated, cmd = m.Update(key("Y"))
	m = updated.(Model)
	if cmd == nil || m.CompareRight != "reflog-sha" || m.currentView() != workspace.Compare {
		t.Fatalf("reflog comparison assignment = cmdnil=%v right=%q view=%q", cmd == nil, m.CompareRight, m.currentView())
	}
}

func TestCompareAssignsCurrentBranchAndSelectedRemote(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Snapshot = repo.Snapshot{Branch: repo.Branch{Name: "main"}}
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, cmd := m.Update(key("Y"))
	m = updated.(Model)
	if cmd != nil || m.CompareLeft != "main" {
		t.Fatalf("remote comparison local assignment = cmdnil=%v left=%q", cmd == nil, m.CompareLeft)
	}
	updated, cmd = m.Update(key("Y"))
	m = updated.(Model)
	if cmd == nil || m.CompareRight != "origin/main" || m.currentView() != workspace.Compare {
		t.Fatalf("remote comparison assignment = cmdnil=%v right=%q view=%q", cmd == nil, m.CompareRight, m.currentView())
	}
}

func TestComparisonRoutesRemoteActionsThroughRemoteEngine(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Compare, "Comparison")
	m.CompareLeft, m.CompareRight = "main", "origin/main"
	m.Snapshot = repo.Snapshot{Branch: repo.Branch{Name: "main"}}
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	for _, keyValue := range []string{"f", "o", "m", "e", "p"} {
		updated, cmd := m.Update(key(keyValue))
		m = updated.(Model)
		if cmd == nil || m.Remotes.Selected != 0 {
			t.Fatalf("comparison action %q = cmdnil=%v remote=%d status=%q", keyValue, cmd == nil, m.Remotes.Selected, m.Status)
		}
		m.State = StateReady
	}
}

func TestMergeStrategyNames(t *testing.T) {
	for _, test := range []struct {
		input string
		want  mergeops.Strategy
	}{
		{"merge", mergeops.Regular},
		{"ff-only", mergeops.FastForwardOnly},
		{"no-ff", mergeops.NoFastForward},
		{"squash", mergeops.Squash},
	} {
		got, ok := mergeStrategy(test.input)
		if !ok || got != test.want {
			t.Fatalf("mergeStrategy(%q) = %v, %v", test.input, got, ok)
		}
	}
	if _, ok := mergeStrategy("unsafe"); ok {
		t.Fatal("unsafe merge strategy accepted")
	}
}

func TestBranchMergeRunsThroughOperationEngineAndRefreshes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("base.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("feature.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("feature\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepositoryWithConfig(discovery, config.Defaults())
	m.repositoryGeneration = 7
	m.Snapshot = repo.Snapshot{Root: dir, Branch: repo.Branch{Name: "main"}}
	m.Branches = branchview.New([]branches.Branch{{Name: "feature"}})
	m.Workspace.Navigate(workspace.Branches, "Branches")
	updated, cmd := m.Update(key("M"))
	m = updated.(Model)
	if cmd != nil || !m.BranchMergeMode {
		t.Fatalf("merge prompt = mode=%v cmdnil=%v", m.BranchMergeMode, cmd != nil)
	}
	for _, value := range []string{"f", "f", "-", "o", "n", "l", "y", "enter"} {
		updated, cmd = m.Update(key(value))
		m = updated.(Model)
	}
	if cmd == nil {
		t.Fatal("merge command was not created")
	}
	message := cmd()
	updated, _ = m.Update(message)
	m = updated.(Model)
	if m.State != StateReady || m.Status != "merge completed" || m.BranchMergeMode {
		t.Fatalf("merge completion = state=%v status=%q mode=%v", m.State, m.Status, m.BranchMergeMode)
	}
	if len(m.OperationEngine.Snapshot()) == 0 {
		t.Fatal("operation engine did not retain merge lifecycle result")
	}
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Fatalf("merged file missing: %v", err)
	}
}

func TestBranchSquashMergeExplainsStagedChangesAndNoMergeCommit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "add", "--", "base.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "commit", "-m", "base"); err != nil {
		t.Fatal(err)
	}
	baseHeadResult, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	baseHead := strings.TrimSpace(string(baseHeadResult.Stdout))
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "add", "--", "feature.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "commit", "-m", "feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepositoryWithConfig(discovery, config.Defaults())
	t.Cleanup(func() { _ = m.Close() })
	m.repositoryGeneration = 3
	m.Snapshot = repo.Snapshot{Root: dir, Branch: repo.Branch{Name: "main", OID: baseHead}}
	m.BranchMergeTarget = "feature"

	command := m.mergeSelectedBranch(mergeops.Squash)
	if command == nil {
		t.Fatal("squash merge command was not created")
	}
	finished := command().(MergeFinishedMsg)
	if finished.Strategy != mergeops.Squash || finished.Outcome.Err != nil || finished.Outcome.Snapshot == nil || finished.Outcome.Snapshot.Counts.Staged == 0 {
		t.Fatalf("squash outcome = %+v", finished)
	}
	updated, _ := m.Update(finished)
	m = updated.(Model)
	if m.State != StateReady || !strings.Contains(m.Status, "no merge commit created") || !strings.Contains(m.Status, "staged") {
		t.Fatalf("squash completion did not explain resulting state: %s", m.Status)
	}
	if m.Snapshot.Counts.Staged == 0 || m.Snapshot.Operation != nil {
		t.Fatalf("squash authoritative snapshot = %+v", m.Snapshot)
	}
	newHead, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(newHead.Stdout)) != baseHead {
		t.Fatalf("squash unexpectedly created a merge commit: HEAD=%q err=%v", newHead.Stdout, err)
	}
}

func TestCherryPickSelectionRunsThroughEngineAndJournalsCompletion(t *testing.T) {
	root := t.TempDir()
	runner := git.NewRunner(root)
	runner.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null"}
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "cherry-pick-test"},
		{"config", "user.email", "cherry-pick@example.invalid"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "--", "base.txt"}, {"commit", "-m", "base"}, {"switch", "-c", "feature"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "picked.txt"), []byte("picked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "add", "--", "picked.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "commit", "-m", "picked"); err != nil {
		t.Fatal(err)
	}
	featureSHAResult, err := runner.Run(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	featureSHA := strings.TrimSpace(string(featureSHAResult.Stdout))
	if _, err := runner.Run(context.Background(), "switch", "main"); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepository(discovery)
	t.Cleanup(func() { _ = m.Close() })
	m.repositoryGeneration = 1
	m.Snapshot = repo.Snapshot{Root: root, Branch: repo.Branch{Name: "main"}}
	head, err := runner.Run(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	m.Snapshot.Branch.OID = strings.TrimSpace(string(head.Stdout))
	m.HistoryCommits = []history.Commit{{SHA: featureSHA, Subject: "picked", Parents: []string{"base"}}}
	m.History = historyview.New(m.HistoryCommits)
	m.Workspace.Navigate(workspace.Log, "History")
	updated, cmd := m.Update(key("P"))
	m = updated.(Model)
	if cmd != nil || !m.CherryPickConfirm {
		t.Fatalf("cherry-pick confirmation = cmdnil=%v confirm=%v status=%q", cmd != nil, m.CherryPickConfirm, m.Status)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.CherryPickConfirm {
		t.Fatalf("cherry-pick start = cmdnil=%v confirm=%v", cmd == nil, m.CherryPickConfirm)
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if m.State != StateReady || m.Status != "cherry-pick completed" {
		t.Fatalf("cherry-pick completion = state=%v status=%q", m.State, m.Status)
	}
	var found bool
	for _, event := range m.ActivityLog.All() {
		if event.Operation != nil && event.Operation.Kind == "cherry-pick" {
			found = true
			if event.Operation.OldHead == "" || event.Operation.NewHead == "" || len(event.Operation.Args) != 2 {
				t.Fatalf("cherry-pick journal = %#v", event.Operation)
			}
		}
	}
	if !found {
		t.Fatal("cherry-pick completion was not journaled")
	}
	if _, err := os.Stat(filepath.Join(root, "picked.txt")); err != nil {
		t.Fatalf("cherry-picked file missing: %v", err)
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

func TestRemoteBranchWorktreeRouting(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/feature", Remote: true, RemoteName: "origin", RemoteBranch: "feature"}})

	updated, cmd := m.Update(key("w"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Worktrees || !m.WorktreeAddMode || m.WorktreeAddCommit != "origin/feature" {
		t.Fatalf("remote worktree routing = cmdnil=%v view=%q mode=%v commit=%q", cmd == nil, m.currentView(), m.WorktreeAddMode, m.WorktreeAddCommit)
	}
	if !contains(m.Status, "origin/feature") {
		t.Fatalf("remote worktree status = %q", m.Status)
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

func TestReflogPaletteRouteStartsBoundedLoad(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	cmd := m.executePaletteAction("reflog")
	if cmd == nil || m.currentView() != workspace.Reflog {
		t.Fatalf("reflog palette route = cmdnil=%v view=%q", cmd == nil, m.currentView())
	}
}

func TestReflogRecoveryPointActionsUseExistingSafeFlows(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Reflog, "Reflog")
	m.Reflog.SetPage([]reflog.Entry{{SHA: "abcdef1234567890", Actor: "actor", Subject: "commit: restore"}}, false)
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("reflog inspection = cmdnil=%v state=%v", cmd == nil, m.State)
	}
	m = New()
	m.Workspace.Navigate(workspace.Reflog, "Reflog")
	m.Reflog.SetPage([]reflog.Entry{{SHA: "abcdef1234567890"}}, false)
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if !m.HistoryActionConfirm || m.HistoryActionTarget != "abcdef1234567890" {
		t.Fatalf("reflog checkout confirmation = confirm=%v target=%q", m.HistoryActionConfirm, m.HistoryActionTarget)
	}
	m = New()
	m.Workspace.Navigate(workspace.Reflog, "Reflog")
	m.Reflog.SetPage([]reflog.Entry{{SHA: "abcdef1234567890"}}, false)
	updated, _ = m.Update(key("B"))
	m = updated.(Model)
	if !m.HistoryBranchCreating || m.HistoryBranchTarget != "abcdef1234567890" {
		t.Fatalf("reflog branch flow = creating=%v target=%q", m.HistoryBranchCreating, m.HistoryBranchTarget)
	}
	m = New()
	m.Discovery.Root = t.TempDir()
	m.Workspace.Navigate(workspace.Reflog, "Reflog")
	m.Reflog.SetPage([]reflog.Entry{{SHA: "abcdef1234567890"}}, false)
	updated, cmd = m.Update(key("d"))
	m = updated.(Model)
	if cmd == nil || !m.ReflogCompareLoading || !strings.Contains(m.Status, "comparing") {
		t.Fatalf("reflog compare flow = cmdnil=%v loading=%v status=%q", cmd == nil, m.ReflogCompareLoading, m.Status)
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

func TestConflictNotificationIsEmittedOncePerOperationTransition(t *testing.T) {
	m := New()
	snapshot := repo.Snapshot{Counts: repo.Counts{Conflicted: 2}}
	updated, _ := m.Update(SnapshotMsg{Snapshot: snapshot})
	m = updated.(Model)
	updated, _ = m.Update(SnapshotMsg{Snapshot: snapshot})
	m = updated.(Model)
	if got := len(m.Notifications.Items()); got != 1 {
		t.Fatalf("repeated conflict refresh notifications = %d", got)
	}
	state, err := sequencer.NewState("repo", 1, sequencer.KindRevert, sequencer.PhasePaused)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Operation = &state
	updated, _ = m.Update(SnapshotMsg{Snapshot: snapshot})
	m = updated.(Model)
	if got := len(m.Notifications.Items()); got != 2 {
		t.Fatalf("operation transition notifications = %d", got)
	}
}

func TestStaleSnapshotDoesNotCrossRepositoryGeneration(t *testing.T) {
	m := New()
	m.repositoryGeneration = 2
	m.Snapshot = repo.Snapshot{Root: "/current", Branch: repo.Branch{Name: "current"}}
	updated, cmd := m.Update(SnapshotMsg{
		Generation: 1,
		Snapshot:   repo.Snapshot{Root: "/old", Branch: repo.Branch{Name: "old"}},
	})
	got := updated.(Model)
	if cmd != nil || got.Snapshot.Root != "/current" || got.Snapshot.Branch.Name != "current" {
		t.Fatalf("stale snapshot applied = cmdnil=%v root=%q branch=%q", cmd != nil, got.Snapshot.Root, got.Snapshot.Branch.Name)
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
	if len(wide) != m.Height || !strings.Contains(wide[5], "│ Diff (unstaged) · notes.txt") {
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
	updated, command := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 10, Y: 6})
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

func TestOperationActivityRendersSemanticJournalDetails(t *testing.T) {
	m := NewRepository(git.Discovery{Root: "/repo-a"})
	m.Width, m.Height = 180, 24
	m.Snapshot.Branch.OID = "old-head"
	m.recordActivityWithOperation(history.OperationSuccess, "feature", "merge completed", &history.OperationRecord{
		Repository: "/repo-a", Kind: "merge", Target: "feature", OldHead: "old-head", NewHead: "new-head",
		RecoverySHA: "new-head", Duration: 1250 * time.Millisecond,
	})
	view := m.View().Content
	for _, expected := range []string{"merge", "HEAD old-head -> new-head", "recovery new-head", "1.25s"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("semantic activity missing %q: %q", expected, view)
		}
	}
}

func TestSelectedProfileOverridesAutoFetchPolicy(t *testing.T) {
	c := config.Defaults()
	c.Profile = "work"
	c.Remote.AutoFetchProfiles = map[string]config.AutoFetchProfile{
		"work": {Enabled: true, Interval: 11 * time.Minute, Jitter: 2 * time.Second},
	}
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, c)
	if !m.AutoFetchEnabled || m.AutoFetchScheduler == nil {
		t.Fatalf("selected auto-fetch profile disabled: enabled=%v scheduler=%v", m.AutoFetchEnabled, m.AutoFetchScheduler != nil)
	}
	policy := m.AutoFetchScheduler.Config()
	if policy.Interval != 11*time.Minute || policy.Jitter != 2*time.Second {
		t.Fatalf("selected auto-fetch profile = %#v", policy)
	}
}

func TestOperationJournalWorkspaceIsBoundedAndNavigable(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Width, m.Height = 160, 24
	for index := 0; index < 3; index++ {
		m.recordActivityWithOperation(history.OperationSuccess, fmt.Sprintf("target-%d", index), "completed", &history.OperationRecord{
			Repository: m.Discovery.Root, Kind: "commit", Target: fmt.Sprintf("target-%d", index), Outcome: "success",
		})
	}
	if cmd := m.executePaletteAction("journal"); cmd != nil || m.currentView() != workspace.Journal {
		t.Fatalf("journal route = cmdnil=%v view=%q", cmd != nil, m.currentView())
	}
	if !strings.Contains(m.View().Content, "operation journal") || !strings.Contains(m.View().Content, "target-2") {
		t.Fatalf("journal view missing latest event: %q", m.View().Content)
	}
	updated, _ := m.Update(key("j"))
	m = updated.(Model)
	if m.JournalOffset != 1 || !strings.Contains(m.View().Content, "target-1") {
		t.Fatalf("journal navigation = offset=%d view=%q", m.JournalOffset, m.View().Content)
	}
	updated, _ = m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 4, Y: 6})
	m = updated.(Model)
	if m.JournalOffset != 2 {
		t.Fatalf("journal mouse navigation = offset=%d", m.JournalOffset)
	}
}

func TestOperationJournalCanFilterByRepository(t *testing.T) {
	m := NewRepository(git.Discovery{Root: "/repo-a"})
	m.Width, m.Height = 160, 24
	m.recordActivityWithOperation(history.OperationSuccess, "a-target", "completed", &history.OperationRecord{
		Repository: "/repo-a", Kind: "fetch", Outcome: "success",
	})
	m.recordActivityWithOperation(history.OperationFailure, "b-target", "failed", &history.OperationRecord{
		Repository: "/repo-b", Kind: "pull", Outcome: "failure",
	})
	if cmd := m.executePaletteAction("journal"); cmd != nil {
		t.Fatal("journal navigation unexpectedly returned a command")
	}
	updated, _ := m.Update(key("/"))
	m = updated.(Model)
	for _, char := range []string{"r", "e", "p", "o", "-", "b", "enter"} {
		updated, _ = m.Update(key(char))
		m = updated.(Model)
	}
	view := m.View().Content
	if !strings.Contains(view, "filter: repo-b") || !strings.Contains(view, "b-target") || strings.Contains(view, "a-target") {
		t.Fatalf("journal repository filter = %q", view)
	}
}

func TestOperationJournalSupportsTypedFiltersAndDetails(t *testing.T) {
	m := NewRepository(git.Discovery{Root: "/repo-a"})
	m.Width, m.Height = 180, 24
	m.recordActivityWithOperation(history.OperationSuccess, "merge-target", "completed", &history.OperationRecord{
		Repository: "/repo-a", Kind: "merge", Outcome: "success", Args: []string{"merge", "--token", "secret"},
		OldHead: "old", NewHead: "new", Refs: []string{"refs/heads/main"},
	})
	m.recordActivityWithOperation(history.OperationFailure, "fetch-target", "failed", &history.OperationRecord{
		Repository: "/repo-a", Kind: "fetch", Outcome: "failure",
	})
	if cmd := m.executePaletteAction("journal"); cmd != nil {
		t.Fatal("journal navigation unexpectedly returned a command")
	}
	updated, _ := m.Update(key("/"))
	m = updated.(Model)
	for _, char := range []string{"t", "y", "p", "e", ":", "m", "e", "r", "g", "e", " ", "o", "u", "t", "c", "o", "m", "e", ":", "s", "u", "c", "c", "e", "s", "s", "enter"} {
		updated, _ = m.Update(key(char))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	view := m.View().Content
	if !strings.Contains(view, "merge-target") || strings.Contains(view, "fetch-target") || !strings.Contains(view, "selected operation details") || strings.Contains(view, "secret") || !strings.Contains(view, "<redacted>") {
		t.Fatalf("typed journal filter/details = %q", view)
	}
}

func TestOperationJournalShowsAndCancelsRunningOperation(t *testing.T) {
	m := NewRepository(git.Discovery{Root: "/repo-a"})
	m.OperationEngine = operations.New(1)
	if err := m.OperationEngine.Submit(context.Background(), "journal-running", "/repo-a", "long fetch", time.Minute, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	if cmd := m.executePaletteAction("journal"); cmd != nil {
		t.Fatal("journal navigation unexpectedly returned a command")
	}
	if !strings.Contains(m.View().Content, "running operations:") {
		t.Fatalf("journal omitted running operation: %q", m.View().Content)
	}
	updated, _ := m.Update(key("K"))
	m = updated.(Model)
	if !m.JournalCancelConfirm {
		t.Fatalf("cancel confirmation not opened: status=%q", m.Status)
	}
	updated, _ = m.Update(key("y"))
	m = updated.(Model)
	if !strings.Contains(m.Status, "cancellation requested") {
		t.Fatalf("cancel status = %q", m.Status)
	}
	select {
	case result := <-m.OperationEngine.Results():
		if result.State != operations.Cancelled {
			t.Fatalf("cancel result = %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled operation did not finish")
	}
}

func TestOperationJournalRetriesReplayableFetch(t *testing.T) {
	m := NewRepository(git.Discovery{Root: "/repo-a"})
	m.OperationEngine = operations.New(1)
	var attempts atomic.Int32
	if err := m.OperationEngine.SubmitWithOptions(context.Background(), "journal-fetch", "/repo-a", "fetch", time.Minute, func(context.Context) error {
		if attempts.Add(1) == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}, operations.Options{Retryable: true}); err != nil {
		t.Fatal(err)
	}
	<-m.OperationEngine.Results()
	if cmd := m.executePaletteAction("journal"); cmd != nil {
		t.Fatal("journal navigation unexpectedly returned a command")
	}
	updated, _ := m.Update(key("Y"))
	m = updated.(Model)
	if !m.JournalRetryConfirm {
		t.Fatalf("retry confirmation not opened: status=%q", m.Status)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("retry confirmation did not return a command")
	}
	message := cmd()
	updated, _ = m.Update(message)
	m = updated.(Model)
	if attempts.Load() != 2 || !strings.Contains(m.Status, "retry complete") {
		t.Fatalf("retry status=%q attempts=%d", m.Status, attempts.Load())
	}
}

func TestJournalRedoRefusalOpensGuidedReflogRecovery(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.recordActivityWithOperation(history.OperationSuccess, "feature", "merge completed", &history.OperationRecord{
		Repository: m.Discovery.Root, Kind: "merge", Target: "feature", Outcome: "success",
	})
	if cmd := m.executePaletteAction("journal"); cmd != nil {
		t.Fatal("journal navigation unexpectedly returned a command")
	}
	updated, cmd := m.Update(key("R"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Reflog {
		t.Fatalf("guided recovery route = cmdnil=%v view=%q status=%q", cmd == nil, m.currentView(), m.Status)
	}
	if !strings.Contains(m.Status, "guided recovery") {
		t.Fatalf("guided recovery status = %q", m.Status)
	}
}

func TestBisectWorkspaceCollectsExplicitBadAndGoodRefs(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	updated, _ := m.Update(key("S"))
	m = updated.(Model)
	if m.BisectStartMode != "bad" {
		t.Fatalf("bisect start mode = %q", m.BisectStartMode)
	}
	for _, ch := range "bad-ref" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.BisectStartMode != "good" || m.BisectStartBad != "bad-ref" {
		t.Fatalf("bad ref selection = mode=%q bad=%q", m.BisectStartMode, m.BisectStartBad)
	}
	for _, ch := range "good-ref" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.BisectStartConfirm || m.BisectStartGood != "good-ref" {
		t.Fatalf("good ref selection = confirm=%v good=%q", m.BisectStartConfirm, m.BisectStartGood)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.BisectStartConfirm {
		t.Fatalf("bisect start confirmation = cmdnil=%v state=%v confirm=%v", cmd == nil, m.State, m.BisectStartConfirm)
	}
}

func TestBisectWorkspaceSupportsGlobalQuitKeys(t *testing.T) {
	for _, shortcut := range []string{"q", "ctrl+c"} {
		t.Run(shortcut, func(t *testing.T) {
			m := New()
			m.Workspace.Navigate(workspace.Bisect, "Bisect")
			updated, cmd := m.Update(key(shortcut))
			m = updated.(Model)
			if cmd == nil || m.State != StateShutdown {
				t.Fatalf("quit shortcut %q = cmdnil=%v state=%v", shortcut, cmd == nil, m.State)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("quit shortcut %q returned %T, want tea.QuitMsg", shortcut, cmd())
			}
		})
	}
}

func TestBisectWorkspaceCanTypeQuitKeyIntoStartRef(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	m.BisectStartMode = "bad"
	updated, cmd := m.Update(key("q"))
	m = updated.(Model)
	if cmd != nil || m.State == StateShutdown || m.BisectStartInput != "q" {
		t.Fatalf("start-ref q input = cmdnil=%v state=%v input=%q", cmd == nil, m.State, m.BisectStartInput)
	}
}

func TestBisectWorkspaceInspectsCandidateAndMapsMouseActions(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	m.Bisect = bisect.State{Repository: m.Discovery.Root, Active: true, Candidate: "0123456789012345678901234567890123456789"}
	updated, cmd := m.Update(key("i"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending {
		t.Fatalf("candidate inspect = cmdnil=%v state=%v", cmd == nil, m.State)
	}
	m.State, m.Status = StateReady, ""
	updated, cmd = m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 5, Y: 9})
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || !strings.Contains(m.Status, "good") {
		t.Fatalf("mouse good action = cmdnil=%v state=%v status=%q", cmd == nil, m.State, m.Status)
	}
}

func TestBisectCandidateInspectorShowsCommitPatch(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "known good")
	runner := git.NewRunner(root)
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("candidate change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
	gitMustRunAppTest(t, ctx, runner, "commit", "-m", "candidate change")
	head, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(head.Stdout))
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRepository(discovery)
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	m.Bisect = bisect.State{Repository: discovery.Root, Active: true, Candidate: sha}
	updated, cmd := m.Update(key("i"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("candidate inspection did not schedule a Git read")
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if m.HistoryInspector.Commit.SHA != sha || !strings.Contains(m.HistoryInspector.Diff, "+candidate change") {
		t.Fatalf("candidate inspector = %#v", m.HistoryInspector)
	}
	view := m.bisectWorkspaceView() + "\n" + inspectorText(m.HistoryInspector)
	if !strings.Contains(view, "Patch:") || !strings.Contains(view, "+candidate change") {
		t.Fatalf("bisect candidate patch missing from workspace view: %q", view)
	}
}

func TestBisectWorkspaceCompletesManualLoopInRealRepository(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "known good")
	runner := git.NewRunner(root)
	head := func() string {
		t.Helper()
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}
	good := head()
	for _, value := range []string{"still good", "first bad", "still bad", "known bad"} {
		if err := os.WriteFile(filepath.Join(root, "README"), []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitMustRunAppTest(t, ctx, runner, "add", "--", "README")
		gitMustRunAppTest(t, ctx, runner, "commit", "-m", value)
	}
	bad := head()
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runner.Run(context.Background(), "bisect", "reset") })
	m := NewRepository(discovery)
	defer func() { _ = m.Close() }()
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	m.BisectStartBad, m.BisectStartGood, m.BisectStartConfirm = bad, good, true
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("bisect start was not scheduled")
	}
	updated, refresh := m.Update(cmd())
	m = updated.(Model)
	if refresh == nil || !m.Bisect.Active || m.Bisect.Candidate == "" || m.Snapshot.Branch.OID != m.Bisect.Candidate {
		t.Fatalf("start state = bisect=%#v snapshot=%#v refreshnil=%v", m.Bisect, m.Snapshot.Branch, refresh == nil)
	}
	if m.Bisect.Subject == "" || !m.Bisect.HasEstimate || !strings.Contains(m.bisectWorkspaceView(), "remaining:") || !strings.Contains(m.bisectWorkspaceView(), "subject:") {
		t.Fatalf("candidate presentation = %#v, view=%q", m.Bisect, m.bisectWorkspaceView())
	}
	firstCandidate := m.Bisect.Candidate
	updated, cmd = m.Update(key("g"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("bisect good was not scheduled")
	}
	updated, refresh = m.Update(cmd())
	m = updated.(Model)
	if refresh == nil || m.Bisect.Good != firstCandidate || m.Bisect.Candidate == firstCandidate || m.Snapshot.Branch.OID != m.Bisect.Candidate {
		t.Fatalf("good state = bisect=%#v snapshot=%#v refreshnil=%v", m.Bisect, m.Snapshot.Branch, refresh == nil)
	}
	secondCandidate := m.Bisect.Candidate
	updated, cmd = m.Update(key("b"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("bisect bad was not scheduled")
	}
	updated, refresh = m.Update(cmd())
	m = updated.(Model)
	if refresh == nil || m.Bisect.Bad != secondCandidate || m.Snapshot.Branch.OID != m.Bisect.Candidate {
		t.Fatalf("bad state = bisect=%#v snapshot=%#v refreshnil=%v", m.Bisect, m.Snapshot.Branch, refresh == nil)
	}
	updated, cmd = m.Update(key("x"))
	m = updated.(Model)
	if cmd != nil || !m.BisectResetConfirm {
		t.Fatalf("reset confirmation = cmdnil=%v confirm=%v", cmd == nil, m.BisectResetConfirm)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("confirmed bisect reset was not scheduled")
	}
	updated, refresh = m.Update(cmd())
	m = updated.(Model)
	if refresh == nil || m.Bisect.Active || head() != bad {
		t.Fatalf("reset state = bisect=%#v HEAD=%s refreshnil=%v", m.Bisect, head(), refresh == nil)
	}
}

func TestBisectWorkspaceCollectsAutomatedRunArgvBeforeConfirmation(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	updated, _ := m.Update(key("A"))
	m = updated.(Model)
	for _, ch := range "test runner" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	for _, ch := range "case with spaces" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.BisectRunConfirm || m.BisectRunExecutable != "test runner" || len(m.BisectRunArgs) != 1 || m.BisectRunArgs[0] != "case with spaces" {
		t.Fatalf("automated run prompt = confirm=%v executable=%q args=%q", m.BisectRunConfirm, m.BisectRunExecutable, m.BisectRunArgs)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.State != StateOperationPending || m.BisectRunConfirm {
		t.Fatalf("automated run confirmation = cmdnil=%v state=%v confirm=%v", cmd == nil, m.State, m.BisectRunConfirm)
	}
}

func TestAppendBisectDisplayOutputSanitizesAndBoundsStreamedText(t *testing.T) {
	value := appendBisectDisplayOutput("", git.OutputChunk{Stream: git.StderrStream, Data: []byte("\x1b[31mwarning\x1b[0m\n")})
	if strings.ContainsAny(value, "\x1b\r\x00") || !strings.Contains(value, "[stderr]") || !strings.Contains(value, "warning") {
		t.Fatalf("streamed output = %q", value)
	}
	value = appendBisectDisplayOutput(value, git.OutputChunk{Stream: git.StdoutStream, Data: []byte(strings.Repeat("x", maxBisectDisplayOutput+32))})
	if len([]rune(value)) > maxBisectDisplayOutput {
		t.Fatalf("streamed output exceeded display bound: %d", len([]rune(value)))
	}
}

func TestKeyboardSelectionOpensDiffWithoutPriorExplicitOpen(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	m.Snapshot.Entries = []repo.Entry{
		{Path: repo.Path("a-staged.txt"), Staged: true},
		{Path: repo.Path("b-mixed.txt"), Staged: true, Unstaged: true},
		{Path: repo.Path("c-unstaged.txt"), Unstaged: true},
	}
	m.Files.SetEntries(m.Snapshot.Entries)

	updated, first := m.Update(key("down"))
	m = updated.(Model)
	if first == nil || m.Files.Selected != 1 || m.DiffPath != "b-mixed.txt" || m.DiffStaged || !m.DiffLoading {
		t.Fatalf("mixed selection diff = commandnil=%v selected=%d path=%q staged=%v loading=%v", first == nil, m.Files.Selected, m.DiffPath, m.DiffStaged, m.DiffLoading)
	}
	firstRequest := m.DiffRequest
	updated, second := m.Update(key("down"))
	m = updated.(Model)
	if second == nil || m.Files.Selected != 2 || m.DiffPath != "c-unstaged.txt" || m.DiffStaged || m.DiffRequest <= firstRequest {
		t.Fatalf("unstaged selection diff = commandnil=%v selected=%d path=%q staged=%v request=%d", second == nil, m.Files.Selected, m.DiffPath, m.DiffStaged, m.DiffRequest)
	}
	updated, third := m.Update(key("up"))
	m = updated.(Model)
	if third == nil || m.Files.Selected != 1 || m.DiffPath != "b-mixed.txt" || m.DiffStaged {
		t.Fatalf("up selection diff = commandnil=%v selected=%d path=%q staged=%v", third == nil, m.Files.Selected, m.DiffPath, m.DiffStaged)
	}
	updated, fourth := m.Update(key("up"))
	m = updated.(Model)
	if fourth == nil || m.Files.Selected != 0 || m.DiffPath != "a-staged.txt" || !m.DiffStaged {
		t.Fatalf("staged selection diff = commandnil=%v selected=%d path=%q staged=%v", fourth == nil, m.Files.Selected, m.DiffPath, m.DiffStaged)
	}
	m.closeDiff()
}

func TestInitialStatusSnapshotPreviewsSelectedUnstagedDiffOnce(t *testing.T) {
	m := NewRepository(git.Discovery{Root: t.TempDir()})
	updated, command := m.Update(SnapshotMsg{Snapshot: repo.Snapshot{Entries: []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}}})
	m = updated.(Model)
	if command == nil || !m.DiffAutoPreviewed || !m.DiffLoading || m.DiffPath != "notes.txt" || m.DiffStaged {
		t.Fatalf("initial preview = commandnil=%v previewed=%v loading=%v path=%q staged=%v", command == nil, m.DiffAutoPreviewed, m.DiffLoading, m.DiffPath, m.DiffStaged)
	}
	m.closeDiff()
	updated, command = m.Update(SnapshotMsg{Snapshot: repo.Snapshot{Entries: []repo.Entry{{Path: repo.Path("notes.txt"), Unstaged: true}}}})
	m = updated.(Model)
	if command != nil || m.DiffPath != "" || m.DiffLoading {
		t.Fatalf("refresh reopened closed preview = commandnil=%v path=%q loading=%v", command == nil, m.DiffPath, m.DiffLoading)
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
	if command == nil || m.LowerPane != "unpushed" || !strings.Contains(ansi.Strip(m.statusView()), "Unpushed commits") {
		t.Fatalf("unpushed shortcut: commandnil=%v pane=%q", command == nil, m.LowerPane)
	}
	updated, _ = m.Update(key("B"))
	m = updated.(Model)
	if m.LowerPane != "branches" || !strings.Contains(ansi.Strip(m.statusView()), "Branches") {
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
	if _, err := runner.Run(ctx, "-c", "commit.gpgsign=false", "-c", "user.name=gitwatch", "-c", "user.email=gitwatch@example.com", "commit", "-m", "initial"); err != nil {
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
	if text == "ctrl+c" {
		return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
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

func TestPaletteIndexesAllRegisteredRepositoriesAndOpensSelectedRepository(t *testing.T) {
	m := New()
	m.Repositories = repoview.New([]registry.Row{
		{Repository: registry.Repository{Name: "alpha", Path: "/alpha"}},
		{Repository: registry.Repository{Name: "beta", Path: "/beta"}},
	})
	m.openPalette()
	found := ""
	for _, result := range m.PaletteResults {
		if strings.Contains(result.Label, "beta") {
			found = result.ID
			break
		}
	}
	if found != "repository_open_1" {
		t.Fatalf("repository palette target = %q", found)
	}
	cmd := m.executePaletteAction(found)
	if cmd == nil || m.Repositories.Selected != 1 || m.State != StateOperationPending {
		t.Fatalf("repository palette execution = cmdnil=%v selected=%d state=%v", cmd == nil, m.Repositories.Selected, m.State)
	}
}

func TestPaletteIndexesLoadedBranchCommitAndFileTargets(t *testing.T) {
	m := New()
	m.Discovery.Root = "/repo"
	m.Branches = branchview.New([]branches.Branch{{Name: "feature"}, {Name: "release"}})
	m.History = historyview.New([]history.Commit{{Short: "abc123", Subject: "improve search"}})
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("README.md"), Untracked: true}})
	m.openPalette()
	for _, target := range []struct {
		id   string
		view workspace.View
	}{
		{id: "palette_branch_1", view: workspace.Branches},
		{id: "palette_commit_0", view: workspace.Log},
		{id: "palette_file_0", view: workspace.Status},
	} {
		_ = m.executePaletteAction(target.id)
		if got := m.currentView(); got != target.view {
			t.Fatalf("palette target %q view = %q, want %q", target.id, got, target.view)
		}
	}
	if m.Branches.Selected != 1 || m.History.Selected != 0 || m.Files.Selected != 0 {
		t.Fatalf("palette selections = branch %d commit %d file %d", m.Branches.Selected, m.History.Selected, m.Files.Selected)
	}
}

func TestPaletteIndexesLoadedProviderAndPluginTargets(t *testing.T) {
	m := New()
	m.GitHub.SetPullRequests([]provider.PullRequest{{Number: 7, Title: "Improve provider navigation"}})
	m.GitHub.SetIssues([]provider.Issue{{Number: 9, Title: "Track acceptance"}})
	m.GitHub.SetReleases([]provider.Release{{TagName: "v1.2.3", Name: "Stable"}})
	m.Plugins.SetEntries([]plugins.Entry{{Manifest: plugins.Manifest{ID: "example", Name: "Example plugin"}}})
	m.openPalette()
	for _, target := range []struct {
		id   string
		view workspace.View
	}{
		{id: "palette_pr_0", view: workspace.GitHub},
		{id: "palette_issue_0", view: workspace.GitHub},
		{id: "palette_release_0", view: workspace.GitHub},
		{id: "palette_plugin_0", view: workspace.Plugins},
	} {
		_ = m.executePaletteAction(target.id)
		if got := m.currentView(); got != target.view {
			t.Fatalf("palette target %q view = %q, want %q", target.id, got, target.view)
		}
	}
	if m.GitHub.Pull.Number != 7 || m.Plugins.Selected != 0 {
		t.Fatalf("provider/plugin selections = pull %d plugin %d", m.GitHub.Pull.Number, m.Plugins.Selected)
	}
}

func TestPluginNotificationContributionUsesSessionNotificationModel(t *testing.T) {
	m := New()
	entry := plugins.Entry{
		Manifest: plugins.Manifest{ID: "health", Name: "Health"},
		Enabled:  true,
		Healthy:  true,
		Contributions: []publicplugin.Contribution{{
			SchemaVersion: publicplugin.APIVersion2,
			Kind:          "notification",
			Title:         "Provider ready",
			Description:   "GitHub data is available",
			ReadOnly:      true,
		}},
	}
	updated, cmd := m.Update(PluginsReadyMsg{Entries: []plugins.Entry{entry}})
	m = updated.(Model)
	if cmd != nil || m.Notifications == nil {
		t.Fatalf("plugin notification update = cmd=%v notifications=%v", cmd != nil, m.Notifications != nil)
	}
	items := m.Notifications.Items()
	if len(items) != 1 || items[0].Kind != notifications.PluginContribution || items[0].Title != "Provider ready" || items[0].Message != "GitHub data is available" {
		t.Fatalf("plugin notifications = %#v", items)
	}
}

func TestPluginMetadataActionUsesHostProviderFromCommandPalette(t *testing.T) {
	m := New()
	defer func() { _ = m.Close() }()
	m.PluginsEnabled, m.GitHubEnabled = true, true
	m.Discovery = git.Discovery{Root: t.TempDir()}
	contribution := publicplugin.Contribution{
		SchemaVersion: publicplugin.APIVersion2,
		Kind:          "repository_metadata",
		Title:         "Repository metadata",
		Action: &publicplugin.ActionSpec{
			ID: "github-repository", Title: "Open GitHub repository metadata",
			Context: "repository", Provider: publicplugin.ActionProviderGitHubRepository, ReadOnly: true,
		},
		ReadOnly: true,
	}
	entry := plugins.Entry{
		Manifest: plugins.Manifest{ID: "metadata", Name: "Metadata", APIVersion: publicplugin.APIVersion2},
		Enabled:  true, Healthy: true,
		GrantedCapabilities: []plugins.Capability{plugins.CapabilityContextAction, plugins.CapabilityRepositoryMeta},
		Contributions:       []publicplugin.Contribution{contribution},
	}
	m.Plugins.SetEntries([]plugins.Entry{entry})
	actionID := "plugin_metadata_0_0"
	var found bool
	for _, action := range m.paletteActions() {
		if action.ID == actionID {
			found = action.Enabled && strings.Contains(action.Label, "Open GitHub repository metadata")
			break
		}
	}
	if !found {
		t.Fatal("negotiated plugin provider action was not available in the command palette")
	}
	if command := m.executePaletteAction(actionID); command == nil || m.currentView() != workspace.GitHub {
		t.Fatalf("provider action route = command:%v view:%q", command != nil, m.currentView())
	}

	m.GitHubEnabled = false
	for _, action := range m.paletteActions() {
		if action.ID == actionID && action.Enabled {
			t.Fatal("plugin provider action bypassed the disabled host provider")
		}
	}
	m.Workspace = workspace.New()
	if command := m.executePaletteAction(actionID); command != nil || m.currentView() != workspace.Status {
		t.Fatalf("disabled provider action route = command:%v view:%q", command != nil, m.currentView())
	}
}

func TestPaletteReindexesWhenLoadedRepositoryStateChanges(t *testing.T) {
	m := New()
	m.Repositories = repoview.New([]registry.Row{{Repository: registry.Repository{Name: "gone", Path: "/gone"}}})
	m.openPalette()
	if !containsPaletteID(m.PaletteResults, "repository_open_0") {
		t.Fatal("initial repository target missing")
	}
	updated, _ := m.Update(RepositoriesReadyMsg{Rows: []registry.Row{}, Repositories: []registry.Repository{}})
	m = updated.(Model)
	if containsPaletteID(m.PaletteResults, "repository_open_0") {
		t.Fatalf("stale repository target remained: %#v", m.PaletteResults)
	}
}

func TestLateGitHubReadyMessageCannotRepopulateCurrentPalette(t *testing.T) {
	m := New()
	m.repositoryGeneration = 2
	m.GitHub.SetPullRequests([]provider.PullRequest{{Number: 1, Title: "current"}})
	m.openPalette()
	if !containsPaletteID(m.PaletteResults, "palette_pr_0") {
		t.Fatal("current provider target missing")
	}
	updated, _ := m.Update(GitHubReadyMsg{
		Generation: 1,
		Pulls:      []provider.PullRequest{{Number: 99, Title: "stale"}},
	})
	m = updated.(Model)
	if len(m.GitHub.Pulls) != 1 || m.GitHub.Pulls[0].Number != 1 {
		t.Fatalf("stale provider data applied: %#v", m.GitHub.Pulls)
	}
}

func TestRepositoryDashboardUsesLoadedHistoryActivity(t *testing.T) {
	m := New()
	m.Discovery.Root = "/repo"
	now := time.Now()
	m.HistoryCommits = []history.Commit{
		{Unix: now.Unix()},
		{Unix: now.Add(-24 * time.Hour).Unix()},
	}
	rows := m.applyCommitActivity([]registry.Row{
		{Repository: registry.Repository{Path: "/repo"}},
		{Repository: registry.Repository{Path: "/other"}},
	})
	if len(rows[0].Activity) != 8 || rows[0].Activity[7] != 1 || rows[0].Activity[6] != 1 {
		t.Fatalf("active repository activity = %#v", rows[0].Activity)
	}
	if len(rows[1].Activity) != 0 {
		t.Fatalf("unrelated repository activity = %#v", rows[1].Activity)
	}
}

func TestRepositoryDashboardProjectsCachedCIAttentionByRepository(t *testing.T) {
	m := New()
	m.ProviderCI = map[string]providerCIAttention{
		"/repo": {State: "failing", Stale: true, Attention: "checks"},
	}
	rows := m.applyProviderCIAttention([]registry.Row{
		{Repository: registry.Repository{Path: "/repo"}},
		{Repository: registry.Repository{Path: "/other"}},
	})
	if rows[0].ProviderCIState != "failing" || !rows[0].ProviderCIStale || rows[0].ProviderCIAttention != "checks" {
		t.Fatalf("cached CI state = %#v", rows[0])
	}
	if rows[1].ProviderCIState != "" || rows[1].Attention != "" {
		t.Fatalf("unrelated repository changed = %#v", rows[1])
	}
}

func TestRepositoriesReadySchedulesBackgroundCIWithoutDelayingLocalStatus(t *testing.T) {
	m := New()
	m.GitHubEnabled = true
	rows := []registry.Row{
		{Repository: registry.Repository{Name: "one", Path: "/one"}, Branch: "main", State: "ready"},
		{Repository: registry.Repository{Name: "two", Path: "/two"}, Branch: "trunk", State: "ready"},
	}
	updated, cmd := m.Update(RepositoriesReadyMsg{Rows: rows})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("GitHub-enabled repository refresh did not schedule background CI work")
	}
	if m.State != StateReady || len(m.Repositories.AllRows) != 2 {
		t.Fatalf("local repository refresh waited for provider work: state=%v rows=%d", m.State, len(m.Repositories.AllRows))
	}
	for _, row := range m.Repositories.AllRows {
		if row.State != "ready" || row.ProviderCIState != "" {
			t.Fatalf("provider scheduling changed local row before results: %#v", row)
		}
	}
}

func TestDashboardCIResultsAreGenerationScopedAndDoNotChangeRefreshState(t *testing.T) {
	m := New()
	m.repositoryGeneration = 3
	m.RepositoryCIRequest = 4
	m.State = StateRefreshing
	m.Repositories.SetRows([]registry.Row{{Repository: registry.Repository{Path: "/repo"}, State: "ready"}})
	stale := DashboardCIReadyMsg{RepositoryGeneration: 3, RequestGeneration: 3, Summaries: []provider.CISummary{{Key: "/repo", State: "failing", Attention: "checks"}}}
	updated, cmd := m.Update(stale)
	got := updated.(Model)
	if cmd != nil || len(got.ProviderCI) != 0 || got.State != StateRefreshing {
		t.Fatalf("stale provider result applied: command=%v provider=%#v state=%v", cmd != nil, got.ProviderCI, got.State)
	}
	current := DashboardCIReadyMsg{RepositoryGeneration: 3, RequestGeneration: 4, Summaries: []provider.CISummary{{Key: "/repo", State: "failing", Attention: "checks", Stale: true}}}
	updated, cmd = got.Update(current)
	got = updated.(Model)
	if cmd != nil || got.State != StateRefreshing || got.ProviderCI["/repo"].State != "failing" || !got.ProviderCI["/repo"].Stale {
		t.Fatalf("current provider result = state %v, provider %#v, command=%v", got.State, got.ProviderCI, cmd != nil)
	}
	if row := got.Repositories.AllRows[0]; row.ProviderCIState != "failing" || !row.ProviderCIStale {
		t.Fatalf("dashboard row = %#v", row)
	}
}

func TestDashboardCISummaryDistinguishesProviderFailureFromCachedChecks(t *testing.T) {
	failed := dashboardCISummary(context.Background(), "/repo", provider.ChecksSnapshot{}, false, provider.ErrProviderUnavailable)
	if failed.State != provider.StateUnavailable || failed.Attention != string(provider.StateUnavailable) || failed.Stale {
		t.Fatalf("uncached provider failure was misprojected: %#v", failed)
	}
	stale := dashboardCISummary(context.Background(), "/repo", provider.ChecksSnapshot{Failing: 1}, true, provider.ErrProviderUnavailable)
	if stale.State != "failing" || stale.Attention != string(provider.StateUnavailable) || !stale.Stale {
		t.Fatalf("stale failing checks lost provider degradation: %#v", stale)
	}
	pending := dashboardCISummary(context.Background(), "/repo", provider.ChecksSnapshot{Pending: 1}, false, nil)
	if pending.State != "pending" || pending.Attention != "pending" {
		t.Fatalf("pending checks = %#v", pending)
	}
}

func containsPaletteID(results []commands.Match, id string) bool {
	for _, result := range results {
		if result.ID == id {
			return true
		}
	}
	return false
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

func TestGitHubUnavailableErrorRemainsInsideOptionalWorkspace(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.State, m.Status = StateReady, "working tree ready"
	updated, _ := m.Update(GitHubReadyMsg{Err: provider.ErrNoGitHubRemote})
	m = updated.(Model)
	if m.State != StateReady || m.Status != "GitHub provider unavailable; local Git remains available" {
		t.Fatalf("optional provider error changed core state: state=%v status=%q", m.State, m.Status)
	}
	view := m.GitHub.View()
	if !strings.Contains(view, "no GitHub remote detected") || !strings.Contains(view, "Add a GitHub remote") {
		t.Fatalf("optional provider error missing from GitHub workspace: %s", view)
	}
}

func TestLateGitHubPRCreationCannotMutateCurrentRepositoryView(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/current"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.repositoryGeneration = 3
	m.State, m.Status = StateReady, "current repository ready"
	m.GitHub.SetData(provider.Repository{Host: "github.com", Owner: "current", Name: "repo"}, "main", provider.PullRequest{Number: 1, Title: "Current"}, provider.ChecksSnapshot{})
	updated, command := m.Update(GitHubPullRequestCreatedMsg{
		Generation: 2,
		Repository: provider.Repository{Host: "github.com", Owner: "previous", Name: "repo"},
		Branch:     "feature",
		Pull:       provider.PullRequest{Number: 99, Title: "Stale"},
	})
	m = updated.(Model)
	if command != nil || m.State != StateReady || m.Status != "current repository ready" || m.GitHub.Pull.Number != 1 {
		t.Fatalf("late PR creation crossed repository generation: cmd=%v state=%v status=%q pull=%#v", command != nil, m.State, m.Status, m.GitHub.Pull)
	}
}

func TestGitHubProviderFailureDoesNotHideIndependentResourcesOrBreakGitState(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "provider isolation")
	runner := git.NewRunner(root)
	gitMustRunAppTest(t, ctx, runner, "remote", "add", "origin", "https://github.com/octo/repo.git")

	var branchPulls, pullLists, checkRuns, issueLists, releaseLists atomic.Int32
	transport := appRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		respond := func(status int, body string) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
		}
		switch {
		case r.URL.Path == "/repos/octo/repo/pulls" && r.URL.Query().Get("head") != "":
			branchPulls.Add(1)
			return respond(http.StatusOK, `[]`)
		case strings.HasSuffix(r.URL.Path, "/check-runs"):
			checkRuns.Add(1)
			return respond(http.StatusServiceUnavailable, `{}`)
		case r.URL.Path == "/repos/octo/repo/pulls":
			pullLists.Add(1)
			return respond(http.StatusOK, `[{"number":8,"title":"Open PR","state":"open","head":{"ref":"feature"},"base":{"ref":"main"}}]`)
		case r.URL.Path == "/repos/octo/repo/issues":
			issueLists.Add(1)
			return respond(http.StatusOK, `[{"number":9,"title":"Open issue","state":"open"}]`)
		case r.URL.Path == "/repos/octo/repo/releases":
			releaseLists.Add(1)
			return respond(http.StatusOK, `[{"id":1,"tag_name":"v1.0.0","name":"First"}]`)
		default:
			return respond(http.StatusNotFound, `{}`)
		}
	})

	m := NewRepositoryWithConfig(git.Discovery{Root: root}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	defer func() { _ = m.Close() }()
	m.Snapshot.Branch.Name = "feature"
	client := provider.GitHubClient{BaseURL: "https://api.test", HTTPClient: &http.Client{Transport: transport}}

	msg, ok := m.loadGitHubWithClient(&client)().(GitHubReadyMsg)
	if !ok {
		t.Fatal("GitHub loader returned the wrong message")
	}
	if msg.Err != nil || msg.Pull.Number != 0 || len(msg.Pulls) != 1 || len(msg.Issues) != 1 || len(msg.Releases) != 1 {
		t.Fatalf("partial provider snapshot = %#v", msg)
	}
	if branchPulls.Load() != 1 || pullLists.Load() != 1 || checkRuns.Load() != 1 || issueLists.Load() != 1 || releaseLists.Load() != 1 {
		t.Fatalf("independent provider requests: branch=%d pulls=%d checks=%d issues=%d releases=%d", branchPulls.Load(), pullLists.Load(), checkRuns.Load(), issueLists.Load(), releaseLists.Load())
	}
	if len(msg.Warnings) != 1 || msg.Warnings[0].Resource != "checks" {
		t.Fatalf("provider resource warnings = %#v", msg.Warnings)
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)
	view := m.GitHub.View()
	for _, want := range []string{"No open pull request for the current branch", "PR #8: Open PR", "Issue #9: Open issue", "v1.0.0", "checks: GitHub HTTP 503"} {
		if !strings.Contains(view, want) {
			t.Fatalf("GitHub view missing %q: %s", want, view)
		}
	}
	if m.State != StateReady {
		t.Fatalf("optional provider failure changed core app state: %v", m.State)
	}
}

func TestGitHubCheckoutRequiresValidatedProviderRefAndExplicitConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 1, Head: "feature/topic", State: "open"}, provider.ChecksSnapshot{})
	m.Remotes.SetDashboard(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin", FetchURL: "https://github.com/octo/repo.git"}}})
	updated, cmd := m.Update(key("x"))
	m = updated.(Model)
	if cmd != nil || !m.RemoteBranchConfirm || m.RemoteBranchTarget.RemoteName != "origin" || m.RemoteBranchTarget.RemoteBranch != "feature/topic" {
		t.Fatalf("provider checkout confirmation = cmd=%v confirm=%v target=%#v", cmd != nil, m.RemoteBranchConfirm, m.RemoteBranchTarget)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.RemoteBranchConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("provider checkout cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.RemoteBranchConfirm, m.Status)
	}
	m.GitHub.Pull.Head = "../escape"
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if !strings.Contains(m.Status, "invalid provider checkout ref") {
		t.Fatalf("unsafe provider ref status = %q", m.Status)
	}
}

func TestGitHubCreatePRFormRequiresExplicitConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Base: "main", State: "open"}, provider.ChecksSnapshot{})
	m.Snapshot.Branch.Name = "feature"
	updated, cmd := m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubCreateMode || m.GitHubCreateField != 0 {
		t.Fatalf("PR form start = cmd=%v mode=%v field=%d", cmd != nil, m.GitHubCreateMode, m.GitHubCreateField)
	}
	m.updateGitHubCreateKey("T")
	m.updateGitHubCreateKey("enter")
	m.updateGitHubCreateKey("enter")
	m.updateGitHubCreateKey("enter")
	if m.GitHubCreateMode || !m.GitHubCreateConfirm || !strings.Contains(m.Status, "create GitHub PR") {
		t.Fatalf("PR form confirmation = mode=%v confirm=%v status=%q", m.GitHubCreateMode, m.GitHubCreateConfirm, m.Status)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.GitHubCreateConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("PR form cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.GitHubCreateConfirm, m.Status)
	}
}

func TestGitHubMergeRefreshesBeforeExplicitConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	pull := provider.PullRequest{Number: 4, Title: "Improve", State: "open", HeadSHA: "abc", Mergeable: "clean"}
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", pull, provider.ChecksSnapshot{})
	updated, cmd := m.Update(key("m"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubMergeMode {
		t.Fatalf("merge start = cmd=%v mode=%v", cmd != nil, m.GitHubMergeMode)
	}
	updated, _ = m.Update(key("s"))
	m = updated.(Model)
	updated, cmd = m.Update(key("enter"))
	m = updated.(Model)
	if cmd == nil || m.GitHubMergeMode || !m.GitHubMergeRefresh || m.GitHubMergeConfirm {
		t.Fatalf("merge refresh = cmd=%v mode=%v refresh=%v confirm=%v", cmd != nil, m.GitHubMergeMode, m.GitHubMergeRefresh, m.GitHubMergeConfirm)
	}
	updated, _ = m.Update(GitHubReadyMsg{Repository: provider.Repository{Owner: "octo", Name: "repo"}, Branch: "main", Pull: pull, Checks: provider.ChecksSnapshot{Passing: 1}, Review: provider.ReviewSnapshot{Approved: 1}})
	m = updated.(Model)
	if !m.GitHubMergeConfirm || !strings.Contains(m.Status, "squash") {
		t.Fatalf("merge confirmation = %v status=%q", m.GitHubMergeConfirm, m.Status)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.GitHubMergeConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("merge cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.GitHubMergeConfirm, m.Status)
	}
}

func TestGitHubMergeOffersSeparateRemoteBranchDeletion(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Head: "feature/topic", State: "open"}, provider.ChecksSnapshot{})
	updated, cmd := m.Update(GitHubMergeFinishedMsg{Result: provider.MergeResult{Merged: true}})
	m = updated.(Model)
	if cmd != nil || !m.GitHubBranchDeleteConfirm || m.GitHubBranchDeleteTarget != "feature/topic" {
		t.Fatalf("branch deletion offer = cmd=%v confirm=%v target=%q", cmd != nil, m.GitHubBranchDeleteConfirm, m.GitHubBranchDeleteTarget)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd == nil || m.GitHubBranchDeleteConfirm || m.GitHubBranchDeleteTarget != "" || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("branch deletion cancellation = cmd=%v confirm=%v target=%q status=%q", cmd != nil, m.GitHubBranchDeleteConfirm, m.GitHubBranchDeleteTarget, m.Status)
	}
}

func TestGitHubReviewActionsRequireIntentAndReason(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	pull := provider.PullRequest{Number: 4, Title: "Improve", State: "open", HeadSHA: "abc"}
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", pull, provider.ChecksSnapshot{})
	updated, cmd := m.Update(key("A"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubReviewConfirm || m.GitHubReviewEvent != provider.ReviewEventApprove {
		t.Fatalf("approve confirmation = cmd=%v confirm=%v event=%q", cmd != nil, m.GitHubReviewConfirm, m.GitHubReviewEvent)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	updated, cmd = m.Update(key("R"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubReviewMode || m.GitHubReviewEvent != provider.ReviewEventRequestChanges {
		t.Fatalf("request-changes mode = cmd=%v mode=%v event=%q", cmd != nil, m.GitHubReviewMode, m.GitHubReviewEvent)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.GitHubReviewConfirm || !strings.Contains(m.Status, "requires a reason") {
		t.Fatalf("empty request-changes body = confirm=%v status=%q", m.GitHubReviewConfirm, m.Status)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.GitHubReviewConfirm || !strings.Contains(m.Status, "submit GitHub request_changes") {
		t.Fatalf("request-changes confirmation = confirm=%v status=%q", m.GitHubReviewConfirm, m.Status)
	}
}

func TestGitHubCommentReplyTargetIsExplicitAndBoundToLoadedComment(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open", HeadSHA: "abc"}, provider.ChecksSnapshot{})
	m.GitHub.SetComments([]provider.ReviewComment{{ID: 12, Author: "reviewer", Body: "please update"}, {ID: 13, Author: "reviewer", Body: "also test"}})
	updated, cmd := m.Update(key("]"))
	m = updated.(Model)
	if cmd != nil || m.GitHubReplyCommentID != 13 {
		t.Fatalf("reply target selection = cmd=%v id=%d", cmd != nil, m.GitHubReplyCommentID)
	}
	updated, cmd = m.Update(key("c"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubReviewMode || m.GitHubReviewEvent != provider.ReviewEventComment || !strings.Contains(m.Status, "comment") {
		t.Fatalf("reply form = cmd=%v mode=%v event=%q status=%q", cmd != nil, m.GitHubReviewMode, m.GitHubReviewEvent, m.Status)
	}
}

func TestGitHubCheckActionsSelectAndConfirm(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	pull := provider.PullRequest{Number: 4, Title: "Improve", State: "open", HeadSHA: "abc"}
	checks := provider.ChecksSnapshot{Runs: []provider.CheckRun{
		{ID: 12, Name: "build", Status: "completed", Conclusion: "failure"},
		{ID: 13, Name: "lint", Status: "in_progress"},
	}}
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", pull, checks)
	updated, cmd := m.Update(key("j"))
	m = updated.(Model)
	if cmd != nil || m.GitHub.SelectedRun != 1 {
		t.Fatalf("check selection = cmd=%v selected=%d", cmd != nil, m.GitHub.SelectedRun)
	}
	updated, cmd = m.Update(key("K"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubCheckActionConfirm || m.GitHubCheckAction != "cancel" || m.GitHubCheckActionRunID != 13 {
		t.Fatalf("cancel confirmation = cmd=%v confirm=%v action=%q id=%d", cmd != nil, m.GitHubCheckActionConfirm, m.GitHubCheckAction, m.GitHubCheckActionRunID)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	if m.GitHubCheckActionConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("cancel state = confirm=%v status=%q", m.GitHubCheckActionConfirm, m.Status)
	}
	updated, cmd = m.Update(key("!"))
	m = updated.(Model)
	if cmd != nil || !strings.Contains(m.Status, "still running") {
		t.Fatalf("rerun running check = cmd=%v status=%q", cmd != nil, m.Status)
	}
}

func TestGitHubSelectedCheckURLOpensFromWorkflowWorkspace(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open"}, provider.ChecksSnapshot{Runs: []provider.CheckRun{
		{Name: "build", Status: "completed", Conclusion: "success", URL: "https://github.com/octo/repo/actions/runs/10"},
		{Name: "lint", Status: "in_progress", URL: "https://github.com/octo/repo/actions/runs/11"},
	}})
	updated, cmd := m.Update(key("j"))
	m = updated.(Model)
	if cmd != nil || m.GitHub.SelectedRun != 1 {
		t.Fatalf("check selection = cmd=%v selected=%d", cmd != nil, m.GitHub.SelectedRun)
	}
	updated, cmd = m.Update(key("W"))
	m = updated.(Model)
	if cmd == nil || !strings.Contains(m.Status, "lint") {
		t.Fatalf("selected check URL = cmd=%v status=%q", cmd != nil, m.Status)
	}
}

func TestGitHubIssueFormRequiresConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 1, State: "open"}, provider.ChecksSnapshot{})
	updated, cmd := m.Update(key("I"))
	m = updated.(Model)
	if cmd != nil || !m.GitHubIssueMode || m.GitHubIssueField != 0 {
		t.Fatalf("issue form start = cmd=%v mode=%v field=%d", cmd != nil, m.GitHubIssueMode, m.GitHubIssueField)
	}
	m.updateGitHubIssueKey("B")
	m.updateGitHubIssueKey("enter")
	m.updateGitHubIssueKey("body")
	m.updateGitHubIssueKey("enter")
	m.updateGitHubIssueKey("bug, ui")
	m.updateGitHubIssueKey("enter")
	if m.GitHubIssueMode || !m.GitHubIssueConfirm || !strings.Contains(m.Status, "create GitHub issue B") {
		t.Fatalf("issue confirmation = mode=%v confirm=%v status=%q", m.GitHubIssueMode, m.GitHubIssueConfirm, m.Status)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.GitHubIssueConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("issue cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.GitHubIssueConfirm, m.Status)
	}
}

func TestGitHubIssueAndReleaseNavigationUsesProviderURLs(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 1, State: "open"}, provider.ChecksSnapshot{})
	m.GitHub.SetIssues([]provider.Issue{{Number: 3, Title: "Bug", URL: "https://github.test/issues/3"}})
	m.GitHub.SetReleases([]provider.Release{{ID: 4, TagName: "v1.0.0", URL: "https://github.test/releases/4"}})
	updated, cmd := m.Update(key("O"))
	m = updated.(Model)
	if cmd == nil || !strings.Contains(m.Status, "issue #3") {
		t.Fatalf("issue navigation = cmd=%v status=%q", cmd != nil, m.Status)
	}
	updated, cmd = m.Update(key("L"))
	m = updated.(Model)
	if cmd == nil || !strings.Contains(m.Status, "v1.0.0") {
		t.Fatalf("release navigation = cmd=%v status=%q", cmd != nil, m.Status)
	}
}

func TestGitHubIssueAndReleasePaletteSelectionDrivesURLActions(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.GitHub.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{}, provider.ChecksSnapshot{})
	m.GitHub.SetIssues([]provider.Issue{{Number: 3, Title: "First", URL: "https://github.test/issues/3"}, {Number: 8, Title: "Second", URL: "https://github.test/issues/8"}})
	m.GitHub.SetReleases([]provider.Release{{TagName: "v1", URL: "https://github.test/releases/1"}, {TagName: "v2", URL: "https://github.test/releases/2"}})
	if command := m.executePaletteAction("palette_issue_1"); m.GitHub.SelectedIssue != 1 || m.currentView() != workspace.GitHub {
		t.Fatalf("issue palette selection = command=%v selected=%d view=%q", command != nil, m.GitHub.SelectedIssue, m.currentView())
	}
	updated, command := m.Update(key("O"))
	m = updated.(Model)
	if command == nil || !strings.Contains(m.Status, "#8") {
		t.Fatalf("selected issue URL action = command=%v status=%q", command != nil, m.Status)
	}
	if command := m.executePaletteAction("palette_release_1"); m.GitHub.SelectedRelease != 1 || m.currentView() != workspace.GitHub {
		t.Fatalf("release palette selection = command=%v selected=%d view=%q", command != nil, m.GitHub.SelectedRelease, m.currentView())
	}
	updated, command = m.Update(key("L"))
	m = updated.(Model)
	if command == nil || !strings.Contains(m.Status, "v2") {
		t.Fatalf("selected release URL action = command=%v status=%q", command != nil, m.Status)
	}
}

func TestGitHubPaletteOpensSelectedLocalCommitBranchAndTag(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	m.GitHub.SetData(provider.Repository{Host: "github.com", Owner: "octo", Name: "repo"}, "main", provider.PullRequest{}, provider.ChecksSnapshot{})
	m.History = historyview.New([]history.Commit{{SHA: "abc123", Subject: "commit"}})
	m.Branches = branchview.New([]branches.Branch{{Name: "feature/x"}})
	m.TagSnapshot = tags.Snapshot{Tags: []tags.Tag{{Name: "v1.2.3"}}}
	for _, test := range []struct {
		id, want string
	}{
		{"github_commit_selected", "opening GitHub commit abc123"},
		{"github_branch_selected", "opening GitHub tree feature/x"},
		{"github_tag_selected", "opening GitHub releases/tag v1.2.3"},
	} {
		if command := m.executePaletteAction(test.id); command == nil || m.Status != test.want {
			t.Fatalf("palette %q = command=%v status=%q, want %q", test.id, command != nil, m.Status, test.want)
		}
	}
}

func TestRepositoryBatchFetchRequiresExplicitConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{})
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	m.Repositories = repoview.New([]registry.Row{{Repository: registry.Repository{Path: "/one", Name: "one"}}})
	updated, cmd := m.Update(key("F"))
	m = updated.(Model)
	if cmd != nil || !m.RepositoryBatchConfirm || !strings.Contains(m.Status, "1 discovered repositories") {
		t.Fatalf("batch confirmation = cmd=%v confirm=%v status=%q", cmd != nil, m.RepositoryBatchConfirm, m.Status)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.RepositoryBatchConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("batch cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.RepositoryBatchConfirm, m.Status)
	}
}

func TestRepositoryBatchPullRequiresExplicitFastForwardConfirmation(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{})
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	m.Repositories = repoview.New([]registry.Row{{Repository: registry.Repository{Path: "/one", Name: "one"}, Branch: "main"}})
	updated, cmd := m.Update(key("P"))
	m = updated.(Model)
	if cmd != nil || !m.RepositoryBatchConfirm || m.RepositoryBatchAction != multirepo.ActionPull || m.RepositoryBatchStrategy != "ff-only" || !strings.Contains(m.Status, "ff-only") {
		t.Fatalf("batch pull confirmation = cmd=%v confirm=%v action=%q strategy=%q status=%q", cmd != nil, m.RepositoryBatchConfirm, m.RepositoryBatchAction, m.RepositoryBatchStrategy, m.Status)
	}
	updated, cmd = m.Update(key("n"))
	m = updated.(Model)
	if cmd != nil || m.RepositoryBatchConfirm || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("batch pull cancellation = cmd=%v confirm=%v status=%q", cmd != nil, m.RepositoryBatchConfirm, m.Status)
	}
}

func TestRepositoryBatchRetryOnlyTargetsFailedResults(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{})
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	m.Repositories = repoview.New([]registry.Row{
		{Repository: registry.Repository{Path: "/one", Name: "one"}},
		{Repository: registry.Repository{Path: "/two", Name: "two"}},
	})
	m.RepositoryBatchResults = []multirepo.Result{
		{Request: multirepo.Request{Repository: multirepo.Repository{Root: "/one"}}, Status: "failed"},
		{Request: multirepo.Request{Repository: multirepo.Repository{Root: "/two"}}, Status: "succeeded"},
	}
	updated, cmd := m.Update(key("R"))
	m = updated.(Model)
	if cmd != nil || !m.RepositoryBatchConfirm || !m.RepositoryBatchRetry || !strings.Contains(m.Status, "1 failed") {
		t.Fatalf("retry confirmation = cmd=%v confirm=%v retry=%v status=%q", cmd != nil, m.RepositoryBatchConfirm, m.RepositoryBatchRetry, m.Status)
	}
	updated, _ = m.Update(key("n"))
	m = updated.(Model)
	if m.RepositoryBatchConfirm || m.RepositoryBatchRetry || !strings.Contains(m.Status, "cancelled") {
		t.Fatalf("retry cancellation = confirm=%v retry=%v status=%q", m.RepositoryBatchConfirm, m.RepositoryBatchRetry, m.Status)
	}
}

func TestRepositoryBatchProgressCommandPreservesEventStream(t *testing.T) {
	events := make(chan tea.Msg, 2)
	events <- RepositoryBatchProgressMsg{Path: "/one", Status: "queued", Total: 1}
	events <- RepositoryBatchFinishedMsg{}
	first, ok := batchProgressCommand(events)().(RepositoryBatchProgressMsg)
	if !ok || first.Path != "/one" || first.Events == nil {
		t.Fatalf("first batch progress = %#v", first)
	}
	if _, ok := batchProgressCommand(first.Events)().(RepositoryBatchFinishedMsg); !ok {
		t.Fatalf("progress stream did not deliver terminal result")
	}
}

func TestRepositoryBatchOperationEmitsBoundedProgressBeforeResults(t *testing.T) {
	root := t.TempDir()
	m := NewRepositoryWithConfig(git.Discovery{Root: root}, config.Config{})
	m.Repositories = repoview.New([]registry.Row{{Repository: registry.Repository{Path: root, Name: "one"}}})
	m.RepositoryBatchAction = multirepo.ActionFetch
	command := m.runRepositoryBatchFetch()
	var statuses []string
	for message := command(); ; {
		switch value := message.(type) {
		case RepositoryBatchProgressMsg:
			statuses = append(statuses, value.Status)
			command = batchProgressCommand(value.Events)
			message = command()
		case RepositoryBatchFinishedMsg:
			if len(value.Results) != 1 || value.Results[0].Status != "failed" {
				t.Fatalf("batch result = %#v", value.Results)
			}
			if strings.Join(statuses, ",") != "queued,running,failed" {
				t.Fatalf("progress statuses = %v", statuses)
			}
			return
		default:
			t.Fatalf("unexpected batch message %T", message)
		}
	}
}

func TestRepositoryBatchCancelUsesActiveContext(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: "/repo"}, config.Config{})
	m.Workspace.Navigate(workspace.Repositories, "Repositories")
	called := false
	m.RepositoryBatchCancel = func() { called = true }
	updated, cmd := m.Update(key("K"))
	m = updated.(Model)
	if cmd != nil || !called || m.RepositoryBatchCancel == nil || !strings.Contains(m.Status, "cancelling") {
		t.Fatalf("batch cancellation = cmd=%v called=%v cancel=%v status=%q", cmd != nil, called, m.RepositoryBatchCancel != nil, m.Status)
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

	m = New()
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, _ = m.Update(key("X"))
	m = updated.(Model)
	for _, ch := range "v1.2.3" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.RemoteTagDeleteConfirm || m.RemoteTagDeleteMode || !strings.Contains(m.Status, "DELETE remote tag") {
		t.Fatalf("remote tag deletion confirmation = confirm=%v mode=%v status=%q", m.RemoteTagDeleteConfirm, m.RemoteTagDeleteMode, m.Status)
	}
	updated, cmd = m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.RemoteTagDeleteConfirm || len(m.Remotes.Dashboard.ActiveJobs()) != 1 {
		t.Fatalf("remote tag deletion = cmdnil=%v confirm=%v jobs=%#v", cmd == nil, m.RemoteTagDeleteConfirm, m.Remotes.Dashboard.Jobs)
	}
}

func TestRemoteLifecycleControlsUseImpactConfirmation(t *testing.T) {
	m := New()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin", FetchURL: "https://example.com/repo.git"}}})
	updated, _ := m.Update(key("A"))
	m = updated.(Model)
	for _, ch := range "backup" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if m.RemoteMutationMode != "add-url" || m.RemoteMutationRemote != "backup" {
		t.Fatalf("remote add name stage = mode=%q remote=%q", m.RemoteMutationMode, m.RemoteMutationRemote)
	}
	for _, ch := range "https://example.com/repo.git" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.RemoteMutationMode != "add" {
		t.Fatalf("remote add submit = cmdnil=%v state=%v mode=%q", command == nil, m.State, m.RemoteMutationMode)
	}

	m = New()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Remotes, "Remotes")
	m.Remotes = remoteview.New(remotes.Dashboard{Remotes: []remotes.Remote{{Name: "origin"}}})
	updated, command = m.Update(key("R"))
	m = updated.(Model)
	if command == nil || m.RemoteMutationMode != "rename-loading" {
		t.Fatalf("remote rename tracking load = cmdnil=%v mode=%q", command == nil, m.RemoteMutationMode)
	}
	updated, _ = m.Update(RemoteTrackingReadyMsg{Repository: 1, Remote: "origin", Branches: []remotes.TrackingBranch{{Local: "main", Upstream: "origin/main"}}})
	m = updated.(Model)
	for _, ch := range "upstream" {
		updated, _ = m.Update(key(string(ch)))
		m = updated.(Model)
	}
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.RemoteMutationConfirm || !strings.Contains(m.Status, "main -> origin/main") {
		t.Fatalf("remote rename impact confirmation = confirm=%v status=%q", m.RemoteMutationConfirm, m.Status)
	}
	updated, command = m.Update(key("y"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending {
		t.Fatalf("remote rename submit = cmdnil=%v state=%v", command == nil, m.State)
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

func TestBranchCheckoutPromptsRemoteTrackingName(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true, RemoteName: "origin", RemoteBranch: "main"}})
	updated, cmd := m.Update(key("enter"))
	m = updated.(Model)
	if cmd != nil || m.RemoteBranchAction != "track" || m.RemoteBranchInput != "main" {
		t.Fatalf("remote tracking prompt = cmdnil=%v action=%q input=%q", cmd == nil, m.RemoteBranchAction, m.RemoteBranchInput)
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

func TestHistoryRevertUsesBasketApplicationOrder(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{
		{SHA: "newest", Short: "newest", Subject: "new"},
		{SHA: "oldest", Short: "oldest", Subject: "old"},
	})
	if err := m.History.SetScope("/repo", "main", 1); err != nil {
		t.Fatal(err)
	}
	if err := m.History.ToggleBasket(); err != nil {
		t.Fatal(err)
	}
	m.History.Move(1)
	if err := m.History.ToggleBasket(); err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(key("R"))
	m = updated.(Model)
	if m.HistoryRevertTarget != "newest oldest" || len(m.HistoryRevertCommits) != 2 || m.HistoryRevertCommits[0] != "newest" {
		t.Fatalf("ordered revert plan = target=%q commits=%v", m.HistoryRevertTarget, m.HistoryRevertCommits)
	}
	if !contains(m.Status, "ordered SHAs") {
		t.Fatalf("ordered prompt = %q", m.Status)
	}
}

func TestHistoryRangeGestureSelectsVisibleRowsInGitOrder(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{
		{SHA: "newest", Short: "newest", Subject: "new"},
		{SHA: "middle", Short: "middle", Subject: "middle"},
		{SHA: "oldest", Short: "oldest", Subject: "old"},
	})
	if err := m.History.SetScope("/repo", "main", 1); err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(key("v"))
	m = updated.(Model)
	if !m.HistoryRangeAnchorSet || m.HistoryRangeAnchor != 0 {
		t.Fatalf("range start = set=%v anchor=%d status=%q", m.HistoryRangeAnchorSet, m.HistoryRangeAnchor, m.Status)
	}
	m.History.Move(2)
	updated, _ = m.Update(key("v"))
	m = updated.(Model)
	if m.HistoryRangeAnchorSet || m.History.Basket.Count() != 3 {
		t.Fatalf("range completion = set=%v basket=%#v status=%q", m.HistoryRangeAnchorSet, m.History.Basket, m.Status)
	}
	got := m.History.Basket.SHAs()
	if got[0] != "oldest" || got[1] != "middle" || got[2] != "newest" {
		t.Fatalf("range order = %#v", got)
	}
}

func TestHistoryRevertRequiresMainlineForMergeCommit(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.Log, "History")
	m.History = historyview.New([]history.Commit{{SHA: "merge", Short: "merge", Parents: []string{"one", "two"}}})
	if err := m.History.SetScope("/repo", "main", 1); err != nil {
		t.Fatal(err)
	}
	if err := m.History.ToggleBasket(); err != nil {
		t.Fatal(err)
	}
	updated, cmd := m.Update(key("R"))
	m = updated.(Model)
	if cmd != nil || m.HistoryRevertConfirm || !m.HistoryRevertParentMode || m.HistoryRevertParentMax != 2 {
		t.Fatalf("merge revert prompt = cmdnil=%v confirm=%v mode=%v max=%d status=%q", cmd != nil, m.HistoryRevertConfirm, m.HistoryRevertParentMode, m.HistoryRevertParentMax, m.Status)
	}
	updated, _ = m.Update(key("2"))
	m = updated.(Model)
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	if !m.HistoryRevertConfirm || m.HistoryRevertParent != 2 || m.HistoryRevertParentMode {
		t.Fatalf("mainline selection = confirm=%v mainline=%d mode=%v status=%q", m.HistoryRevertConfirm, m.HistoryRevertParent, m.HistoryRevertParentMode, m.Status)
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

func TestRemoteBranchControlsRequireQualifiedTarget(t *testing.T) {
	m := New()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true, RemoteName: "origin", RemoteBranch: "main"}})

	updated, command := m.Update(key("enter"))
	m = updated.(Model)
	if command != nil || m.RemoteBranchAction != "track" || m.RemoteBranchInput != "main" {
		t.Fatalf("remote tracking prompt = cmdnil=%v action=%q input=%q", command == nil, m.RemoteBranchAction, m.RemoteBranchInput)
	}
	updated, command = m.Update(key("enter"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.RemoteBranchAction != "" {
		t.Fatalf("remote tracking submit = cmdnil=%v state=%v action=%q", command == nil, m.State, m.RemoteBranchAction)
	}

	m = New()
	m.repositoryGeneration = 1
	m.Workspace.Navigate(workspace.Branches, "Branches")
	m.Branches = branchview.New([]branches.Branch{{Name: "origin/main", Remote: true, RemoteName: "origin", RemoteBranch: "main"}})
	updated, _ = m.Update(key("D"))
	m = updated.(Model)
	if !m.RemoteBranchConfirm || !strings.Contains(m.Status, "origin/main") {
		t.Fatalf("remote delete confirmation = confirm=%v status=%q", m.RemoteBranchConfirm, m.Status)
	}
	updated, command = m.Update(key("y"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending {
		t.Fatalf("remote delete submit = cmdnil=%v state=%v", command == nil, m.State)
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
