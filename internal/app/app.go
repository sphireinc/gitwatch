package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/commitmodel"
	"github.com/jusanchez/gitwatch/internal/config"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/history"
	"github.com/jusanchez/gitwatch/internal/remotes"
	"github.com/jusanchez/gitwatch/internal/repo"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/ui/branchview"
	"github.com/jusanchez/gitwatch/internal/ui/commitview"
	"github.com/jusanchez/gitwatch/internal/ui/historyview"
	"github.com/jusanchez/gitwatch/internal/ui/layout"
	uimouse "github.com/jusanchez/gitwatch/internal/ui/mouse"
	"github.com/jusanchez/gitwatch/internal/ui/remoteview"
	"github.com/jusanchez/gitwatch/internal/ui/stashview"
	"github.com/jusanchez/gitwatch/internal/ui/table"
	"github.com/jusanchez/gitwatch/internal/ui/theme"
	"github.com/jusanchez/gitwatch/internal/workspace"
)

type State uint8

const (
	StateLoading State = iota
	StateReady
	StateRefreshing
	StateOperationPending
	StateError
	StateModal
	StateShutdown
)

type SnapshotMsg struct{ Snapshot repo.Snapshot }
type RefreshStartedMsg struct{}
type RefreshFinishedMsg struct{ Err error }
type WatcherStateMsg struct {
	Mode string
	Err  error
}
type OperationStartedMsg struct{ Name string }
type OperationFinishedMsg struct {
	Name string
	Err  error
}
type TickMsg struct{ At time.Time }
type ToastMsg struct {
	Text  string
	Error bool
}
type ModalMsg struct {
	Open bool
	Name string
}
type FocusMsg struct{ Pane string }
type ShutdownMsg struct{}
type DiffReadyMsg struct {
	Path, Text string
	Binary     bool
	Err        error
}
type BranchesReadyMsg struct {
	Entries []branches.Branch
	Err     error
}
type StashesReadyMsg struct {
	Entries []stash.Entry
	Err     error
}
type HistoryReadyMsg struct {
	Commits []history.Commit
	Skip    int
	HasMore bool
	Err     error
}
type HistoryInspectorReadyMsg struct {
	Inspector history.Inspector
	Err       error
}
type HistoryTagsReadyMsg struct {
	Tags []history.Ref
	Err  error
}
type HistoryActionFinishedMsg struct {
	Action, Target string
	Err            error
}
type CommitFinishedMsg struct {
	SHA string
	Err error
}
type BranchOperationFinishedMsg struct {
	Name string
	Err  error
}
type StashPreviewReadyMsg struct {
	Ref, Text string
	Err       error
}
type RemotesReadyMsg struct {
	Dashboard remotes.Dashboard
	Err       error
}
type RemoteOperationFinishedMsg struct {
	Operation, Remote string
	Err               error
}

type Model struct {
	State                 State
	Width, Height         int
	Focus, Modal, Status  string
	Toast                 ToastMsg
	Snapshot              repo.Snapshot
	Discovery             git.Discovery
	Files                 table.Model
	Theme                 theme.Roles
	ctx                   context.Context
	RefreshInterval       time.Duration
	DiffPath, DiffText    string
	DiffBinary            bool
	Workspace             *workspace.Model
	Branches              branchview.Model
	Stashes               stashview.Model
	History               historyview.Model
	HistoryCommits        []history.Commit
	HistorySkip           int
	HistoryHasMore        bool
	HistoryFilter         string
	HistorySearching      bool
	HistoryInspector      history.Inspector
	HistoryTags           []history.Ref
	HistoryActionConfirm  bool
	HistoryActionTarget   string
	HistoryBranchCreating bool
	HistoryBranchTarget   string
	HistoryBranchName     string
	HistoryRevertConfirm  bool
	HistoryRevertTarget   string
	HistoryRevertInput    string
	HistoryRevertInvalid  bool
	Composer              commitview.Composer
	StashPreview          string
	StashPreviewRef       string
	Remotes               remoteview.Model
	RemoteForceConfirm    bool
}

func New() Model {
	return Model{State: StateLoading, Focus: "files", Theme: theme.New(theme.Auto, false), ctx: context.Background(), RefreshInterval: 2 * time.Second, Workspace: workspace.New()}
}
func NewRepository(d git.Discovery) Model { m := New(); m.Discovery = d; return m }
func NewRepositoryWithConfig(d git.Discovery, c config.Config) Model {
	m := NewRepository(d)
	m.RefreshInterval = c.Interval
	m.Theme = theme.New(theme.Name(c.Theme), false)
	return m
}
func (m Model) Init() tea.Cmd {
	if m.Discovery.Root == "" {
		return nil
	}
	return tea.Batch(m.refresh(), m.tick())
}

func (m Model) tick() tea.Cmd {
	interval := m.RefreshInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return TickMsg{At: t} })
}
func (m Model) refresh() tea.Cmd {
	d := m.Discovery
	return func() tea.Msg {
		s, err := git.Snapshot(context.Background(), d, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Snapshot: s}
	}
}

func (m Model) mutate() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	if e.Conflicted {
		return func() tea.Msg {
			return OperationFinishedMsg{Name: "stage", Err: fmt.Errorf("conflicted path requires external resolution")}
		}
	}
	r, path := git.NewRunner(m.Discovery.Root), append([]byte(nil), e.Path...)
	return func() tea.Msg {
		var err error
		if e.Staged && !e.Unstaged {
			_, err = r.Unstage(context.Background(), path)
		} else {
			_, err = r.Stage(context.Background(), path)
		}
		if err != nil {
			return OperationFinishedMsg{Name: "path operation", Err: err}
		}
		s, err := git.Snapshot(context.Background(), m.Discovery, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Snapshot: s}
	}
}

func (m Model) openDiff() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	path := append([]byte(nil), e.Path...)
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		d, err := r.Diff(context.Background(), path, e.Staged && !e.Unstaged)
		return DiffReadyMsg{Path: string(path), Text: string(d.Text), Binary: d.Binary, Err: err}
	}
}

func (m Model) loadBranches() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := branches.List(context.Background(), r)
		return BranchesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) checkoutSelectedBranch() tea.Cmd {
	if m.Branches.Selected < 0 || m.Branches.Selected >= len(m.Branches.Entries) {
		return nil
	}
	branch := m.Branches.Entries[m.Branches.Selected]
	if branch.Remote {
		return func() tea.Msg {
			return BranchOperationFinishedMsg{Name: branch.Name, Err: fmt.Errorf("remote branch cannot be checked out directly: %s", branch.Name)}
		}
	}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := branches.Checkout(context.Background(), runner, branch.Name)
		return BranchOperationFinishedMsg{Name: branch.Name, Err: err}
	}
}

func (m Model) loadStashes() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := stash.List(context.Background(), r)
		return StashesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) previewSelectedStash() tea.Cmd {
	if m.Stashes.Selected < 0 || m.Stashes.Selected >= len(m.Stashes.Entries) {
		return nil
	}
	ref := m.Stashes.Entries[m.Stashes.Selected].Ref
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		result, err := stash.Show(context.Background(), runner, ref)
		return StashPreviewReadyMsg{Ref: ref, Text: string(result.Stdout), Err: err}
	}
}

func (m Model) loadHistory() tea.Cmd {
	return m.loadHistoryPage(0)
}

func (m Model) loadHistoryPage(skip int) tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := history.LoadPage(context.Background(), r, skip, 100)
		return HistoryReadyMsg{Commits: page.Commits, Skip: skip, HasMore: page.HasMore, Err: err}
	}
}

func (m Model) inspectSelectedCommit() tea.Cmd {
	if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
		return nil
	}
	sha := m.History.Rows[m.History.Selected].Commit.SHA
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		inspector, err := history.Inspect(context.Background(), runner, sha, "")
		return HistoryInspectorReadyMsg{Inspector: inspector, Err: err}
	}
}

func (m Model) loadHistoryTags() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		tags, err := history.ListTags(context.Background(), runner)
		return HistoryTagsReadyMsg{Tags: tags, Err: err}
	}
}

func (m Model) checkoutSelectedHistory() tea.Cmd {
	if m.HistoryActionTarget == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	target := m.HistoryActionTarget
	return func() tea.Msg {
		_, err := history.CheckoutCommit(context.Background(), runner, target)
		return HistoryActionFinishedMsg{Action: "checkout", Target: target, Err: err}
	}
}

func (m Model) createHistoryBranch() tea.Cmd {
	if m.HistoryBranchTarget == "" || strings.TrimSpace(m.HistoryBranchName) == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	target, name := m.HistoryBranchTarget, strings.TrimSpace(m.HistoryBranchName)
	return func() tea.Msg {
		_, err := history.CreateBranchAt(context.Background(), runner, name, target)
		return HistoryActionFinishedMsg{Action: "created branch " + name, Target: target, Err: err}
	}
}

func (m Model) revertSelectedHistory() tea.Cmd {
	if m.HistoryRevertTarget == "" || !(history.RevertConfirmation{SHA: m.HistoryRevertTarget}).Accept(m.HistoryRevertInput) {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	confirmation := history.RevertConfirmation{SHA: m.HistoryRevertTarget}
	return func() tea.Msg {
		_, err := history.Revert(context.Background(), runner, confirmation, m.HistoryRevertInput)
		return HistoryActionFinishedMsg{Action: "reverted", Target: m.HistoryRevertTarget, Err: err}
	}
}

func (m Model) loadRemotes() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	branch := m.Snapshot.Branch
	return func() tea.Msg {
		entries, err := remotes.List(context.Background(), runner)
		if err != nil {
			return RemotesReadyMsg{Err: err}
		}
		return RemotesReadyMsg{Dashboard: remotes.Dashboard{
			Remotes: entries, CurrentBranch: branch.Name, Ahead: branch.Ahead,
			Behind: branch.Behind, Now: time.Now(), StaleAfter: remoteview.DefaultStaleAfter(),
		}}
	}
}

func (m Model) fetchSelectedRemote() tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := remotes.Fetch(context.Background(), runner, remote)
		return RemoteOperationFinishedMsg{Operation: "fetch", Remote: remote, Err: err}
	}
}

func (m Model) pullSelectedRemote(strategy string) tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	branch := m.Snapshot.Branch.Name
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := remotes.Pull(context.Background(), runner, remote, branch, strategy)
		return RemoteOperationFinishedMsg{Operation: "pull " + strategy, Remote: remote, Err: err}
	}
}

func (m Model) pushSelectedRemote(forceWithLease bool) tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	branch := m.Snapshot.Branch.Name
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := remotes.Push(context.Background(), runner, remote, branch, forceWithLease)
		return RemoteOperationFinishedMsg{Operation: "push", Remote: remote, Err: err}
	}
}

func (m Model) navigate(view workspace.View, label string) tea.Cmd {
	if m.Workspace == nil {
		m.Workspace = workspace.New()
	}
	m.Workspace.Navigate(view, label)
	switch view {
	case workspace.Branches:
		return m.loadBranches()
	case workspace.Stashes:
		return m.loadStashes()
	case workspace.Log:
		return m.loadHistory()
	case workspace.Remotes:
		return m.loadRemotes()
	default:
		return nil
	}
}

func (m *Model) beginCommit() {
	files := make([]commitmodel.File, 0, len(m.Snapshot.Entries))
	for _, entry := range m.Snapshot.Entries {
		files = append(files, commitmodel.File{Path: string(entry.Path), Staged: entry.Staged})
	}
	m.Composer = commitview.New(files)
	m.Workspace.Navigate(workspace.Commit, "Commit")
}

func (m Model) commit() tea.Cmd {
	if !m.Composer.Ready() {
		return nil
	}
	draft := m.Composer.Draft
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		result, err := runner.Commit(context.Background(), git.CommitOptions{
			Message: []byte(draft.Message()), Amend: draft.Amend, NoEdit: draft.NoEdit,
			Signoff: draft.Signoff, Sign: draft.Sign, Author: draft.Author,
		})
		return CommitFinishedMsg{SHA: result.SHA, Err: err}
	}
}

func removeLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}

func (m *Model) updateComposerKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.Workspace.Back()
		return nil
	case "ctrl+s":
		if !m.Composer.Ready() {
			m.Status = strings.Join(m.Composer.Draft.Validate().Errors, "; ")
			return nil
		}
		m.State = StateOperationPending
		m.Status = "committing"
		return m.commit()
	case "tab":
		if m.Composer.Focus == "subject" {
			m.Composer.Focus = "body"
		} else {
			m.Composer.Focus = "subject"
		}
	case "enter":
		if m.Composer.Focus == "subject" {
			m.Composer.Focus = "body"
		} else {
			m.Composer.SetBody(m.Composer.Draft.Body + "\n")
		}
	case "backspace":
		if m.Composer.Focus == "subject" {
			m.Composer.SetSubject(removeLastRune(m.Composer.Draft.Subject))
		} else {
			m.Composer.SetBody(removeLastRune(m.Composer.Draft.Body))
		}
	default:
		if len([]rune(key)) != 1 || key == " " {
			if key == "space" {
				if m.Composer.Focus == "subject" {
					m.Composer.SetSubject(m.Composer.Draft.Subject + " ")
				} else {
					m.Composer.SetBody(m.Composer.Draft.Body + " ")
				}
			}
			return nil
		}
		if m.Composer.Focus == "subject" {
			m.Composer.SetSubject(m.Composer.Draft.Subject + key)
		} else {
			m.Composer.SetBody(m.Composer.Draft.Body + key)
		}
	}
	return nil
}

func (m *Model) updateHistorySearch(key string) tea.Cmd {
	if key == "esc" || key == "enter" {
		m.HistorySearching = false
		m.Status = ""
		return nil
	}
	if key == "backspace" {
		m.HistoryFilter = removeLastRune(m.HistoryFilter)
	} else if key == "space" {
		m.HistoryFilter += " "
	} else if len([]rune(key)) == 1 {
		m.HistoryFilter += key
	} else {
		return nil
	}
	m.History.SetFilter(m.HistoryFilter, m.HistoryCommits)
	m.Status = "filter: " + m.HistoryFilter
	return nil
}

func (m Model) currentView() workspace.View {
	if m.Workspace == nil {
		return workspace.Status
	}
	view, _, _, _ := m.Workspace.Snapshot()
	return view
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		if m.currentView() == workspace.Commit {
			return m, m.updateComposerKey(v.String())
		}
		if m.currentView() == workspace.Log && m.HistorySearching {
			return m, m.updateHistorySearch(v.String())
		}
		if m.currentView() == workspace.Log && m.HistoryActionConfirm {
			switch v.String() {
			case "y":
				m.HistoryActionConfirm, m.State, m.Status = false, StateOperationPending, "checking out "+m.HistoryActionTarget
				return m, m.checkoutSelectedHistory()
			case "n", "esc":
				m.HistoryActionConfirm, m.HistoryActionTarget, m.Status = false, "", "history action cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Log && m.HistoryBranchCreating {
			switch v.String() {
			case "esc":
				m.HistoryBranchCreating, m.HistoryBranchName, m.HistoryBranchTarget, m.Status = false, "", "", "branch creation cancelled"
			case "enter":
				if strings.TrimSpace(m.HistoryBranchName) == "" {
					m.Status = "branch name is required"
				} else {
					m.HistoryBranchCreating, m.State, m.Status = false, StateOperationPending, "creating branch"
					return m, m.createHistoryBranch()
				}
			case "backspace":
				m.HistoryBranchName = removeLastRune(m.HistoryBranchName)
			case "space":
				m.HistoryBranchName += " "
			default:
				if len([]rune(v.String())) == 1 {
					m.HistoryBranchName += v.String()
				}
			}
			m.Status = "branch at " + m.HistoryBranchTarget + ": " + m.HistoryBranchName
			return m, nil
		}
		if m.currentView() == workspace.Log && m.HistoryRevertConfirm {
			switch v.String() {
			case "esc":
				m.HistoryRevertConfirm, m.HistoryRevertTarget, m.HistoryRevertInput, m.HistoryRevertInvalid, m.Status = false, "", "", false, "revert cancelled"
			case "backspace":
				m.HistoryRevertInput = removeLastRune(m.HistoryRevertInput)
				m.HistoryRevertInvalid = false
			case "enter":
				if !(history.RevertConfirmation{SHA: m.HistoryRevertTarget}).Accept(m.HistoryRevertInput) {
					m.HistoryRevertInvalid = true
					m.Status = "type the exact SHA to revert"
				} else {
					m.HistoryRevertConfirm, m.HistoryRevertInvalid, m.State, m.Status = false, false, StateOperationPending, "reverting"
					return m, m.revertSelectedHistory()
				}
			default:
				if len([]rune(v.String())) == 1 {
					m.HistoryRevertInput += v.String()
				}
				m.HistoryRevertInvalid = false
			}
			if m.HistoryRevertConfirm && !m.HistoryRevertInvalid {
				m.Status = "type SHA " + m.HistoryRevertTarget + ": " + m.HistoryRevertInput
			}
			return m, nil
		}
		if m.currentView() == workspace.Stashes && v.String() == "enter" {
			m.State, m.Status = StateOperationPending, "loading stash preview"
			return m, m.previewSelectedStash()
		}
		if m.currentView() == workspace.Remotes && m.RemoteForceConfirm {
			switch v.String() {
			case "y":
				m.RemoteForceConfirm, m.State, m.Status = false, StateOperationPending, "force pushing"
				return m, m.pushSelectedRemote(true)
			case "n", "esc":
				m.RemoteForceConfirm, m.Status = false, "force push cancelled"
			}
			return m, nil
		}
		switch v.String() {
		case "q", "ctrl+c":
			m.State = StateShutdown
			return m, tea.Quit
		case "esc":
			if m.Modal != "" {
				m.Modal = ""
				m.State = StateReady
			} else if m.currentView() != workspace.Status {
				m.Workspace.Back()
			}
		case "1":
			m.Workspace.Navigate(workspace.Status, "Status")
		case "b":
			return m, m.navigate(workspace.Branches, "Branches")
		case "s":
			return m, m.navigate(workspace.Stashes, "Stashes")
		case "l":
			return m, m.navigate(workspace.Log, "History")
		case "n":
			return m, m.navigate(workspace.Remotes, "Remotes")
		case "f":
			if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "fetching"
				return m, m.fetchSelectedRemote()
			}
		case "m", "e", "o":
			if m.currentView() == workspace.Remotes {
				strategy := map[string]string{"m": "merge", "e": "rebase", "o": "ff-only"}[v.String()]
				m.State, m.Status = StateOperationPending, "pulling "+strategy
				return m, m.pullSelectedRemote(strategy)
			}
		case "p":
			if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "pushing"
				return m, m.pushSelectedRemote(false)
			}
		case "P":
			if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.RemoteForceConfirm, m.Status = true, "confirm force-with-lease push to "+remote+" for "+m.Snapshot.Branch.Name+" (y/n)"
			}
		case "]":
			if m.currentView() == workspace.Log && m.HistoryHasMore {
				m.State, m.Status = StateOperationPending, "loading more history"
				return m, m.loadHistoryPage(m.HistorySkip)
			}
		case "/":
			if m.currentView() == workspace.Log {
				m.HistorySearching, m.Status = true, "filter: "
			}
		case "x":
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryActionTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryActionConfirm = true
				m.Status = "checkout commit " + m.HistoryActionTarget + "? (y/n)"
			}
		case "B":
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryBranchTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryBranchName, m.HistoryBranchCreating = "", true
				m.Status = "branch at " + m.HistoryBranchTarget + ": enter name"
			}
		case "R":
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryRevertTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryRevertInput, m.HistoryRevertConfirm, m.HistoryRevertInvalid = "", true, false
				m.Status = "type SHA " + m.HistoryRevertTarget + ": "
			}
		case "t":
			if m.currentView() == workspace.Log {
				m.State, m.Status = StateOperationPending, "loading tags"
				return m, m.loadHistoryTags()
			}
		case "c":
			m.beginCommit()
			return m, nil
		case "enter":
			if m.currentView() == workspace.Branches {
				m.State, m.Status = StateOperationPending, "checking out"
				return m, m.checkoutSelectedBranch()
			}
			if m.currentView() == workspace.Log {
				m.State, m.Status = StateOperationPending, "loading commit details"
				return m, m.inspectSelectedCommit()
			}
		case "j", "down":
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(1)
			case workspace.Stashes:
				m.Stashes.Move(1)
			case workspace.Log:
				m.History.Move(1)
			case workspace.Remotes:
				m.Remotes.Move(1)
			default:
				m.Files.Move(1, m.Height-8)
			}
		case "k", "up":
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(-1)
			case workspace.Stashes:
				m.Stashes.Move(-1)
			case workspace.Log:
				m.History.Move(-1)
			case workspace.Remotes:
				m.Remotes.Move(-1)
			default:
				m.Files.Move(-1, m.Height-8)
			}
		case "space":
			return m, m.mutate()
		case "d":
			return m, m.openDiff()
		case "?":
			m.Modal, m.State = "help", StateModal
		case "r":
			return m, m.refresh()
		}
	case tea.MouseWheelMsg:
		if v.Button == tea.MouseWheelUp {
			m.Files.Move(-1, m.Height-8)
		} else if v.Button == tea.MouseWheelDown {
			m.Files.Move(1, m.Height-8)
		}
	case tea.MouseClickMsg:
		if v.Button == tea.MouseLeft {
			hit := uimouse.HitMap{Files: layout.Rect{X: 0, Y: 3, Width: m.Width, Height: max(1, m.Height-8)}, RowTop: 3, RowHeight: 1, StageX: 0, StageWidth: 3, RowCount: len(m.Files.Visible)}
			action, row, ok := hit.Hit(v.X, v.Y, 0)
			if ok {
				m.Files.Selected = row
				if action == uimouse.ToggleStage {
					return m, m.mutate()
				}
				if action == uimouse.SelectRow {
					return m, m.openDiff()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.Width, m.Height = v.Width, v.Height
	case SnapshotMsg:
		m.Snapshot = v.Snapshot
		m.Files.SetEntries(v.Snapshot.Entries)
		m.State = StateReady
	case RefreshStartedMsg:
		m.State = StateRefreshing
	case RefreshFinishedMsg:
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
		} else if m.State == StateRefreshing {
			m.State = StateReady
		}
	case TickMsg:
		return m, tea.Batch(m.refresh(), m.tick())
	case WatcherStateMsg:
		if v.Err != nil {
			m.Status = "watcher fallback: " + v.Err.Error()
		}
	case OperationStartedMsg:
		m.State = StateOperationPending
		m.Status = v.Name
	case OperationFinishedMsg:
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
		} else {
			m.State = StateReady
			m.Status = v.Name + " complete"
		}
	case ToastMsg:
		m.Toast = v
	case ModalMsg:
		m.Modal = v.Name
		if v.Open {
			m.State = StateModal
		} else {
			m.State = StateReady
		}
	case FocusMsg:
		m.Focus = v.Pane
	case ShutdownMsg:
		m.State = StateShutdown
	case DiffReadyMsg:
		m.DiffPath, m.DiffText, m.DiffBinary, m.Status = v.Path, v.Text, v.Binary, ""
		if v.Err != nil {
			m.Status = v.Err.Error()
		}
	case BranchesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Branches, m.State = branchview.New(v.Entries), StateReady
		}
	case StashesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Stashes, m.State = stashview.New(v.Entries), StateReady
		}
	case StashPreviewReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.StashPreview, m.StashPreviewRef, m.State, m.Status = v.Text, v.Ref, StateReady, ""
		}
	case CommitFinishedMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.State, m.Status = StateReady, "commit "+v.SHA
			m.Workspace.Back()
			return m, m.refresh()
		}
	case BranchOperationFinishedMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.State, m.Status = StateReady, "checked out "+v.Name
			return m, tea.Batch(m.refresh(), m.loadBranches())
		}
	case HistoryReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if v.Skip == 0 {
				m.HistoryCommits = append([]history.Commit(nil), v.Commits...)
			} else {
				m.HistoryCommits = append(m.HistoryCommits, v.Commits...)
			}
			if v.Skip == 0 {
				m.History = historyview.New(m.HistoryCommits)
			} else {
				m.History.SetCommits(m.HistoryCommits)
			}
			m.HistorySkip, m.HistoryHasMore, m.State = v.Skip+len(v.Commits), v.HasMore, StateReady
		}
	case HistoryInspectorReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.HistoryInspector, m.State, m.Status = v.Inspector, StateReady, ""
		}
	case HistoryTagsReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.HistoryTags, m.State, m.Status = v.Tags, StateReady, ""
		}
	case HistoryActionFinishedMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.State, m.Status = StateReady, v.Action+" "+v.Target
			m.Workspace.Back()
			return m, m.refresh()
		}
	case RemotesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Remotes, m.State = remoteview.New(v.Dashboard), StateReady
		}
	case RemoteOperationFinishedMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.State, m.Status = StateReady, v.Operation+" complete: "+v.Remote
			return m, tea.Batch(m.refresh(), m.loadRemotes())
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if view := m.currentView(); view == workspace.Branches || view == workspace.Stashes || view == workspace.Log || view == workspace.Commit || view == workspace.Remotes {
		return m.featureView(view)
	}
	name := m.Snapshot.Branch.Name
	if name == "" {
		name = "repository"
	}
	lines := []string{fmt.Sprintf("gitwatch · %s · %s", name, stateName(m.State)), fmt.Sprintf("STAGED %d  MODIFIED %d  UNTRACKED %d  CONFLICTS %d", m.Snapshot.Counts.Staged, m.Snapshot.Counts.Unstaged, m.Snapshot.Counts.Untracked, m.Snapshot.Counts.Conflicted), "──────────────────────────────────────────────────────────────"}
	rows := m.Height - 8
	if rows < 1 {
		rows = 1
	}
	for i := m.Files.Offset; i < len(m.Files.Visible) && i < m.Files.Offset+rows; i++ {
		e := m.Files.Entries[m.Files.Visible[i]]
		prefix := "  "
		if i == m.Files.Selected {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", prefix, theme.Symbol(repo.StatusLabel(e)), string(e.Path)))
	}
	if len(lines) == 3 {
		lines = append(lines, "  clean worktree")
	}
	if m.Width >= 100 && m.DiffPath != "" {
		lines = append(lines, "", "Selected diff: "+m.DiffPath)
		if m.DiffBinary {
			lines = append(lines, "  [binary file]")
		} else {
			for i, line := range strings.Split(m.DiffText, "\n") {
				if i >= max(1, m.Height-10) {
					break
				}
				lines = append(lines, "  "+line)
			}
		}
	}
	if m.Modal == "help" {
		lines = []string{"gitwatch help", "", "↑/↓ or j/k   move selection", "click row     select and open diff", "Space          stage or unstage", "Enter or d     select/inspect", "b              branches", "s              stashes", "l              history", "n              remotes", "f              fetch", "m/e/o          pull merge/rebase/ff-only", "p              push", "c              commit", "1              status", "r              refresh", "Esc            close help", "q              quit"}
	}
	lines = append(lines, "──────────────────────────────────────────────────────────────", "[j/k] move  [space] stage/unstage  [enter/d] diff  [r] refresh  [?] help  [q] quit")
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) featureView(view workspace.View) tea.View {
	title, content := "gitwatch", "Loading…"
	if view == workspace.Branches {
		title, content = "gitwatch · branches", m.Branches.View()
	} else if view == workspace.Stashes {
		title, content = "gitwatch · stashes", m.Stashes.View()
		if m.StashPreviewRef != "" {
			content += "\n\nPreview " + m.StashPreviewRef + ":\n" + m.StashPreview
		}
	} else if view == workspace.Log {
		title, content = "gitwatch · history", m.History.View()
		if m.HistorySearching {
			content = "Search: " + m.HistoryFilter + "\n\n" + content
		}
		if m.HistoryInspector.Commit.SHA != "" {
			content += "\n\n" + inspectorText(m.HistoryInspector)
		}
		if m.HistoryActionConfirm {
			content += "\n\n" + m.Status
		}
		if m.HistoryBranchCreating {
			content += "\n\nBranch name: " + m.HistoryBranchName + "\n" + m.Status
		}
		if m.HistoryRevertConfirm {
			content += "\n\nRevert confirmation: type " + m.HistoryRevertTarget + "\n" + m.HistoryRevertInput
		}
		if len(m.HistoryTags) > 0 {
			content += "\n\nTags:\n"
			for _, tag := range m.HistoryTags {
				content += "  " + tag.Name + " (" + tag.OID + ")\n"
			}
		}
	} else if view == workspace.Commit {
		title, content = "gitwatch · commit", m.Composer.View()
	} else if view == workspace.Remotes {
		title, content = "gitwatch · remotes", m.Remotes.View()
		if m.RemoteForceConfirm {
			content += "\n\n" + m.Status
		}
	}
	lines := []string{title, "", content, "", "──────────────────────────────────────────────────────────────", "[j/k] move  [1] status  [b] branches  [s] stashes  [l] history  [n] remotes  [esc] back  [q] quit"}
	if view == workspace.Log {
		lines[len(lines)-1] = "[j/k] move  [enter] inspect  [/] search  [] more  [t] tags  [x] checkout  [B] branch  [R] revert (exact SHA)  [1] status  [esc] back  [q] quit"
	}
	if view == workspace.Remotes {
		lines[len(lines)-1] = "[j/k] move  [f] fetch  [m] merge  [e] rebase  [o] ff-only  [p] push  [P] force-with-lease  [esc] back  [q] quit"
	}
	if view == workspace.Commit {
		lines[len(lines)-1] = "[tab] subject/body  [ctrl+s] commit  [esc] back  [q] quit"
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
	return v
}

func inspectorText(inspector history.Inspector) string {
	lines := []string{"Selected commit: " + inspector.Summary(), "Files:"}
	for _, stat := range inspector.Stats {
		if stat.Binary {
			lines = append(lines, "  "+stat.Path+" [binary]")
		} else {
			lines = append(lines, fmt.Sprintf("  %s +%d -%d", stat.Path, stat.Added, stat.Deleted))
		}
	}
	if inspector.Diff != "" {
		lines = append(lines, "Patch:")
		for i, line := range strings.Split(inspector.Diff, "\n") {
			if i >= 80 {
				lines = append(lines, "  …")
				break
			}
			lines = append(lines, "  "+line)
		}
	}
	return strings.Join(lines, "\n")
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func stateName(s State) string {
	switch s {
	case StateLoading:
		return "loading"
	case StateReady:
		return "ready"
	case StateRefreshing:
		return "refreshing"
	case StateOperationPending:
		return "operation pending"
	case StateError:
		return "error"
	case StateModal:
		return "modal"
	case StateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}
