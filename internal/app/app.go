package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/branches"
	"github.com/sphireinc/git-watch/internal/commands"
	"github.com/sphireinc/git-watch/internal/commitmodel"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/notifications"
	"github.com/sphireinc/git-watch/internal/operations"
	"github.com/sphireinc/git-watch/internal/patch"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/remotes"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/stash"
	"github.com/sphireinc/git-watch/internal/ui/branchview"
	"github.com/sphireinc/git-watch/internal/ui/committree"
	"github.com/sphireinc/git-watch/internal/ui/commitview"
	"github.com/sphireinc/git-watch/internal/ui/details"
	"github.com/sphireinc/git-watch/internal/ui/githubview"
	"github.com/sphireinc/git-watch/internal/ui/historyview"
	"github.com/sphireinc/git-watch/internal/ui/hunkview"
	"github.com/sphireinc/git-watch/internal/ui/layout"
	uimouse "github.com/sphireinc/git-watch/internal/ui/mouse"
	"github.com/sphireinc/git-watch/internal/ui/pluginview"
	"github.com/sphireinc/git-watch/internal/ui/rebaseview"
	"github.com/sphireinc/git-watch/internal/ui/remoteview"
	"github.com/sphireinc/git-watch/internal/ui/repoview"
	"github.com/sphireinc/git-watch/internal/ui/stashview"
	"github.com/sphireinc/git-watch/internal/ui/table"
	"github.com/sphireinc/git-watch/internal/ui/theme"
	"github.com/sphireinc/git-watch/internal/ui/worktreeview"
	"github.com/sphireinc/git-watch/internal/watch"
	"github.com/sphireinc/git-watch/internal/workspace"
	"github.com/sphireinc/git-watch/internal/worktrees"
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
type watcherStartedMsg struct {
	Generation uint64
	Manager    *watch.Manager
	Warning    error
}
type watcherEventMsg struct {
	Manager *watch.Manager
	Event   watch.Event
	Open    bool
}
type refreshResultMsg struct {
	Coordinator *git.RefreshCoordinator
	Result      git.RefreshResult
	Open        bool
}
type refreshRequestedMsg struct {
	Coordinator *git.RefreshCoordinator
	Context     context.Context
}
type OperationStartedMsg struct{ Name string }
type OperationFinishedMsg struct {
	Name       string
	Repository uint64
	Err        error
}
type RebaseFinishedMsg struct {
	Repository uint64
	Outcome    git.RebaseOutcome
	Err        error
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
	Staged     bool
	Binary     bool
	Added      int
	Deleted    int
	Request    uint64
	Err        error
	Truncated  bool
}
type CommitTreeReadyMsg struct {
	Tree       git.CommitTree
	Generation uint64
	Request    uint64
	Err        error
}
type UnpushedReadyMsg struct {
	Commits    git.UnpushedCommits
	Generation uint64
	Request    uint64
	Err        error
}
type PartialOperationFinishedMsg struct {
	Name       string
	Repository uint64
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
type StatusCommitInspectorReadyMsg struct {
	Inspector  history.Inspector
	Generation uint64
	Request    uint64
	Err        error
}
type HistoryRefReadyMsg struct {
	Ref, SHA string
	Err      error
}
type HistoryTagsReadyMsg struct {
	Tags []history.Ref
	Err  error
}
type HistoryActionFinishedMsg struct {
	Action, Target string
	Repository     uint64
	Err            error
}
type CommitFinishedMsg struct {
	SHA        string
	HookOutput string
	Repository uint64
	Err        error
}
type CommitConfigReadyMsg struct{ Config git.CommitConfig }
type BranchOperationFinishedMsg struct {
	Operation  string
	Name       string
	Repository uint64
	Err        error
}
type StashPreviewReadyMsg struct {
	Ref, Text string
	Err       error
}
type StashOperationFinishedMsg struct {
	Operation, Ref string
	Repository     uint64
	Err            error
}
type RemotesReadyMsg struct {
	Dashboard remotes.Dashboard
	Err       error
}
type WorktreesReadyMsg struct {
	Entries []worktrees.Entry
	Err     error
}
type WorktreeOperationFinishedMsg struct {
	Operation  string
	Target     string
	Repository uint64
	Err        error
}
type RepositoriesReadyMsg struct {
	Rows         []registry.Row
	Repositories []registry.Repository
	Err          error
}
type RepositoryOpenedMsg struct {
	Path           string
	Discovery      git.Discovery
	Err            error
	PersistenceErr error
}
type RemoteOperationFinishedMsg struct {
	Operation, Remote string
	Repository        uint64
	Err               error
}
type PushPreviewReadyMsg struct {
	Preview remotes.RefMovement
	Err     error
}
type GitHubReadyMsg struct {
	Repository provider.Repository
	Branch     string
	Pull       provider.PullRequest
	Checks     provider.ChecksSnapshot
	Review     provider.ReviewSnapshot
	Err        error
}
type PluginsReadyMsg struct {
	Entries []plugins.Entry
	Err     error
}
type PluginStateSavedMsg struct{ Err error }

type Model struct {
	State                    State
	Width, Height            int
	Focus, Modal, Status     string
	Motion                   Motion
	Keymap                   map[string]string
	Toast                    ToastMsg
	Notifications            *notifications.Model
	Snapshot                 repo.Snapshot
	Discovery                git.Discovery
	Files                    table.Model
	FileFilterMode           bool
	FileFilterInput          string
	FileConflictOnly         bool
	Theme                    theme.Roles
	PanelSplit               layout.Split
	DetailsCache             *details.Cache
	ActivityLog              *history.Log
	ctx                      context.Context
	cancel                   context.CancelFunc
	repositoryCtx            context.Context
	repositoryCancel         context.CancelFunc
	RefreshInterval          time.Duration
	ReconciliationInterval   time.Duration
	WatchDebounce            time.Duration
	WatchRequested           watch.RequestedMode
	WatchMode                watch.Mode
	WatchManager             *watch.Manager
	RefreshCoordinator       *git.RefreshCoordinator
	repositoryGeneration     uint64
	DiffPath, DiffText       string
	DiffBinary               bool
	DiffStaged               bool
	DiffLoading              bool
	DiffErr                  error
	DiffOffset               int
	DiffAdded                int
	DiffDeleted              int
	DiffRequest              uint64
	DiffCancel               context.CancelFunc
	DiffSearchMode           bool
	DiffSearchInput          string
	DiffSearchMatch          int
	DiffTruncated            bool
	DiffMaxBytes             int64
	DiffMaxLines             int
	CommitTreeEnabled        bool
	CommitTreeMaxCommits     int
	CommitTreeLines          []string
	CommitTreeHead           string
	CommitTreeOffset         int
	CommitTreeFocused        bool
	CommitTreeLoading        bool
	CommitTreeErr            error
	CommitTreeRequest        uint64
	CommitTreeCancel         context.CancelFunc
	LowerPane                string
	UnpushedLines            []string
	UnpushedHead             string
	UnpushedUpstream         string
	UnpushedCount            int
	UnpushedOffset           int
	UnpushedFocused          bool
	UnpushedLoading          bool
	UnpushedErr              error
	UnpushedRequest          uint64
	UnpushedCancel           context.CancelFunc
	StatusCommitActive       bool
	StatusCommitInspector    history.Inspector
	StatusCommitSHA          string
	StatusCommitSelectedLine int
	StatusCommitLoading      bool
	StatusCommitErr          error
	StatusCommitRequest      uint64
	StatusCommitCancel       context.CancelFunc
	Restore                  Confirmation
	RestoreInput             string
	HunkContext              int
	Workspace                *workspace.Model
	Branches                 branchview.Model
	BranchSearching          bool
	BranchCreateMode         bool
	BranchRenameMode         bool
	BranchRenameOld          string
	BranchUpstreamMode       bool
	BranchMutationInput      string
	BranchDeleteMode         bool
	BranchDeleteTarget       branches.Branch
	BranchDeleteForce        bool
	Stashes                  stashview.Model
	History                  historyview.Model
	Rebase                   rebaseview.Model
	HistoryCommits           []history.Commit
	HistorySkip              int
	HistoryHasMore           bool
	HistoryCancel            context.CancelFunc
	HistoryPulse             uint8
	WatchPulse               uint8
	HistoryFilter            string
	HistorySearching         bool
	HistoryInspector         history.Inspector
	HistoryInspectorParent   string
	HistoryInspectorPathMode bool
	HistoryInspectorPath     string
	HistoryRefMode           bool
	HistoryRefInput          string
	HistoryTags              []history.Ref
	HistoryActionConfirm     bool
	HistoryActionTarget      string
	HistoryBranchCreating    bool
	HistoryBranchTarget      string
	HistoryBranchName        string
	HistoryRevertConfirm     bool
	HistoryRevertTarget      string
	HistoryRevertInput       string
	HistoryRevertInvalid     bool
	Composer                 commitview.Composer
	Hunks                    hunkview.Model
	HunkDiscardConfirm       bool
	HunkDiscardInput         string
	CommitConfig             git.CommitConfig
	CommitConfigReady        bool
	CommitAmendConfirm       bool
	CommitAuthorMode         bool
	StashPreview             string
	StashPreviewRef          string
	StashCreateMode          bool
	StashCreateMessage       string
	StashIncludeUntracked    bool
	StashConfirmAction       string
	StashConfirmRef          string
	StashBranchMode          bool
	StashBranchRef           string
	StashBranchName          string
	Remotes                  remoteview.Model
	Worktrees                worktreeview.Model
	WorktreeAddMode          bool
	WorktreeAddPath          string
	WorktreeConfirmAction    string
	WorktreeConfirmTarget    string
	RemoteForceConfirm       bool
	RemotePushConfirm        bool
	RemotePushPreview        remotes.RefMovement
	RemoteSetUpstream        bool
	RemoteTag                string
	RemoteTagMode            bool
	RemoteCancel             context.CancelFunc
	RemoteJobID              string
	GitHub                   githubview.Model
	GitHubEnabled            bool
	GitHubTokenEnv           string
	GitHubCache              *provider.PullRequestCache
	GitHubChecksCache        *provider.Cache[provider.ChecksSnapshot]
	GitHubReviewsCache       *provider.Cache[provider.ReviewSnapshot]
	Plugins                  pluginview.Model
	PluginsEnabled           bool
	PluginDirectories        []string
	PluginStatePath          string
	Repositories             repoview.Model
	RepositorySearching      bool
	RepositoryRoots          []string
	RepositoryGroups         map[string][]string
	RepositoryGroup          string
	RepositoryMaxDepth       int
	RepositoryMaxCount       int
	RepositoryIgnoreDirs     []string
	RepositoryRegistry       []registry.Repository
	RepositoryRegistryPath   string
	RepositoryEngine         *registry.Engine
	OperationEngine          *operations.Engine
	PaletteMode              bool
	PaletteQuery             string
	PaletteSelected          int
	PaletteResults           []commands.Match
	PaletteActions           []commands.Action
	PaletteCommands          map[string]func() tea.Cmd
}

func New() Model {
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		State: StateLoading, Focus: "files", Motion: MotionFull,
		Keymap: config.DefaultKeymap(), GitHub: githubview.New(),
		GitHubCache:        provider.NewPullRequestCache(2 * time.Minute),
		GitHubChecksCache:  provider.NewCache[provider.ChecksSnapshot](2 * time.Minute),
		GitHubReviewsCache: provider.NewCache[provider.ReviewSnapshot](2 * time.Minute),
		Plugins:            pluginview.New(nil), Theme: theme.New(theme.Auto, false), PanelSplit: layout.DefaultSplit(),
		DetailsCache: details.NewCache(), ActivityLog: history.New(100),
		ctx: ctx, cancel: cancel, RefreshInterval: 2 * time.Second,
		ReconciliationInterval: 30 * time.Second, WatchDebounce: 75 * time.Millisecond,
		DiffMaxBytes: 4 << 20, DiffMaxLines: 20_000, CommitTreeMaxCommits: config.DefaultCommitTreeCommits,
		WatchRequested: watch.RequestedAuto, Workspace: workspace.New(),
		Notifications: notifications.New(100, false), OperationEngine: operations.New(4),
	}
}

func (m Model) paletteActions() []commands.Action {
	actions := []commands.Action{
		{ID: "status", Label: "Show status", Shortcut: "1", Enabled: m.Discovery.Root != ""},
		{ID: "branches", Label: "Open branches", Shortcut: "b", Enabled: m.Discovery.Root != ""},
		{ID: "stashes", Label: "Open stashes", Shortcut: "s", Enabled: m.Discovery.Root != ""},
		{ID: "history", Label: "Open history", Shortcut: "l", Enabled: m.Discovery.Root != ""},
		{ID: "rebase", Label: "Open interactive rebase", Shortcut: "I", Enabled: m.Discovery.Root != "" && len(m.HistoryCommits) > 0},
		{ID: "remotes", Label: "Open remotes", Shortcut: "n", Enabled: m.Discovery.Root != ""},
		{ID: "github", Label: "Open GitHub", Shortcut: "G", Enabled: m.GitHubEnabled && m.Discovery.Root != ""},
		{ID: "plugins", Label: "Open plugins", Shortcut: "E", Enabled: m.PluginsEnabled},
		{ID: "worktrees", Label: "Open worktrees", Shortcut: "w", Enabled: m.Discovery.Root != ""},
		{ID: "repositories", Label: "Open repositories", Shortcut: "v", Enabled: len(m.RepositoryRoots) > 0 || m.Discovery.Root != ""},
		{ID: "refresh", Label: "Refresh repository", Shortcut: "r", Enabled: m.Discovery.Root != ""},
		{ID: "commit_tree", Label: "Open commit tree", Shortcut: m.Keymap["commit_tree"], Enabled: m.Discovery.Root != ""},
		{ID: "unpushed", Label: "Show unpushed commits", Shortcut: m.Keymap["unpushed"], Enabled: m.Discovery.Root != ""},
		{ID: "branch_summary", Label: "Show branch summary", Shortcut: m.Keymap["branch_summary"], Enabled: m.Discovery.Root != ""},
	}
	for index, row := range m.Repositories.Rows {
		if !row.NeedsAttention() {
			continue
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("repository_attention_%d", index), Label: "Open repository attention: " + platform.SafeText(row.Repository.Name), Shortcut: "v", Enabled: true})
	}
	return append(actions, m.PaletteActions...)
}

// RegisterPaletteAction exposes provider and plugin commands without coupling
// those packages to the Bubble Tea model. The command factory runs only after
// the user selects the action.
func (m *Model) RegisterPaletteAction(action commands.Action, command func() tea.Cmd) {
	if m.PaletteCommands == nil {
		m.PaletteCommands = make(map[string]func() tea.Cmd)
	}
	m.PaletteActions = append(m.PaletteActions, action)
	m.PaletteCommands[action.ID] = command
}

func (m *Model) openPalette() {
	m.PaletteMode, m.PaletteQuery, m.PaletteSelected = true, "", 0
	m.PaletteResults = commands.Search(m.paletteActions(), "")
	m.Status = "command palette"
}

func (m *Model) updatePaletteKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.PaletteMode, m.PaletteQuery, m.PaletteResults = false, "", nil
		m.Status = "palette closed"
	case "backspace":
		m.PaletteQuery = removeLastRune(m.PaletteQuery)
	case "up", "k":
		m.PaletteSelected--
		if m.PaletteSelected < 0 {
			m.PaletteSelected = 0
		}
	case "down", "j":
		m.PaletteSelected++
		if m.PaletteSelected >= len(m.PaletteResults) {
			m.PaletteSelected = max(0, len(m.PaletteResults)-1)
		}
	case "enter":
		if m.PaletteSelected >= 0 && m.PaletteSelected < len(m.PaletteResults) && m.PaletteResults[m.PaletteSelected].Enabled {
			id := m.PaletteResults[m.PaletteSelected].ID
			m.PaletteMode, m.PaletteQuery, m.PaletteResults = false, "", nil
			return m.executePaletteAction(id)
		}
	default:
		if key == "space" {
			m.PaletteQuery += " "
		} else if len([]rune(key)) == 1 {
			m.PaletteQuery += key
		}
	}
	if m.PaletteMode {
		m.PaletteResults = commands.Search(m.paletteActions(), m.PaletteQuery)
		if m.PaletteSelected >= len(m.PaletteResults) {
			m.PaletteSelected = max(0, len(m.PaletteResults)-1)
		}
		m.Status = "command palette: " + m.PaletteQuery
	}
	return nil
}

func (m *Model) executePaletteAction(id string) tea.Cmd {
	if command := m.PaletteCommands[id]; command != nil {
		return command()
	}
	if strings.HasPrefix(id, "repository_attention_") {
		index, err := strconv.Atoi(strings.TrimPrefix(id, "repository_attention_"))
		if err == nil && index >= 0 && index < len(m.Repositories.Rows) {
			m.Repositories.Selected = index
			return m.navigate(workspace.Repositories, "Repositories")
		}
		return nil
	}
	switch id {
	case "status":
		m.Workspace.Navigate(workspace.Status, "Status")
	case "branches":
		return m.navigate(workspace.Branches, "Branches")
	case "stashes":
		return m.navigate(workspace.Stashes, "Stashes")
	case "history":
		return m.navigate(workspace.Log, "History")
	case "rebase":
		return m.openRebaseWorkspace()
	case "remotes":
		return m.navigate(workspace.Remotes, "Remotes")
	case "github":
		return m.navigate(workspace.GitHub, "GitHub")
	case "plugins":
		return m.navigate(workspace.Plugins, "Plugins")
	case "worktrees":
		return m.navigate(workspace.Worktrees, "Worktrees")
	case "repositories":
		return m.navigate(workspace.Repositories, "Repositories")
	case "refresh":
		m.State, m.Status = StateRefreshing, "refreshing"
		return m.refresh()
	case "commit_tree":
		m.Workspace.Navigate(workspace.Status, "Status")
		return m.selectLowerPane("commit-tree")
	case "unpushed":
		m.Workspace.Navigate(workspace.Status, "Status")
		return m.selectLowerPane("unpushed")
	case "branch_summary":
		m.Workspace.Navigate(workspace.Status, "Status")
		return m.selectLowerPane("branches")
	}
	return nil
}
func NewRepository(d git.Discovery) Model {
	m := New()
	if err := m.setRepository(d); err != nil {
		m.Status = err.Error()
	}
	return m
}
func NewRepositoryWithConfig(d git.Discovery, c config.Config) Model {
	m := NewRepository(d)
	m.Notifications = notifications.New(100, c.Notifications.Quiet)
	m.RefreshInterval = c.Interval
	m.ReconciliationInterval = c.Reconciliation
	m.WatchDebounce = c.Debounce
	m.DiffMaxBytes, m.DiffMaxLines = c.Diff.MaxBytes, c.Diff.MaxLines
	m.CommitTreeEnabled, m.CommitTreeMaxCommits = c.ShowCommitTree, c.CommitTree.MaxCommits
	if requested, ok := watch.ParseMode(c.Watch); ok {
		m.WatchRequested = requested
	}
	m.Keymap = mergeKeymap(config.EffectiveKeymap(c))
	m.GitHubEnabled, m.GitHubTokenEnv = c.GitHub.Enabled, c.GitHub.TokenEnv
	m.GitHubCache = provider.NewPullRequestCache(c.GitHub.CacheTTL)
	m.GitHubChecksCache = provider.NewCache[provider.ChecksSnapshot](c.GitHub.CacheTTL)
	m.GitHubReviewsCache = provider.NewCache[provider.ReviewSnapshot](c.GitHub.CacheTTL)
	m.PluginsEnabled, m.PluginDirectories = c.Plugins.Enabled, append([]string(nil), c.Plugins.Directories...)
	if path, err := plugins.StatePath(); err == nil {
		m.PluginStatePath = path
	}
	switch c.Motion {
	case "reduced":
		m.Motion = MotionReduced
	case "off":
		m.Motion = MotionOff
	default:
		m.Motion = MotionFull
	}
	m.Theme = theme.New(theme.Name(c.Theme), false)
	m.PanelSplit = layout.Split{FilesPercent: c.Layout.FilesPercent, DetailsPercent: c.Layout.DetailsPercent}
	m.RepositoryRoots = append([]string(nil), c.Repositories.Roots...)
	m.RepositoryGroups = cloneGroups(c.Repositories.Groups)
	m.RepositoryMaxDepth, m.RepositoryMaxCount = c.Repositories.MaxDepth, c.Repositories.MaxRepositories
	m.RepositoryIgnoreDirs = append([]string(nil), c.Repositories.IgnoreDirs...)
	if path, err := registry.StatePath(); err == nil {
		m.RepositoryRegistryPath = path
	}
	m.RepositoryEngine = registry.NewEngine(c.Remote.Workers)
	groupRefresh := cloneRefreshPolicies(c.Repositories.GroupRefresh)
	m.RepositoryEngine.InactiveAfterFor = func(repository registry.Repository) time.Duration {
		interval := m.RepositoryEngine.InactiveAfter
		for _, group := range repository.Groups {
			if policy, ok := groupRefresh[group]; ok && policy < interval {
				interval = policy
			}
		}
		return interval
	}
	return m
}

func (m *Model) setRepository(discovery git.Discovery) error {
	var closeErr error
	m.closeDiff()
	m.repositoryGeneration++
	if m.repositoryCancel != nil {
		m.repositoryCancel()
		m.repositoryCancel = nil
	}
	if m.HistoryCancel != nil {
		m.HistoryCancel()
		m.HistoryCancel = nil
	}
	if m.RemoteCancel != nil {
		m.RemoteCancel()
		m.RemoteCancel = nil
	}
	if m.WatchManager != nil {
		closeErr = m.WatchManager.Close()
		m.WatchManager = nil
		m.WatchMode = ""
	}
	if m.RefreshCoordinator != nil {
		m.RefreshCoordinator.Close()
	}
	if m.CommitTreeCancel != nil {
		m.CommitTreeCancel()
		m.CommitTreeCancel = nil
	}
	if m.UnpushedCancel != nil {
		m.UnpushedCancel()
		m.UnpushedCancel = nil
	}
	if m.StatusCommitCancel != nil {
		m.StatusCommitCancel()
		m.StatusCommitCancel = nil
	}
	m.Discovery = discovery
	m.LowerPane = ""
	m.CommitTreeLines, m.CommitTreeHead, m.CommitTreeOffset, m.CommitTreeErr = nil, "", 0, nil
	m.UnpushedLines, m.UnpushedHead, m.UnpushedUpstream, m.UnpushedOffset, m.UnpushedCount, m.UnpushedErr = nil, "", "", 0, 0, nil
	m.StatusCommitActive, m.StatusCommitInspector, m.StatusCommitSHA, m.StatusCommitSelectedLine, m.StatusCommitLoading, m.StatusCommitErr = false, history.Inspector{}, "", -1, false, nil
	if discovery.Root != "" {
		m.repositoryCtx, m.repositoryCancel = context.WithCancel(m.ctx)
		m.RefreshCoordinator = git.NewRefreshCoordinator(func(ctx context.Context, generation uint64) (repo.Snapshot, error) {
			return git.Snapshot(ctx, discovery, generation)
		})
	} else {
		m.repositoryCtx = nil
		m.RefreshCoordinator = nil
	}
	return closeErr
}

func (m Model) commandContext() context.Context {
	if m.repositoryCtx != nil {
		return m.repositoryCtx
	}
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m Model) acceptsRepository(generation uint64) bool {
	return generation == 0 || generation == m.repositoryGeneration
}

func (m *Model) applySnapshot(snapshot repo.Snapshot) {
	if m.ActivityLog != nil && !m.Snapshot.ObservedAt.IsZero() {
		for _, event := range history.Diff(m.Snapshot, snapshot) {
			m.ActivityLog.Add(event)
		}
	}
	if m.DiffPath != "" && !m.StatusCommitActive && !snapshotContainsPath(snapshot.Entries, m.DiffPath) {
		m.closeDiff()
	}
	m.Snapshot = snapshot
	if !m.StatusCommitActive {
		m.Files.SetEntries(snapshot.Entries)
	}
	if snapshot.Counts.Conflicted > 0 {
		m.notify(notifications.Conflict, notifications.Error, "repository conflicts", fmt.Sprintf("%d conflicted file(s)", snapshot.Counts.Conflicted), true)
	}
}

func snapshotContainsPath(entries []repo.Entry, path string) bool {
	for _, entry := range entries {
		if string(entry.Path) == path {
			return true
		}
	}
	return false
}

func (m *Model) recordActivity(kind history.Kind, path, message string) {
	if m.ActivityLog != nil {
		m.ActivityLog.Add(history.Event{At: time.Now(), Kind: kind, Path: path, Message: message})
	}
}

func cloneGroups(groups map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(groups))
	for group, paths := range groups {
		cloned[group] = append([]string(nil), paths...)
	}
	return cloned
}

func cloneRefreshPolicies(policies map[string]time.Duration) map[string]time.Duration {
	cloned := make(map[string]time.Duration, len(policies))
	for group, duration := range policies {
		cloned[group] = duration
	}
	return cloned
}

func mergeKeymap(values map[string]string) map[string]string {
	merged := config.DefaultKeymap()
	for action, binding := range values {
		merged[action] = binding
	}
	return merged
}

func (m Model) normalizeKey(input string) string {
	canonical := config.DefaultKeymap()
	for action, binding := range m.Keymap {
		if input == binding && canonical[action] != input {
			return canonical[action]
		}
	}
	return input
}
func (m Model) Init() tea.Cmd {
	if m.Discovery.Root == "" {
		return nil
	}
	commands := []tea.Cmd{m.refresh(), m.tick(), m.startWatcher()}
	if tree := m.loadCommitTreeAtInit(); tree != nil {
		commands = append(commands, tree)
	}
	if m.RefreshCoordinator != nil {
		commands = append(commands, waitForRefresh(m.RefreshCoordinator))
	}
	return tea.Batch(commands...)
}

func (m Model) contextPaneEnabled() bool { return m.CommitTreeEnabled || m.LowerPane != "" }

func (m Model) showCommitTreePane() bool {
	return m.LowerPane == "" && m.CommitTreeEnabled || m.LowerPane == "commit-tree"
}

func (m Model) showUnpushedPane() bool { return m.LowerPane == "unpushed" }

func (m Model) showBranchSummaryPane() bool { return m.LowerPane == "branches" }

func (m Model) contextPaneFocused() bool {
	return m.showBranchSummaryPane() || m.CommitTreeFocused || m.UnpushedFocused
}

func (m *Model) selectLowerPane(name string) tea.Cmd {
	if name == "commit-tree" && !m.CommitTreeEnabled {
		m.CommitTreeEnabled = true
	}
	m.LowerPane = name
	m.CommitTreeFocused, m.UnpushedFocused = name == "commit-tree", name == "unpushed"
	switch name {
	case "commit-tree":
		if m.StatusCommitSelectedLine < 0 {
			m.StatusCommitSelectedLine = 0
		}
		m.Status = "commit tree focused"
		return m.refreshCommitTreeIfNeeded()
	case "unpushed":
		m.Status = "unpushed commits focused"
		return m.loadUnpushed()
	case "branches":
		m.Status = "branch summary focused"
		return m.loadBranches()
	default:
		m.CommitTreeFocused, m.UnpushedFocused = false, false
		m.Status = "status files focused"
		return nil
	}
}

func (m *Model) loadUnpushed() tea.Cmd {
	if m.Discovery.Root == "" || m.LowerPane != "unpushed" {
		return nil
	}
	if m.UnpushedCancel != nil {
		m.UnpushedCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.UnpushedCancel = cancel
	m.UnpushedRequest++
	request, generation, limit, upstream := m.UnpushedRequest, m.repositoryGeneration, git.DefaultUnpushedCommits, m.Snapshot.Branch.Upstream
	runner := git.NewRunner(m.Discovery.Root)
	m.UnpushedLoading, m.UnpushedErr = true, nil
	return func() tea.Msg {
		commits, err := git.LoadUnpushed(ctx, runner, upstream, limit)
		return UnpushedReadyMsg{Commits: commits, Generation: generation, Request: request, Err: err}
	}
}

func (m *Model) refreshUnpushedIfNeeded() tea.Cmd {
	if !m.showUnpushedPane() || m.UnpushedLoading {
		return nil
	}
	if m.Snapshot.Branch.Upstream == "" && m.UnpushedRequest > 0 && m.UnpushedUpstream == "" {
		return nil
	}
	if m.Snapshot.Branch.OID == m.UnpushedHead && m.Snapshot.Branch.Upstream == m.UnpushedUpstream && m.UnpushedHead != "" {
		return nil
	}
	return m.loadUnpushed()
}

func (m *Model) refreshStatusContextIfNeeded() tea.Cmd {
	return tea.Batch(m.refreshCommitTreeIfNeeded(), m.refreshUnpushedIfNeeded(), m.refreshBranchSummaryIfNeeded())
}

func (m *Model) refreshBranchSummaryIfNeeded() tea.Cmd {
	if !m.showBranchSummaryPane() || m.Discovery.Root == "" {
		return nil
	}
	return m.loadBranches()
}

// loadCommitTreeAtInit starts the first tree load without mutating a value
// receiver. Subsequent loads run through loadCommitTree and advance the live
// model's request counter in Update.
func (m Model) loadCommitTreeAtInit() tea.Cmd {
	if !m.CommitTreeEnabled || m.Discovery.Root == "" {
		return nil
	}
	ctx := m.commandContext()
	runner := git.NewRunner(m.Discovery.Root)
	generation, request, limit := m.repositoryGeneration, m.CommitTreeRequest, m.CommitTreeMaxCommits
	return func() tea.Msg {
		tree, err := git.LoadCommitTree(ctx, runner, limit)
		return CommitTreeReadyMsg{Tree: tree, Generation: generation, Request: request, Err: err}
	}
}

func (m Model) tick() tea.Cmd {
	if !m.Motion.Ticks() {
		return nil
	}
	interval := m.RefreshInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return TickMsg{At: t} })
}
func (m Model) refresh() tea.Cmd {
	coordinator, ctx := m.RefreshCoordinator, m.repositoryCtx
	if ctx == nil {
		ctx = m.ctx
	}
	if coordinator != nil {
		return func() tea.Msg {
			return refreshRequestedMsg{Coordinator: coordinator, Context: ctx}
		}
	}
	discovery := m.Discovery
	return func() tea.Msg {
		snapshot, err := git.Snapshot(ctx, discovery, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Snapshot: snapshot}
	}
}

func (m *Model) loadCommitTree() tea.Cmd {
	if !m.CommitTreeEnabled || m.Discovery.Root == "" {
		return nil
	}
	if m.CommitTreeCancel != nil {
		m.CommitTreeCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.CommitTreeCancel = cancel
	m.CommitTreeRequest++
	request, generation, limit := m.CommitTreeRequest, m.repositoryGeneration, m.CommitTreeMaxCommits
	runner := git.NewRunner(m.Discovery.Root)
	m.CommitTreeLoading, m.CommitTreeErr = true, nil
	return func() tea.Msg {
		tree, err := git.LoadCommitTree(ctx, runner, limit)
		return CommitTreeReadyMsg{Tree: tree, Generation: generation, Request: request, Err: err}
	}
}

func (m *Model) refreshCommitTreeIfNeeded() tea.Cmd {
	if !m.CommitTreeEnabled || m.CommitTreeLoading || m.Snapshot.Branch.OID == m.CommitTreeHead && m.CommitTreeHead != "" {
		return nil
	}
	return m.loadCommitTree()
}

func waitForRefresh(coordinator *git.RefreshCoordinator) tea.Cmd {
	return func() tea.Msg {
		select {
		case result := <-coordinator.Results():
			return refreshResultMsg{Coordinator: coordinator, Result: result, Open: true}
		case <-coordinator.Done():
			return refreshResultMsg{Coordinator: coordinator}
		}
	}
}

func (m Model) startWatcher() tea.Cmd {
	root, ctx, generation := m.Discovery.Root, m.repositoryCtx, m.repositoryGeneration
	if ctx == nil {
		ctx = m.ctx
	}
	requested := m.WatchRequested
	interval, reconciliation, debounce := m.RefreshInterval, m.ReconciliationInterval, m.WatchDebounce
	metadata := []string{m.Discovery.GitDir, m.Discovery.CommonDir}
	return func() tea.Msg {
		manager, warning := watch.StartWithMetadata(ctx, root, metadata, requested, interval, reconciliation, debounce)
		if manager == nil {
			var fallbackErr error
			manager, fallbackErr = watch.StartWithMetadata(ctx, root, metadata, watch.RequestedPoll, interval, reconciliation, debounce)
			warning = errors.Join(warning, fallbackErr)
		}
		return watcherStartedMsg{Generation: generation, Manager: manager, Warning: warning}
	}
}

func waitForWatcher(manager *watch.Manager) tea.Cmd {
	return func() tea.Msg {
		event, open := <-manager.Events()
		return watcherEventMsg{Manager: manager, Event: event, Open: open}
	}
}

func (m Model) mutate() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	generation := m.repositoryGeneration
	if e.Conflicted {
		return func() tea.Msg {
			return OperationFinishedMsg{Name: "stage", Repository: generation, Err: fmt.Errorf("conflicted path requires external resolution")}
		}
	}
	r, path, ctx := git.NewRunner(m.Discovery.Root), append([]byte(nil), e.Path...), m.commandContext()
	return func() tea.Msg {
		var err error
		if e.Staged && !e.Unstaged {
			_, err = r.Unstage(ctx, path)
		} else {
			_, err = r.Stage(ctx, path)
		}
		if err != nil {
			return OperationFinishedMsg{Name: "path operation", Repository: generation, Err: err}
		}
		return OperationFinishedMsg{Name: "path operation", Repository: generation}
	}
}

func (m Model) mutateAll(stage bool) tea.Cmd {
	runner, ctx, generation := git.NewRunner(m.Discovery.Root), m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		var err error
		name := "stage all (tracked, untracked, and deletions)"
		if stage {
			_, err = runner.StageAll(ctx)
		} else {
			name = "unstage all (working-tree changes preserved)"
			_, err = runner.UnstageAll(ctx)
		}
		return OperationFinishedMsg{Name: name, Repository: generation, Err: err}
	}
}

func (m *Model) beginRestore() {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return
	}
	entry := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	if entry.Untracked {
		m.Status = "untracked deletion is intentionally unavailable"
		return
	}
	if entry.Conflicted {
		m.Status = "resolve conflicted paths externally, then stage them"
		return
	}
	m.Restore = RestoreConfirmation(string(entry.Path), entry.Staged, entry.Unstaged)
	m.RestoreInput = ""
	m.Status = m.Restore.Prompt + " Type yes to confirm."
}

func (m Model) restoreSelected() tea.Cmd {
	confirmation, ctx, generation := m.Restore, m.commandContext(), m.repositoryGeneration
	if !confirmation.Accept(m.RestoreInput) {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := runner.Restore(ctx, []byte(confirmation.Path), strings.Contains(confirmation.Scope, "staged"), true)
		return OperationFinishedMsg{Name: "restore " + confirmation.Path, Repository: generation, Err: err}
	}
}

func (m *Model) updateRestoreKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.Restore, m.RestoreInput, m.Status = Confirmation{}, "", "restore cancelled"
	case "backspace":
		m.RestoreInput = removeLastRune(m.RestoreInput)
	case "enter":
		if !m.Restore.Accept(m.RestoreInput) {
			m.Status = "type yes to confirm restore of " + m.Restore.Path
			return nil
		}
		command := m.restoreSelected()
		m.Restore, m.RestoreInput = Confirmation{}, ""
		m.State, m.Status = StateOperationPending, "restoring selected path"
		return command
	default:
		if len([]rune(key)) == 1 {
			m.RestoreInput += key
		}
	}
	if m.Restore.Open {
		m.Status = m.Restore.Prompt + " Type yes: " + m.RestoreInput
	}
	return nil
}

func (m *Model) updateFileFilterKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.FileFilterMode, m.FileFilterInput, m.FileConflictOnly = false, "", false
		m.Files.SetFilter("")
		m.Status = "file filter cleared"
		return nil
	case "enter":
		m.FileFilterMode = false
		m.Status = "file filter: " + m.FileFilterInput
		if m.DiffPath != "" {
			return m.openDiff()
		}
		return nil
	case "backspace":
		m.FileFilterInput = removeLastRune(m.FileFilterInput)
	case "space":
		m.FileFilterInput += " "
	default:
		if len([]rune(key)) != 1 {
			return nil
		}
		m.FileFilterInput += key
	}
	m.FileConflictOnly = false
	m.Files.SetFilter(m.FileFilterInput)
	m.Status = "file filter: " + m.FileFilterInput
	return nil
}

func (m *Model) updateDiffSearchKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.DiffSearchMode, m.DiffSearchInput = false, ""
		m.Status = "diff search cancelled"
	case "backspace":
		m.DiffSearchInput = removeLastRune(m.DiffSearchInput)
	case "enter":
		m.DiffSearchMode = false
		if !m.seekDiffMatch(0) {
			m.Status = "diff search: no matches"
		} else {
			m.Status = "diff search: " + m.DiffSearchInput
		}
	case "space":
		m.DiffSearchInput += " "
	default:
		if len([]rune(key)) == 1 {
			m.DiffSearchInput += key
		}
	}
	if m.DiffSearchMode {
		m.Status = "diff search: " + m.DiffSearchInput
	}
	return nil
}

func (m *Model) seekDiffMatch(start int) bool {
	query := strings.ToLower(m.DiffSearchInput)
	if query == "" || m.DiffText == "" {
		return false
	}
	lines := strings.Split(m.DiffText, "\n")
	if start < 0 {
		start = 0
	}
	for offset := 0; offset < len(lines); offset++ {
		index := (start + offset) % len(lines)
		if strings.Contains(strings.ToLower(lines[index]), query) {
			m.DiffSearchMatch = index
			m.DiffOffset = index
			return true
		}
	}
	return false
}

func (m *Model) openDiff() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	return m.openDiffMode(e.Staged && !e.Unstaged)
}

func (m *Model) openDiffMode(staged bool) tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	path := append([]byte(nil), e.Path...)
	r := git.NewRunner(m.Discovery.Root)
	if m.DiffCancel != nil {
		m.DiffCancel()
	}
	base := m.repositoryCtx
	if base == nil {
		base = m.ctx
	}
	loadCtx, cancel := context.WithCancel(base)
	m.DiffCancel = cancel
	m.DiffRequest++
	request := m.DiffRequest
	m.DiffPath, m.DiffText = string(path), ""
	m.DiffBinary, m.DiffStaged, m.DiffLoading, m.DiffErr, m.DiffOffset, m.DiffAdded, m.DiffDeleted, m.DiffTruncated = false, staged, true, nil, 0, 0, 0, false
	m.Status = "loading diff for " + string(path)
	contextLines := m.HunkContext
	if contextLines <= 0 {
		contextLines = 3
	}
	if m.StatusCommitActive {
		sha, parent := m.StatusCommitSHA, m.StatusCommitInspector.Parent
		return func() tea.Msg {
			inspector, err := history.InspectPath(loadCtx, r, sha, parent, string(path))
			text, truncated := limitDiffText(inspector.Diff, m.DiffMaxBytes, m.DiffMaxLines)
			added, deleted := diffStat(text)
			return DiffReadyMsg{Path: string(path), Text: text, Staged: false, Binary: false, Added: added, Deleted: deleted, Request: request, Err: err, Truncated: truncated}
		}
	}
	return func() tea.Msg {
		d, err := r.DiffWithContext(loadCtx, path, staged, contextLines)
		text, truncated := limitDiffText(string(d.Text), m.DiffMaxBytes, m.DiffMaxLines)
		added, deleted := diffStat(text)
		return DiffReadyMsg{Path: string(path), Text: text, Staged: d.Staged, Binary: d.Binary, Added: added, Deleted: deleted, Request: request, Err: err, Truncated: truncated}
	}
}

func (m *Model) inspectStatusCommit(line int) tea.Cmd {
	if !m.showCommitTreePane() || line < 0 || line >= len(m.CommitTreeLines) {
		return nil
	}
	plain := committree.Plain(m.CommitTreeLines[line])
	short := firstCommitToken(plain)
	if short == "" {
		m.Status = "no commit at selected tree row"
		return nil
	}
	if m.StatusCommitCancel != nil {
		m.StatusCommitCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.StatusCommitCancel = cancel
	m.StatusCommitRequest++
	request, generation := m.StatusCommitRequest, m.repositoryGeneration
	m.StatusCommitSelectedLine, m.StatusCommitLoading, m.StatusCommitErr = line, true, nil
	m.Status = "loading commit " + short
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		sha, err := history.ResolveRef(ctx, runner, short)
		if err == nil {
			commit, metadataErr := history.LoadCommit(ctx, runner, sha)
			if metadataErr != nil {
				return StatusCommitInspectorReadyMsg{Generation: generation, Request: request, Err: metadataErr}
			}
			parent := ""
			if len(commit.Parents) > 0 {
				parent = commit.Parents[0]
			}
			inspector, inspectErr := history.InspectPath(ctx, runner, sha, parent, "")
			if inspectErr != nil {
				err = inspectErr
			} else {
				inspector.Commit = commit
				return StatusCommitInspectorReadyMsg{Inspector: inspector, Generation: generation, Request: request}
			}
		}
		return StatusCommitInspectorReadyMsg{Generation: generation, Request: request, Err: err}
	}
}

func firstCommitToken(line string) string {
	for _, token := range strings.Fields(line) {
		token = strings.Trim(token, "|*\\/()[],")
		if len(token) < 7 || len(token) > 40 {
			continue
		}
		valid := true
		for _, r := range token {
			if !isHexRune(r) {
				valid = false
				break
			}
		}
		if valid {
			return token
		}
	}
	return ""
}

func isHexRune(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
}

func (m *Model) clearStatusCommitInspection() {
	if m.StatusCommitCancel != nil {
		m.StatusCommitCancel()
		m.StatusCommitCancel = nil
	}
	m.StatusCommitActive, m.StatusCommitInspector, m.StatusCommitSHA, m.StatusCommitSelectedLine, m.StatusCommitLoading, m.StatusCommitErr = false, history.Inspector{}, "", -1, false, nil
	m.Files.SetEntries(m.Snapshot.Entries)
	m.closeDiff()
}

func limitDiffText(text string, maxBytes int64, maxLines int) (string, bool) {
	if maxBytes <= 0 {
		maxBytes = 4 << 20
	}
	if maxLines <= 0 {
		maxLines = 20_000
	}
	truncated := false
	if int64(len(text)) > maxBytes {
		text = text[:maxBytes]
		truncated = true
	}
	lines := strings.Split(text, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}
	return strings.Join(lines, "\n"), truncated
}

func diffStat(text string) (added, deleted int) {
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			added++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			deleted++
		}
	}
	return added, deleted
}

func (m *Model) closeDiff() {
	if m.DiffCancel != nil {
		m.DiffCancel()
		m.DiffCancel = nil
	}
	m.DiffRequest++
	m.DiffPath, m.DiffText = "", ""
	m.DiffBinary, m.DiffStaged, m.DiffLoading, m.DiffErr, m.DiffOffset, m.DiffAdded, m.DiffDeleted, m.DiffTruncated = false, false, false, nil, 0, 0, 0, false
}

func (m *Model) beginHunks() {
	files, err := patch.Parse(m.DiffText)
	if err != nil || len(files) == 0 {
		m.Status = "selected diff is not a patch"
		return
	}
	m.Hunks = hunkview.New(files)
	m.Workspace.Navigate(workspace.Hunks, "Hunks")
	m.Status = "hunk selection"
}

func (m Model) applySelectedHunks(discard bool) tea.Cmd {
	generation := m.repositoryGeneration
	if len(m.Hunks.Files) == 0 || m.Hunks.Selection.Count() == 0 {
		return nil
	}
	data, err := m.Hunks.Selection.BuildPatch(m.Hunks.Files)
	if err != nil {
		return func() tea.Msg {
			return PartialOperationFinishedMsg{Name: "partial patch", Repository: generation, Err: err}
		}
	}
	runner := git.NewRunner(m.Discovery.Root)
	staged, ctx := m.DiffStaged, m.commandContext()
	return func() tea.Msg {
		var operationErr error
		if discard {
			_, operationErr = runner.ApplyReversePatch(ctx, git.PartialPatch{Patch: data})
		} else if staged {
			_, operationErr = runner.ApplyReverseCachedPatch(ctx, git.PartialPatch{Patch: data})
		} else {
			_, operationErr = runner.ApplyCachedPatch(ctx, git.PartialPatch{Patch: data})
		}
		return PartialOperationFinishedMsg{Name: map[bool]string{true: "discard", false: "partial stage"}[discard], Repository: generation, Err: operationErr}
	}
}

func (m Model) loadBranches() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		worktreeEntries, err := worktrees.List(m.commandContext(), r)
		if err != nil {
			return BranchesReadyMsg{Err: err}
		}
		entries, err := branches.ListWithOccupancy(m.commandContext(), r, worktrees.Occupancy(worktreeEntries))
		return BranchesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) checkoutSelectedBranch() tea.Cmd {
	if m.Branches.Selected < 0 || m.Branches.Selected >= len(m.Branches.Entries) {
		return nil
	}
	branch := m.Branches.Entries[m.Branches.Selected]
	generation := m.repositoryGeneration
	if branch.Remote {
		return func() tea.Msg {
			return BranchOperationFinishedMsg{Name: branch.Name, Repository: generation, Err: fmt.Errorf("remote branch cannot be checked out directly: %s", branch.Name)}
		}
	}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := branches.Checkout(m.commandContext(), runner, branch.Name)
		return BranchOperationFinishedMsg{Operation: "checked out", Name: branch.Name, Repository: generation, Err: err}
	}
}

func (m Model) branchMutation(operation, name string, work func(context.Context, git.Runner) error) tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		return BranchOperationFinishedMsg{Operation: operation, Name: name, Repository: generation, Err: work(ctx, runner)}
	}
}

func (m Model) createBranch(name string) tea.Cmd {
	return m.branchMutation("created", name, func(ctx context.Context, r git.Runner) error {
		_, err := branches.Create(ctx, r, name)
		return err
	})
}

func (m Model) renameBranch(oldName, newName string) tea.Cmd {
	return m.branchMutation("renamed", newName, func(ctx context.Context, r git.Runner) error {
		_, err := branches.Rename(ctx, r, oldName, newName)
		return err
	})
}

func (m Model) setBranchUpstream(local, upstream string) tea.Cmd {
	return m.branchMutation("set upstream", local, func(ctx context.Context, r git.Runner) error {
		_, err := branches.SetUpstream(ctx, r, local, upstream)
		return err
	})
}

func (m Model) unsetBranchUpstream(local string) tea.Cmd {
	return m.branchMutation("unset upstream", local, func(ctx context.Context, r git.Runner) error {
		_, err := branches.UnsetUpstream(ctx, r, local)
		return err
	})
}

func (m Model) deleteBranch(branch branches.Branch, force bool, input string) tea.Cmd {
	return m.branchMutation("deleted", branch.Name, func(ctx context.Context, r git.Runner) error {
		_, err := branches.Delete(ctx, r, branch, branches.DeletePrompt(branch.Name, force), input)
		return err
	})
}

func (m *Model) updateBranchMutationKey(key string) tea.Cmd {
	if !m.BranchCreateMode && !m.BranchRenameMode && !m.BranchUpstreamMode && !m.BranchDeleteMode {
		return nil
	}
	if key == "esc" {
		m.BranchCreateMode, m.BranchRenameMode, m.BranchUpstreamMode, m.BranchDeleteMode = false, false, false, false
		m.BranchMutationInput, m.BranchRenameOld = "", ""
		m.Status = "branch action cancelled"
		return nil
	}
	if key == "backspace" {
		m.BranchMutationInput = removeLastRune(m.BranchMutationInput)
	} else if key == "space" {
		m.BranchMutationInput += " "
	} else if key == "enter" {
		input := strings.TrimSpace(m.BranchMutationInput)
		if input == "" {
			m.Status = "branch name is required"
			return nil
		}
		renameMode, upstreamMode := m.BranchRenameMode, m.BranchUpstreamMode
		m.BranchCreateMode, m.BranchRenameMode, m.BranchUpstreamMode, m.BranchDeleteMode = false, false, false, false
		m.State = StateOperationPending
		switch {
		case renameMode:
			old := m.BranchRenameOld
			m.BranchRenameOld, m.BranchMutationInput, m.Status = "", "", "renaming branch"
			return m.renameBranch(old, input)
		case m.BranchDeleteTarget.Name != "":
			branch, force := m.BranchDeleteTarget, m.BranchDeleteForce
			m.BranchDeleteTarget, m.BranchMutationInput, m.Status = branches.Branch{}, "", "deleting branch"
			return m.deleteBranch(branch, force, input)
		case upstreamMode:
			local := m.Branches.Entries[m.Branches.Selected].Name
			m.BranchMutationInput, m.Status = "", "setting upstream"
			return m.setBranchUpstream(local, input)
		default:
			m.BranchMutationInput, m.Status = "", "creating branch"
			return m.createBranch(input)
		}
	} else if len([]rune(key)) == 1 {
		m.BranchMutationInput += key
	} else {
		return nil
	}
	label := "branch name"
	if m.BranchRenameOld != "" {
		label = "rename " + m.BranchRenameOld + " to"
	} else if m.BranchUpstreamMode {
		label = "upstream"
	} else if m.BranchDeleteTarget.Name != "" {
		label = "type " + m.BranchDeleteTarget.Name + " to confirm"
	}
	m.Status = label + ": " + m.BranchMutationInput
	return nil
}

func (m Model) loadStashes() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := stash.List(m.commandContext(), r)
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
		result, err := stash.Show(m.commandContext(), runner, ref)
		return StashPreviewReadyMsg{Ref: ref, Text: string(result.Stdout), Err: err}
	}
}

func (m Model) createStash() tea.Cmd {
	message := strings.TrimSpace(m.StashCreateMessage)
	if message == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := stash.CreateWithOptions(ctx, runner, message, m.StashIncludeUntracked)
		return StashOperationFinishedMsg{Operation: "created stash", Ref: message, Repository: generation, Err: err}
	}
}

func (m Model) executeStashAction() tea.Cmd {
	if m.StashConfirmRef == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	action, ref := m.StashConfirmAction, m.StashConfirmRef
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		var err error
		switch action {
		case "apply":
			_, err = stash.ApplyChecked(ctx, runner, ref)
		case "pop":
			_, err = stash.PopChecked(ctx, runner, ref)
		case "drop":
			_, err = stash.Drop(ctx, runner, ref)
		default:
			err = fmt.Errorf("unknown stash action: %s", action)
		}
		return StashOperationFinishedMsg{Operation: action, Ref: ref, Repository: generation, Err: err}
	}
}

func (m Model) createStashBranch() tea.Cmd {
	if strings.TrimSpace(m.StashBranchName) == "" || m.StashBranchRef == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	name, ref := strings.TrimSpace(m.StashBranchName), m.StashBranchRef
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := stash.BranchChecked(ctx, runner, name, ref)
		return StashOperationFinishedMsg{Operation: "created branch " + name, Ref: ref, Repository: generation, Err: err}
	}
}

func (m *Model) updateStashCreateKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.StashCreateMode, m.StashCreateMessage, m.Status = false, "", "stash creation cancelled"
	case "backspace":
		m.StashCreateMessage = removeLastRune(m.StashCreateMessage)
	case "enter":
		if strings.TrimSpace(m.StashCreateMessage) == "" {
			m.Status = "stash message is required"
		} else {
			m.StashCreateMode, m.State, m.Status = false, StateOperationPending, "creating stash"
			return m.createStash()
		}
	case "space":
		m.StashCreateMessage += " "
	case "u":
		m.StashIncludeUntracked = !m.StashIncludeUntracked
	default:
		if len([]rune(key)) == 1 {
			m.StashCreateMessage += key
		}
	}
	if m.StashCreateMode {
		m.Status = "stash message: " + m.StashCreateMessage
	}
	return nil
}

func (m *Model) loadHistory() tea.Cmd {
	return m.loadHistoryPage(0)
}

func (m *Model) loadHistoryPage(skip int) tea.Cmd {
	if m.HistoryCancel != nil {
		m.HistoryCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.HistoryCancel = cancel
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := history.LoadPage(ctx, r, skip, 100)
		return HistoryReadyMsg{Commits: page.Commits, Skip: skip, HasMore: page.HasMore, Err: err}
	}
}

func (m Model) inspectSelectedCommit() tea.Cmd {
	if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
		return nil
	}
	commit := m.History.Rows[m.History.Selected].Commit
	return m.inspectCommit(commit, m.HistoryInspectorParent, m.HistoryInspectorPath)
}

func (m Model) inspectCommit(commit history.Commit, parent, path string) tea.Cmd {
	sha := commit.SHA
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		inspector, err := history.InspectPath(m.commandContext(), runner, sha, parent, path)
		inspector.Commit = commit
		return HistoryInspectorReadyMsg{Inspector: inspector, Err: err}
	}
}

func (m Model) loadHistoryTags() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		tags, err := history.ListTags(m.commandContext(), runner)
		return HistoryTagsReadyMsg{Tags: tags, Err: err}
	}
}

func (m Model) resolveHistoryRef() tea.Cmd {
	ref := strings.TrimSpace(m.HistoryRefInput)
	if ref == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		sha, err := history.ResolveRef(m.commandContext(), runner, ref)
		return HistoryRefReadyMsg{Ref: ref, SHA: sha, Err: err}
	}
}

func (m Model) checkoutSelectedHistory() tea.Cmd {
	if m.HistoryActionTarget == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	target, ctx, generation := m.HistoryActionTarget, m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := history.CheckoutCommit(ctx, runner, target)
		return HistoryActionFinishedMsg{Action: "checkout", Target: target, Repository: generation, Err: err}
	}
}

func (m Model) createHistoryBranch() tea.Cmd {
	if m.HistoryBranchTarget == "" || strings.TrimSpace(m.HistoryBranchName) == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	target, name := m.HistoryBranchTarget, strings.TrimSpace(m.HistoryBranchName)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := history.CreateBranchAt(ctx, runner, name, target)
		return HistoryActionFinishedMsg{Action: "created branch " + name, Target: target, Repository: generation, Err: err}
	}
}

func (m Model) revertSelectedHistory() tea.Cmd {
	if m.HistoryRevertTarget == "" || !(history.RevertConfirmation{SHA: m.HistoryRevertTarget}).Accept(m.HistoryRevertInput) {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	confirmation := history.RevertConfirmation{SHA: m.HistoryRevertTarget}
	target, input, ctx, generation := m.HistoryRevertTarget, m.HistoryRevertInput, m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := history.Revert(ctx, runner, confirmation, input)
		return HistoryActionFinishedMsg{Action: "reverted", Target: target, Repository: generation, Err: err}
	}
}

func (m Model) loadRemotes() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	branch := m.Snapshot.Branch
	activity := append([]remotes.Activity(nil), m.Remotes.Dashboard.Activity...)
	return func() tea.Msg {
		entries, err := remotes.List(m.commandContext(), runner)
		if err != nil {
			return RemotesReadyMsg{Err: err}
		}
		return RemotesReadyMsg{Dashboard: remotes.Dashboard{
			Remotes: entries, CurrentBranch: branch.Name, Ahead: branch.Ahead,
			Behind: branch.Behind, Activity: activity, Now: time.Now(), StaleAfter: remoteview.DefaultStaleAfter(),
		}}
	}
}

func (m Model) loadGitHub() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	branch := m.Snapshot.Branch.Name
	tokenEnv := m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	return func() tea.Msg {
		entries, err := remotes.List(m.commandContext(), runner)
		if err != nil {
			return GitHubReadyMsg{Branch: branch, Err: err}
		}
		var repository provider.Repository
		for _, remote := range entries {
			if candidate, ok := provider.ParseGitHubRemote(remote.FetchURL); ok {
				repository = candidate
				break
			}
		}
		if repository.Owner == "" {
			return GitHubReadyMsg{Branch: branch, Err: fmt.Errorf("no GitHub remote detected")}
		}
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		cache := m.GitHubCache
		if cache == nil {
			cache = provider.NewPullRequestCache(2 * time.Minute)
		}
		pull, err := cache.Get(m.commandContext(), client, repository, branch)
		if err != nil {
			return GitHubReadyMsg{Repository: repository, Branch: branch, Err: err}
		}
		checksCache := m.GitHubChecksCache
		if checksCache == nil {
			checksCache = provider.NewCache[provider.ChecksSnapshot](2 * time.Minute)
		}
		checks, err := checksCache.Get(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"@"+branch, func(ctx context.Context) (provider.ChecksSnapshot, error) {
			return client.Checks(ctx, repository, branch)
		})
		if err != nil {
			return GitHubReadyMsg{Repository: repository, Branch: branch, Pull: pull, Err: err}
		}
		reviewsCache := m.GitHubReviewsCache
		if reviewsCache == nil {
			reviewsCache = provider.NewCache[provider.ReviewSnapshot](2 * time.Minute)
		}
		review, err := reviewsCache.Get(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"#"+fmt.Sprint(pull.Number), func(ctx context.Context) (provider.ReviewSnapshot, error) {
			return client.Reviews(ctx, repository, pull.Number)
		})
		return GitHubReadyMsg{Repository: repository, Branch: branch, Pull: pull, Checks: checks, Review: review, Err: err}
	}
}

func (m Model) loadPlugins() tea.Cmd {
	directories := append([]string(nil), m.PluginDirectories...)
	statePath := m.PluginStatePath
	return func() tea.Msg {
		entries, err := plugins.Discover(m.commandContext(), directories, 128)
		if err == nil && statePath != "" {
			state, stateErr := plugins.LoadState(statePath)
			if stateErr != nil {
				return PluginsReadyMsg{Err: stateErr}
			}
			entries = plugins.ApplyState(entries, state)
		}
		return PluginsReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) savePluginState(entries []plugins.Entry) tea.Cmd {
	path := m.PluginStatePath
	if path == "" {
		return nil
	}
	return func() tea.Msg {
		return PluginStateSavedMsg{Err: plugins.SaveState(path, entries)}
	}
}

func (m *Model) recordRemoteActivity(operation, message string, success bool) {
	activity := append(m.Remotes.Dashboard.Activity, remotes.Activity{At: time.Now(), Operation: operation, Message: message, Success: success})
	if len(activity) > 50 {
		activity = activity[len(activity)-50:]
	}
	m.Remotes.Dashboard.Activity = activity
}

func (m Model) loadWorktrees() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := worktrees.List(m.commandContext(), runner)
		return WorktreesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) loadRepositories() tea.Cmd {
	roots := append([]string(nil), m.RepositoryRoots...)
	if len(roots) == 0 && m.Discovery.Root != "" {
		roots = []string{m.Discovery.Root}
	}
	engine := m.RepositoryEngine
	if engine == nil {
		engine = registry.NewEngine(2)
	}
	statePath := m.RepositoryRegistryPath
	groups := cloneGroups(m.RepositoryGroups)
	return func() tea.Msg {
		repositories, err := registry.Discover(m.commandContext(), roots, registry.Options{MaxDepth: m.RepositoryMaxDepth, MaxRepositories: m.RepositoryMaxCount, IgnoreDirs: m.RepositoryIgnoreDirs})
		if err != nil {
			return RepositoriesReadyMsg{Err: err}
		}
		var stored []registry.Repository
		if statePath != "" {
			stored, err = registry.Load(statePath)
			if err != nil {
				return RepositoriesReadyMsg{Err: err}
			}
		}
		repositories = registry.Merge(repositories, stored, groups)
		if m.RepositoryGroup != "" {
			repositories = registry.InGroup(repositories, m.RepositoryGroup)
		}
		if statePath != "" {
			if err := registry.Save(statePath, repositories); err != nil {
				return RepositoriesReadyMsg{Err: err}
			}
		}
		results := engine.Refresh(m.commandContext(), repositories, m.Discovery.Root)
		return RepositoriesReadyMsg{Rows: registry.Rows(results), Repositories: repositories}
	}
}

func (m Model) openSelectedRepository() tea.Cmd {
	if m.Repositories.Selected < 0 || m.Repositories.Selected >= len(m.Repositories.Rows) {
		return nil
	}
	path := m.Repositories.Rows[m.Repositories.Selected].Repository.Path
	registryEntries := append([]registry.Repository(nil), m.RepositoryRegistry...)
	registryPath := m.RepositoryRegistryPath
	return func() tea.Msg {
		discovery, err := git.Discover(m.commandContext(), path)
		var persistenceErr error
		if err == nil && registryPath != "" {
			now := time.Now()
			for i := range registryEntries {
				if registryEntries[i].Path == path {
					registryEntries[i].LastOpened = now
				}
			}
			persistenceErr = registry.Save(registryPath, registryEntries)
		}
		return RepositoryOpenedMsg{Path: path, Discovery: discovery, Err: err, PersistenceErr: persistenceErr}
	}
}

func (m Model) openSelectedWorktree() tea.Cmd {
	if m.Worktrees.Selected < 0 || m.Worktrees.Selected >= len(m.Worktrees.Entries) {
		return nil
	}
	path := m.Worktrees.Entries[m.Worktrees.Selected].Path
	return func() tea.Msg {
		discovery, err := git.Discover(m.commandContext(), path)
		return RepositoryOpenedMsg{Path: path, Discovery: discovery, Err: err}
	}
}

func (m Model) addWorktree() tea.Cmd {
	path := strings.TrimSpace(m.WorktreeAddPath)
	if path == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := worktrees.Add(ctx, runner, path, "")
		return WorktreeOperationFinishedMsg{Operation: "added worktree", Target: path, Repository: generation, Err: err}
	}
}

func (m Model) executeWorktreeAction() tea.Cmd {
	target := m.WorktreeConfirmTarget
	if target == "" {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	action := m.WorktreeConfirmAction
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		var err error
		switch action {
		case "remove":
			_, err = worktrees.Remove(ctx, runner, target, false)
		case "prune":
			_, err = worktrees.Prune(ctx, runner, false)
		default:
			err = fmt.Errorf("unknown worktree action: %s", action)
		}
		return WorktreeOperationFinishedMsg{Operation: action, Target: target, Repository: generation, Err: err}
	}
}

func (m *Model) updateWorktreeAddKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.WorktreeAddMode, m.WorktreeAddPath, m.Status = false, "", "worktree creation cancelled"
	case "backspace":
		m.WorktreeAddPath = removeLastRune(m.WorktreeAddPath)
	case "enter":
		if strings.TrimSpace(m.WorktreeAddPath) == "" {
			m.Status = "worktree path is required"
		} else {
			m.WorktreeAddMode, m.State, m.Status = false, StateOperationPending, "adding worktree"
			return m.addWorktree()
		}
	case "space":
		m.WorktreeAddPath += " "
	default:
		if len([]rune(key)) == 1 {
			m.WorktreeAddPath += key
		}
	}
	if m.WorktreeAddMode {
		m.Status = "worktree path: " + m.WorktreeAddPath
	}
	return nil
}

func (m *Model) startRemoteJob(operation, remote string) context.Context {
	base := m.repositoryCtx
	if base == nil {
		base = m.commandContext()
	}
	ctx, cancel := context.WithCancel(base)
	m.RemoteCancel = cancel
	m.RemoteJobID = fmt.Sprintf("remote-%d", time.Now().UnixNano())
	now := time.Now()
	m.Remotes.Dashboard.Jobs = append(m.Remotes.Dashboard.Jobs, remotes.Job{ID: m.RemoteJobID, Operation: operation, Remote: remote, State: remotes.JobRunning, Progress: "starting", Started: now, Updated: now})
	return ctx
}

func (m *Model) remoteCommand(ctx context.Context, operation, remote string, work operations.Work) tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	id, repoRoot, generation := m.RemoteJobID, m.Discovery.Root, m.repositoryGeneration
	command := m.OperationEngine.Command(ctx, id, repoRoot, operation, 5*time.Minute, work)
	return func() tea.Msg {
		result := command()
		return RemoteOperationFinishedMsg{Operation: operation, Remote: remote, Repository: generation, Err: result.Result.Err}
	}
}

func (m *Model) fetchSelectedRemote() tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.startRemoteJob("fetch", remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = "fetching remote refs"
	return m.remoteCommand(ctx, "fetch", remote, func(ctx context.Context) error {
		_, err := remotes.Fetch(ctx, runner, remote)
		return err
	})
}

func (m *Model) pullSelectedRemote(strategy string) tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	branch := m.Snapshot.Branch.Name
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.startRemoteJob("pull "+strategy, remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = "integrating " + strategy
	return m.remoteCommand(ctx, "pull "+strategy, remote, func(ctx context.Context) error {
		_, err := remotes.Pull(ctx, runner, remote, branch, strategy)
		return err
	})
}

func (m *Model) pushSelectedRemote(forceWithLease bool) tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	branch := m.Snapshot.Branch.Name
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.startRemoteJob("push", remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = "sending refs"
	setUpstream := m.RemoteSetUpstream
	return m.remoteCommand(ctx, "push", remote, func(ctx context.Context) error {
		_, err := remotes.PushWithOptions(ctx, runner, remote, branch, remotes.PushOptions{ForceWithLease: forceWithLease, SetUpstream: setUpstream})
		return err
	})
}

func (m *Model) pushSelectedTag() tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) || strings.TrimSpace(m.RemoteTag) == "" {
		return nil
	}
	remote, tag := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name, strings.TrimSpace(m.RemoteTag)
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.startRemoteJob("push tag", remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = "sending tag"
	return m.remoteCommand(ctx, "push tag "+tag, remote, func(ctx context.Context) error {
		_, err := remotes.PushTag(ctx, runner, remote, tag)
		return err
	})
}

func (m Model) previewSelectedRemotePush() tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) || m.Snapshot.Branch.Name == "" {
		return nil
	}
	remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
	branch := m.Snapshot.Branch.Name
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		preview, err := remotes.PreviewPush(m.commandContext(), runner, remote, branch)
		return PushPreviewReadyMsg{Preview: preview, Err: err}
	}
}

func (m *Model) navigate(view workspace.View, label string) tea.Cmd {
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
	case workspace.GitHub:
		return m.loadGitHub()
	case workspace.Plugins:
		return m.loadPlugins()
	case workspace.Worktrees:
		return m.loadWorktrees()
	case workspace.Repositories:
		return m.loadRepositories()
	default:
		return nil
	}
}

func (m *Model) openRebaseWorkspace() tea.Cmd {
	choices := make([]rebaseview.Base, 0, 2)
	if m.Snapshot.Branch.Upstream != "" {
		choices = append(choices, rebaseview.Base{Label: "upstream", Ref: m.Snapshot.Branch.Upstream})
	}
	if m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
		parents := m.History.Rows[m.History.Selected].Commit.Parents
		if len(parents) > 0 {
			choices = append(choices, rebaseview.Base{Label: "selected commit parent", Ref: parents[0]})
		}
	}
	if len(choices) == 0 {
		m.Status = "interactive rebase requires an explicit base"
		return nil
	}
	view, err := rebaseview.New(m.Snapshot.Branch.Name, m.Snapshot.Branch.Upstream, choices, m.HistoryCommits)
	if err != nil {
		m.Status = "rebase plan: " + err.Error()
		return nil
	}
	for _, commit := range m.HistoryCommits {
		for _, ref := range commit.Refs {
			if strings.Contains(ref, "origin/") {
				view.ReachableRemote = true
			}
		}
	}
	view.Published = view.ReachableRemote
	m.Rebase = view
	m.Workspace.Navigate(workspace.Rebase, "Interactive rebase")
	return nil
}

func (m *Model) startRebase() tea.Cmd {
	if !m.Rebase.StartEnabled() {
		m.Status = "rebase plan is loading, invalid, or has no explicit base"
		return nil
	}
	request := git.RebaseRequest{Base: m.Rebase.Base.Ref, Plan: m.Rebase.Plan}
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		outcome, err := runner.StartInteractiveRebase(m.commandContext(), request)
		return RebaseFinishedMsg{Repository: generation, Outcome: outcome, Err: err}
	}
}

func (m *Model) beginCommit() tea.Cmd {
	files := make([]commitmodel.File, 0, len(m.Snapshot.Entries))
	for _, entry := range m.Snapshot.Entries {
		files = append(files, commitmodel.File{Path: string(entry.Path), Staged: entry.Staged})
	}
	m.Composer = commitview.New(files)
	m.CommitConfig, m.CommitConfigReady = git.CommitConfig{}, false
	m.Workspace.Navigate(workspace.Commit, "Commit")
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg { return CommitConfigReadyMsg{Config: runner.CommitConfig(m.commandContext())} }
}

func (m Model) commit() tea.Cmd {
	if !m.Composer.Ready() {
		return nil
	}
	draft := m.Composer.Draft
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		result, err := runner.Commit(ctx, git.CommitOptions{
			Message: []byte(draft.Message()), Amend: draft.Amend, NoEdit: draft.NoEdit,
			Signoff: draft.Signoff, Sign: draft.Sign, Author: draft.Author,
		})
		return CommitFinishedMsg{SHA: result.SHA, HookOutput: platform.SafeText(string(append(result.Result.Stdout, result.Result.Stderr...))), Repository: generation, Err: err}
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
	if m.CommitAuthorMode {
		switch key {
		case "esc":
			m.CommitAuthorMode = false
		case "enter":
			m.CommitAuthorMode = false
		case "backspace":
			m.Composer.Draft.Author = removeLastRune(m.Composer.Draft.Author)
		case "space":
			m.Composer.Draft.Author += " "
		default:
			if len([]rune(key)) == 1 && key != "\n" && key != "\r" {
				m.Composer.Draft.Author += key
			}
		}
		return nil
	}
	switch key {
	case "esc":
		m.Workspace.Back()
		return nil
	case "ctrl+s":
		if !m.Composer.Ready() {
			m.Status = strings.Join(m.Composer.Draft.Validate().Errors, "; ")
			return nil
		}
		if m.Composer.Draft.Amend {
			m.CommitAmendConfirm = true
			m.Status = "amend rewrites the current commit; press y to confirm or n to cancel"
			return nil
		}
		m.State = StateOperationPending
		m.Status = "committing"
		return m.commit()
	case "A":
		m.Composer.Draft.Amend = !m.Composer.Draft.Amend
		if !m.Composer.Draft.Amend {
			m.Composer.Draft.NoEdit = false
		}
	case "N":
		m.Composer.Draft.NoEdit = !m.Composer.Draft.NoEdit
	case "o":
		m.Composer.Draft.Signoff = !m.Composer.Draft.Signoff
	case "S":
		m.Composer.Draft.Sign = !m.Composer.Draft.Sign
	case "@":
		m.CommitAuthorMode = true
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

func (m *Model) updateHunkKey(key string) tea.Cmd {
	if m.HunkDiscardConfirm {
		switch key {
		case "esc", "n":
			m.HunkDiscardConfirm, m.HunkDiscardInput = false, ""
			m.Status = "partial discard cancelled"
		case "backspace":
			m.HunkDiscardInput = removeLastRune(m.HunkDiscardInput)
		case "enter":
			if m.HunkDiscardInput == "discard" {
				m.HunkDiscardConfirm, m.HunkDiscardInput, m.State = false, "", StateOperationPending
				m.Status = "discarding selected hunks"
				return m.applySelectedHunks(true)
			}
			m.Status = "type discard to confirm"
		case "space":
			m.HunkDiscardInput += " "
		default:
			if len([]rune(key)) == 1 {
				m.HunkDiscardInput += key
			}
		}
		return nil
	}
	switch key {
	case "esc":
		m.Workspace.Back()
	case "j", "down":
		m.Hunks.Move(1)
	case "k", "up":
		m.Hunks.Move(-1)
	case "n", "]":
		m.Hunks.MoveHunk(1)
	case "p", "[":
		m.Hunks.MoveHunk(-1)
	case "N":
		m.Hunks.MoveFile(1)
	case "P":
		m.Hunks.MoveFile(-1)
	case "c":
		switch m.HunkContext {
		case 0, 3:
			m.HunkContext = 8
		case 8:
			m.HunkContext = 20
		default:
			m.HunkContext = 3
		}
		m.Status = fmt.Sprintf("loading %d context lines", m.HunkContext)
		return m.openDiff()
	case "space":
		m.Hunks.Toggle()
	case "a":
		if m.Hunks.File < len(m.Hunks.Files) && m.Hunks.Hunk < len(m.Hunks.Files[m.Hunks.File].Hunks) {
			m.Hunks.Selection.SelectHunk(m.Hunks.File, m.Hunks.Hunk, m.Hunks.Files[m.Hunks.File].Hunks[m.Hunks.Hunk])
		}
	case "A":
		m.Hunks.Selection.SelectAll(m.Hunks.Files)
	case "i":
		m.Hunks.Selection.Invert(m.Hunks.Files)
	case "s":
		m.State, m.Status = StateOperationPending, "applying selected hunks"
		return m.applySelectedHunks(false)
	case "d":
		if m.DiffStaged {
			m.Status = "discard is available only for working-tree hunks"
		} else if m.Hunks.Selection.Count() == 0 {
			m.Status = "select hunks before discarding"
		} else {
			m.HunkDiscardConfirm, m.HunkDiscardInput = true, ""
			m.Status = "type discard to confirm partial discard"
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

func (m *Model) updateRepositorySearch(key string) tea.Cmd {
	if key == "esc" || key == "enter" {
		m.RepositorySearching = false
		m.Status = ""
		return nil
	}
	switch key {
	case "backspace":
		m.Repositories.SetFilter(removeLastRune(m.Repositories.Query))
	case "space":
		m.Repositories.SetFilter(m.Repositories.Query + " ")
	default:
		if len([]rune(key)) != 1 {
			return nil
		}
		m.Repositories.SetFilter(m.Repositories.Query + key)
	}
	m.Status = "repository filter: " + m.Repositories.Query
	return nil
}

func (m *Model) updateBranchSearch(key string) tea.Cmd {
	if key == "esc" || key == "enter" {
		m.BranchSearching = false
		m.Status = ""
		return nil
	}
	switch key {
	case "backspace":
		m.Branches.SetFilter(removeLastRune(m.Branches.Query))
	case "space":
		m.Branches.SetFilter(m.Branches.Query + " ")
	default:
		if len([]rune(key)) != 1 {
			return nil
		}
		m.Branches.SetFilter(m.Branches.Query + key)
	}
	m.Status = "branch filter: " + m.Branches.Query
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
		if normalized := m.normalizeKey(v.String()); normalized != v.String() && len([]rune(normalized)) > 0 {
			v = tea.KeyPressMsg(tea.Key{Text: normalized, Code: []rune(normalized)[0]})
		}
		if m.PaletteMode {
			return m, m.updatePaletteKey(v.String())
		}
		if m.currentView() == workspace.Status && m.Restore.Open {
			return m, m.updateRestoreKey(v.String())
		}
		if m.currentView() == workspace.Status && m.FileFilterMode {
			return m, m.updateFileFilterKey(v.String())
		}
		if m.currentView() == workspace.Status && m.DiffSearchMode {
			return m, m.updateDiffSearchKey(v.String())
		}
		if m.currentView() == workspace.Hunks {
			return m, m.updateHunkKey(v.String())
		}
		if m.currentView() == workspace.Commit && m.CommitAmendConfirm {
			switch v.String() {
			case "y", "Y":
				m.CommitAmendConfirm = false
				m.State, m.Status = StateOperationPending, "committing amended commit"
				return m, m.commit()
			case "n", "N", "esc":
				m.CommitAmendConfirm = false
				m.Status = "amend cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Commit {
			if v.Mod&tea.ModCtrl != 0 && v.String() == "s" {
				return m, m.updateComposerKey("ctrl+s")
			}
			return m, m.updateComposerKey(v.String())
		}
		if m.currentView() == workspace.Log && m.HistorySearching {
			return m, m.updateHistorySearch(v.String())
		}
		if m.currentView() == workspace.Repositories && m.RepositorySearching {
			return m, m.updateRepositorySearch(v.String())
		}
		if m.currentView() == workspace.Branches && m.BranchSearching {
			return m, m.updateBranchSearch(v.String())
		}
		if m.currentView() == workspace.Branches && (m.BranchCreateMode || m.BranchRenameMode || m.BranchUpstreamMode || m.BranchDeleteMode) {
			return m, m.updateBranchMutationKey(v.String())
		}
		if m.currentView() == workspace.Log && m.HistoryInspectorPathMode {
			switch v.String() {
			case "esc":
				m.HistoryInspectorPathMode, m.HistoryInspectorPath = false, ""
				m.Status = "path filter cancelled"
			case "backspace":
				m.HistoryInspectorPath = removeLastRune(m.HistoryInspectorPath)
			case "space":
				m.HistoryInspectorPath += " "
			case "enter":
				m.HistoryInspectorPathMode, m.State, m.Status = false, StateOperationPending, "loading filtered commit details"
				return m, m.inspectSelectedCommit()
			default:
				if len([]rune(v.String())) == 1 {
					m.HistoryInspectorPath += v.String()
				}
			}
			if m.HistoryInspectorPathMode {
				m.Status = "path filter: " + m.HistoryInspectorPath
			}
			return m, nil
		}
		if m.currentView() == workspace.Log && m.HistoryRefMode {
			switch v.String() {
			case "esc":
				m.HistoryRefMode, m.HistoryRefInput = false, ""
				m.Status = "ref jump cancelled"
			case "backspace":
				m.HistoryRefInput = removeLastRune(m.HistoryRefInput)
			case "space":
				m.HistoryRefInput += " "
			case "enter":
				if strings.TrimSpace(m.HistoryRefInput) == "" {
					m.Status = "ref is required"
				} else {
					m.HistoryRefMode, m.State, m.Status = false, StateOperationPending, "resolving ref"
					return m, m.resolveHistoryRef()
				}
			default:
				if len([]rune(v.String())) == 1 {
					m.HistoryRefInput += v.String()
				}
			}
			if m.HistoryRefMode {
				m.Status = "jump to ref: " + m.HistoryRefInput
			}
			return m, nil
		}
		if m.currentView() == workspace.Stashes && m.StashCreateMode {
			return m, m.updateStashCreateKey(v.String())
		}
		if m.currentView() == workspace.Worktrees && m.WorktreeAddMode {
			return m, m.updateWorktreeAddKey(v.String())
		}
		if m.currentView() == workspace.Remotes && m.RemoteTagMode {
			switch v.String() {
			case "esc":
				m.RemoteTagMode, m.RemoteTag = false, ""
				m.Status = "tag push cancelled"
			case "backspace":
				m.RemoteTag = removeLastRune(m.RemoteTag)
			case "enter":
				if strings.TrimSpace(m.RemoteTag) == "" {
					m.Status = "tag name is required"
				} else {
					m.RemoteTagMode, m.RemotePushConfirm = false, true
					m.Status = "confirm push tag " + strings.TrimSpace(m.RemoteTag) + "? (y/n)"
				}
			case "space":
				m.RemoteTag += " "
			default:
				if len([]rune(v.String())) == 1 {
					m.RemoteTag += v.String()
				}
			}
			if m.RemoteTagMode {
				m.Status = "tag name: " + m.RemoteTag
			}
			return m, nil
		}
		if m.currentView() == workspace.Stashes && m.StashBranchMode {
			switch v.String() {
			case "esc":
				m.StashBranchMode, m.StashBranchName, m.StashBranchRef, m.Status = false, "", "", "stash branch cancelled"
			case "backspace":
				m.StashBranchName = removeLastRune(m.StashBranchName)
			case "enter":
				if strings.TrimSpace(m.StashBranchName) == "" {
					m.Status = "branch name is required"
				} else {
					m.StashBranchMode, m.State, m.Status = false, StateOperationPending, "creating stash branch"
					return m, m.createStashBranch()
				}
			case "space":
				m.StashBranchName += " "
			default:
				if len([]rune(v.String())) == 1 {
					m.StashBranchName += v.String()
				}
			}
			if m.StashBranchMode {
				m.Status = "branch from " + m.StashBranchRef + ": " + m.StashBranchName
			}
			return m, nil
		}
		if m.currentView() == workspace.Stashes && m.StashConfirmAction != "" {
			switch v.String() {
			case "y":
				m.State, m.Status = StateOperationPending, m.StashConfirmAction+" stash"
				action := m.executeStashAction()
				m.StashConfirmAction, m.StashConfirmRef = "", ""
				return m, action
			case "n", "esc":
				m.StashConfirmAction, m.StashConfirmRef, m.Status = "", "", "stash action cancelled"
			}
			return m, nil
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
		if m.currentView() == workspace.Remotes && m.RemotePushConfirm {
			switch v.String() {
			case "y":
				m.RemotePushConfirm, m.State, m.Status = false, StateOperationPending, "pushing"
				if m.RemoteTag != "" {
					return m, m.pushSelectedTag()
				}
				return m, m.pushSelectedRemote(false)
			case "n", "esc":
				m.RemotePushConfirm, m.RemoteSetUpstream, m.RemoteTag, m.Status = false, false, "", "push cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Worktrees && m.WorktreeConfirmAction != "" {
			switch v.String() {
			case "y":
				m.State, m.Status = StateOperationPending, m.WorktreeConfirmAction+" worktree"
				action := m.executeWorktreeAction()
				m.WorktreeConfirmAction, m.WorktreeConfirmTarget = "", ""
				return m, action
			case "n", "esc":
				m.WorktreeConfirmAction, m.WorktreeConfirmTarget, m.Status = "", "", "worktree action cancelled"
			}
			return m, nil
		}
		switch v.String() {
		case "q", "ctrl+c":
			m.State = StateShutdown
			if err := m.shutdown(); err != nil {
				m.Status = "shutdown: " + err.Error()
			}
			return m, tea.Quit
		case "ctrl+p":
			m.openPalette()
			return m, nil
		case "esc":
			if m.RemoteCancel != nil && m.State == StateOperationPending {
				m.RemoteCancel()
				m.RemoteCancel = nil
				m.Status = "remote operation cancellation requested"
				return m, nil
			}
			if m.Modal != "" {
				m.Modal = ""
				m.State = StateReady
			} else if m.currentView() == workspace.Status && m.DiffPath != "" {
				m.closeDiff()
				m.Status = "diff closed"
			} else if m.currentView() == workspace.Status && m.StatusCommitActive {
				m.clearStatusCommitInspection()
				m.Status = "returned to worktree status"
			} else if m.currentView() != workspace.Status {
				if m.currentView() == workspace.Log && m.HistoryCancel != nil {
					m.HistoryCancel()
					m.HistoryCancel = nil
				}
				m.Workspace.Back()
			}
		case "1":
			if m.StatusCommitActive {
				m.clearStatusCommitInspection()
			}
			m.Workspace.Navigate(workspace.Status, "Status")
		case "b":
			if m.currentView() == workspace.Rebase {
				m.Rebase.BaseMode = !m.Rebase.BaseMode
				return m, nil
			}
			if m.currentView() == workspace.Status {
				return m, m.navigate(workspace.Branches, "Branches")
			}
			return m, m.navigate(workspace.Branches, "Branches")
		case "s":
			if m.currentView() == workspace.Repositories {
				m.Status = "repository sort: " + string(m.Repositories.CycleSort())
				return m, nil
			}
			if m.currentView() == workspace.Branches {
				m.Branches.CycleSort()
				m.Status = "branch sort: " + m.Branches.SortLabel()
				return m, nil
			}
			return m, m.navigate(workspace.Stashes, "Stashes")
		case "l":
			return m, m.navigate(workspace.Log, "History")
		case "n":
			if m.currentView() == workspace.Status && m.DiffPath != "" && m.DiffSearchInput != "" {
				if !m.seekDiffMatch(m.DiffSearchMatch + 1) {
					m.Status = "diff search: no matches"
				}
				return m, nil
			}
			return m, m.navigate(workspace.Remotes, "Remotes")
		case "G":
			if m.GitHubEnabled {
				return m, m.navigate(workspace.GitHub, "GitHub")
			}
		case "E":
			if m.PluginsEnabled {
				return m, m.navigate(workspace.Plugins, "Plugins")
			}
		case "w":
			return m, m.navigate(workspace.Worktrees, "Worktrees")
		case "v":
			return m, m.navigate(workspace.Repositories, "Repositories")
		case "A":
			if m.currentView() == workspace.Worktrees {
				m.WorktreeAddMode, m.WorktreeAddPath = true, ""
				m.Status = "worktree path: "
			}
		case "U":
			if m.currentView() == workspace.Status {
				m.State, m.Status = StateOperationPending, "unstaging all changes; working-tree content will be preserved"
				return m, m.mutateAll(false)
			}
		case "S":
			if m.currentView() == workspace.Status {
				m.Files.CycleSort()
				m.Status = "file sort mode changed"
			}
		case "V":
			if m.currentView() == workspace.Status && m.DiffPath != "" {
				return m, m.openDiffMode(!m.DiffStaged)
			}
		case "!":
			if m.currentView() == workspace.Status {
				m.FileConflictOnly = !m.FileConflictOnly
				if m.FileConflictOnly {
					m.Files.SetConflictFilter(true)
					m.Status = "showing conflicted files only"
				} else {
					m.Files.SetFilter(m.FileFilterInput)
					m.Status = "conflict filter cleared"
				}
			}
		case "D":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Current || branch.OccupiedPath != "" {
					m.Status = "cannot delete checked-out or worktree-bound branch"
				} else {
					m.BranchDeleteMode, m.BranchDeleteTarget, m.BranchDeleteForce, m.BranchMutationInput = true, branch, false, ""
					m.Status = "type " + branch.Name + " to confirm deletion: "
				}
			} else if m.currentView() == workspace.Worktrees && m.Worktrees.Selected >= 0 && m.Worktrees.Selected < len(m.Worktrees.Entries) {
				path := m.Worktrees.Entries[m.Worktrees.Selected].Path
				m.WorktreeConfirmAction, m.WorktreeConfirmTarget = "remove", path
				m.Status = "confirm remove worktree " + path + "? (y/n)"
			} else if m.currentView() == workspace.Stashes && m.Stashes.Selected >= 0 && m.Stashes.Selected < len(m.Stashes.Entries) {
				ref := m.Stashes.Entries[m.Stashes.Selected].Ref
				m.StashConfirmAction, m.StashConfirmRef = "drop", ref
				m.Status = "confirm drop " + ref + "? (y/n)"
			}
		case "f":
			if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "fetching"
				return m, m.fetchSelectedRemote()
			} else if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				m.HistoryInspectorPathMode, m.HistoryInspectorPath = true, ""
				m.Status = "path filter: "
			}
		case "m", "e", "o":
			if m.currentView() == workspace.GitHub && v.String() == "o" {
				if command, err := platform.OpenURLCommand(m.GitHub.Pull.URL); err == nil {
					m.Status = "opening pull request"
					return m, tea.ExecProcess(command, nil)
				} else {
					m.Status = err.Error()
				}
				break
			}
			if m.currentView() == workspace.Remotes {
				strategy := map[string]string{"m": "merge", "e": "rebase", "o": "ff-only"}[v.String()]
				m.State, m.Status = StateOperationPending, "pulling "+strategy
				return m, m.pullSelectedRemote(strategy)
			}
		case "p":
			if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "preparing push preview"
				return m, m.previewSelectedRemotePush()
			}
			if m.currentView() == workspace.Stashes && m.Stashes.Selected >= 0 && m.Stashes.Selected < len(m.Stashes.Entries) {
				ref := m.Stashes.Entries[m.Stashes.Selected].Ref
				m.StashConfirmAction, m.StashConfirmRef = "pop", ref
				m.Status = "confirm pop " + ref + "? (y/n)"
			}
		case "u":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.Status = "select a local branch"
				} else {
					m.BranchUpstreamMode, m.BranchMutationInput = true, ""
					m.Status = "upstream for " + branch.Name + ": "
				}
			} else if m.currentView() == workspace.Remotes {
				m.RemoteSetUpstream, m.RemotePushConfirm = true, true
				m.Status = "confirm push with upstream tracking to selected remote? (y/n)"
			}
		case "T":
			if m.currentView() == workspace.Status {
				return m, m.selectLowerPane("commit-tree")
			} else if m.currentView() == workspace.Remotes {
				m.RemoteTagMode, m.RemoteTag = true, ""
				m.Status = "tag name: "
			}
		case "P":
			if m.currentView() == workspace.Status {
				return m, m.selectLowerPane("unpushed")
			} else if m.currentView() == workspace.Worktrees {
				m.WorktreeConfirmAction, m.WorktreeConfirmTarget = "prune", "repository"
				m.Status = "confirm prune stale worktrees? (y/n)"
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.RemoteForceConfirm, m.Status = true, "confirm force-with-lease push to "+remote+" for "+m.Snapshot.Branch.Name+" (y/n)"
			}
		case "N":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Upstream == "" {
					m.Status = "branch has no upstream"
				} else {
					m.State, m.Status = StateOperationPending, "unsetting upstream"
					return m, m.unsetBranchUpstream(branch.Name)
				}
			}
		case "X":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Current || branch.OccupiedPath != "" {
					m.Status = "cannot delete checked-out or worktree-bound branch"
				} else {
					m.BranchDeleteMode, m.BranchDeleteTarget, m.BranchDeleteForce, m.BranchMutationInput = true, branch, true, ""
					m.Status = "type " + branch.Name + " to confirm FORCE deletion: "
				}
			}
		case "]":
			if m.currentView() == workspace.Log && m.HistoryHasMore {
				m.State, m.Status = StateOperationPending, "loading more history"
				return m, m.loadHistoryPage(m.HistorySkip)
			}
		case "/":
			if m.currentView() == workspace.Status && m.DiffPath != "" {
				m.DiffSearchMode, m.DiffSearchInput = true, ""
				m.Status = "diff search: "
				return m, nil
			}
			if m.currentView() == workspace.Log {
				m.HistorySearching, m.Status = true, "filter: "
			}
			if m.currentView() == workspace.Repositories {
				m.RepositorySearching, m.Status = true, "repository filter: "
			}
			if m.currentView() == workspace.Branches {
				m.BranchSearching, m.Status = true, "branch filter: "
			}
			if m.currentView() == workspace.Status {
				m.FileFilterMode, m.FileConflictOnly = true, false
				m.Files.SetFilter(m.FileFilterInput)
				m.Status = "file filter: " + m.FileFilterInput
			}
		case "x":
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryActionTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryActionConfirm = true
				m.Status = "checkout commit " + m.HistoryActionTarget + "? (y/n)"
			}
		case "ctrl+n":
			if m.Notifications != nil {
				active := m.Notifications.Active()
				if len(active) > 0 {
					latest := active[len(active)-1]
					if m.Notifications.Dismiss(latest.ID) {
						m.Status = "dismissed notification"
					}
				}
			}
		case "B":
			if m.currentView() == workspace.Status {
				return m, m.selectLowerPane("branches")
			}
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryBranchTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryBranchName, m.HistoryBranchCreating = "", true
				m.Status = "branch at " + m.HistoryBranchTarget + ": enter name"
			}
			if m.currentView() == workspace.Stashes && m.Stashes.Selected >= 0 && m.Stashes.Selected < len(m.Stashes.Entries) {
				m.StashBranchRef, m.StashBranchName, m.StashBranchMode = m.Stashes.Entries[m.Stashes.Selected].Ref, "", true
				m.Status = "branch from " + m.StashBranchRef + ": enter name"
			}
		case "R":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.Status = "remote branch cannot be renamed here"
				} else {
					m.BranchRenameMode, m.BranchRenameOld, m.BranchMutationInput = true, branch.Name, ""
					m.Status = "rename " + branch.Name + " to: "
				}
			} else if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryRevertTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryRevertInput, m.HistoryRevertConfirm, m.HistoryRevertInvalid = "", true, false
				m.Status = "type SHA " + m.HistoryRevertTarget + ": "
			} else if m.currentView() == workspace.Status {
				m.beginRestore()
			}
		case "I":
			if m.currentView() == workspace.Log {
				return m, m.openRebaseWorkspace()
			}
		case "M":
			if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" && len(m.HistoryInspector.Commit.Parents) > 0 {
				parent := ""
				for i, candidate := range m.HistoryInspector.Commit.Parents {
					if candidate == m.HistoryInspectorParent {
						parent = m.HistoryInspector.Commit.Parents[(i+1)%len(m.HistoryInspector.Commit.Parents)]
						break
					}
				}
				if parent == "" {
					parent = m.HistoryInspector.Commit.Parents[0]
				}
				m.HistoryInspectorParent, m.State, m.Status = parent, StateOperationPending, "loading parent-relative details"
				return m, m.inspectSelectedCommit()
			}
		case "g":
			if m.currentView() == workspace.Log {
				m.HistoryRefMode, m.HistoryRefInput = true, ""
				m.Status = "jump to ref: "
			}
		case "y":
			if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				m.Status = "copied " + m.HistoryInspector.Commit.SHA
				return m, tea.SetClipboard(m.HistoryInspector.Commit.SHA)
			}
			if m.currentView() == workspace.GitHub && m.GitHub.Pull.URL != "" {
				m.Status = "copied pull request URL"
				return m, tea.SetClipboard(m.GitHub.Pull.URL)
			}
		case "t":
			if m.currentView() == workspace.Log {
				m.State, m.Status = StateOperationPending, "loading tags"
				return m, m.loadHistoryTags()
			}
		case "C":
			if m.currentView() == workspace.Stashes {
				m.StashCreateMode, m.StashCreateMessage, m.StashIncludeUntracked = true, "", true
				m.Status = "stash message: "
			}
		case "a":
			if m.currentView() == workspace.Stashes && m.Stashes.Selected >= 0 && m.Stashes.Selected < len(m.Stashes.Entries) {
				ref := m.Stashes.Entries[m.Stashes.Selected].Ref
				action := map[string]string{"a": "apply", "D": "drop"}[v.String()]
				m.StashConfirmAction, m.StashConfirmRef = action, ref
				m.Status = "confirm " + action + " " + ref + "? (y/n)"
			} else if m.currentView() == workspace.Status {
				m.State, m.Status = StateOperationPending, "staging all tracked, untracked, and deleted paths"
				return m, m.mutateAll(true)
			}
		case "c":
			if m.currentView() == workspace.Branches {
				m.BranchCreateMode, m.BranchMutationInput = true, ""
				m.Status = "branch name: "
				return m, nil
			}
			if m.currentView() == workspace.GitHub {
				for _, run := range m.GitHub.Checks.Runs {
					if run.URL == "" {
						continue
					}
					if command, err := platform.OpenURLCommand(run.URL); err == nil {
						m.Status = "opening check " + run.Name
						return m, tea.ExecProcess(command, nil)
					}
				}
				m.Status = "no check URL available"
				return m, nil
			}
			return m, m.beginCommit()
		case "enter":
			if m.currentView() == workspace.Rebase {
				if m.Rebase.BaseMode {
					if err := m.Rebase.SetBase(m.Rebase.BaseSelected); err != nil {
						m.Status = err.Error()
					} else {
						m.Rebase.BaseMode = false
						m.Status = "rebase base selected: " + m.Rebase.Base.Ref
					}
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "starting interactive rebase"
				return m, m.startRebase()
			}
			if m.currentView() == workspace.Repositories {
				m.State, m.Status = StateOperationPending, "opening repository"
				return m, m.openSelectedRepository()
			}
			if m.currentView() == workspace.Branches {
				m.State, m.Status = StateOperationPending, "checking out"
				return m, m.checkoutSelectedBranch()
			}
			if m.currentView() == workspace.Worktrees {
				m.State, m.Status = StateOperationPending, "opening worktree"
				return m, m.openSelectedWorktree()
			}
			if m.currentView() == workspace.Log {
				m.State, m.Status = StateOperationPending, "loading commit details"
				return m, m.inspectSelectedCommit()
			}
			if m.currentView() == workspace.Status && m.showCommitTreePane() && m.CommitTreeFocused {
				return m, m.inspectStatusCommit(m.StatusCommitSelectedLine)
			}
			return m, m.openDiff()
		case "j", "down":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				if m.showBranchSummaryPane() {
					m.Branches.Move(1)
				} else if m.showCommitTreePane() {
					m.StatusCommitSelectedLine = min(len(m.CommitTreeLines)-1, max(0, m.StatusCommitSelectedLine+1))
					m.scrollCommitTree(1)
				} else {
					m.scrollContextPane(1)
				}
				return m, nil
			}
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(1)
			case workspace.Stashes:
				m.Stashes.Move(1)
			case workspace.Log:
				m.History.Move(1)
			case workspace.Remotes:
				m.Remotes.Move(1)
			case workspace.Worktrees:
				m.Worktrees.Move(1)
			case workspace.Repositories:
				m.Repositories.Move(1)
			case workspace.Rebase:
				m.Rebase.Move(1)
			case workspace.Plugins:
				m.Plugins.Move(1)
			default:
				m.Files.Move(1, m.statusRowCount())
				if m.DiffPath != "" {
					return m, m.openDiff()
				}
			}
		case "k", "up":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				if m.showBranchSummaryPane() {
					m.Branches.Move(-1)
				} else if m.showCommitTreePane() {
					m.StatusCommitSelectedLine = max(0, m.StatusCommitSelectedLine-1)
					m.scrollCommitTree(-1)
				} else {
					m.scrollContextPane(-1)
				}
				return m, nil
			}
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(-1)
			case workspace.Stashes:
				m.Stashes.Move(-1)
			case workspace.Log:
				m.History.Move(-1)
			case workspace.Remotes:
				m.Remotes.Move(-1)
			case workspace.Worktrees:
				m.Worktrees.Move(-1)
			case workspace.Repositories:
				m.Repositories.Move(-1)
			case workspace.Rebase:
				m.Rebase.Move(-1)
			case workspace.Plugins:
				m.Plugins.Move(-1)
			default:
				m.Files.Move(-1, m.statusRowCount())
				if m.DiffPath != "" {
					return m, m.openDiff()
				}
			}
		case "pgup":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.scrollContextPane(-m.statusLayout().CommitTree.Height)
				return m, nil
			}
			m.scrollDiff(-m.statusRowCount())
		case "pgdown":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.scrollContextPane(m.statusLayout().CommitTree.Height)
				return m, nil
			}
			m.scrollDiff(m.statusRowCount())
		case "home":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.CommitTreeOffset = 0
				m.UnpushedOffset = 0
				return m, nil
			}
		case "end":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.CommitTreeOffset = len(m.CommitTreeLines)
				m.UnpushedOffset = len(m.UnpushedLines)
				m.scrollCommitTree(0)
				m.scrollUnpushed(0)
				return m, nil
			}
		case "space":
			if m.currentView() == workspace.Plugins && m.Plugins.Selected >= 0 && m.Plugins.Selected < len(m.Plugins.Entries) {
				entry := m.Plugins.Entries[m.Plugins.Selected]
				m.Plugins.SetEntries(plugins.SetEnabled(m.Plugins.Entries, entry.Manifest.ID, !entry.Enabled))
				state := "disabled"
				if !entry.Enabled {
					state = "enabled"
				}
				m.Status = "plugin " + entry.Manifest.ID + " " + state
				return m, m.savePluginState(m.Plugins.Entries)
			}
			return m, m.mutate()
		case "d":
			return m, m.openDiff()
		case "H":
			if m.DiffText != "" {
				m.beginHunks()
			}
		case "?":
			m.Modal, m.State = "help", StateModal
		case "r":
			if m.currentView() == workspace.Plugins {
				m.State, m.Status = StateOperationPending, "reloading plugins"
				return m, m.loadPlugins()
			}
			return m, m.refresh()
		}
	case tea.MouseWheelMsg:
		statusLayout := m.statusLayout()
		if m.currentView() == workspace.Status && m.contextPaneFocused() && statusLayout.CommitTree.Contains(v.X, v.Y) {
			m.CommitTreeFocused, m.UnpushedFocused = m.showCommitTreePane(), m.showUnpushedPane()
			switch v.Button {
			case tea.MouseWheelUp:
				m.scrollContextPane(-3)
			case tea.MouseWheelDown:
				m.scrollContextPane(3)
			}
			return m, nil
		}
		if m.DiffPath != "" && (statusLayout.Mode != layout.Wide || statusLayout.Details.Contains(v.X, v.Y)) {
			switch v.Button {
			case tea.MouseWheelUp:
				m.scrollDiff(-3)
			case tea.MouseWheelDown:
				m.scrollDiff(3)
			}
			return m, nil
		}
		switch v.Button {
		case tea.MouseWheelUp:
			m.Files.Move(-1, m.statusRowCount())
		case tea.MouseWheelDown:
			m.Files.Move(1, m.statusRowCount())
		}
	case tea.MouseClickMsg:
		if v.Button == tea.MouseLeft {
			if m.currentView() == workspace.Commit {
				staged := 0
				for _, file := range m.Composer.Draft.Files {
					if file.Staged {
						staged++
					}
				}
				// Feature title/blank lines precede the composer. The subject and
				// body rows follow the staged-file list and its separating blank row.
				subjectY, bodyY := 6+staged, 8+staged
				switch v.Y {
				case subjectY:
					m.Composer.Focus = "subject"
				case bodyY:
					m.Composer.Focus = "body"
				}
				return m, nil
			}
			if m.currentView() == workspace.Branches {
				row := v.Y - 3
				if row >= 0 && row < len(m.Branches.Entries) {
					m.Branches.Selected = row
				}
				return m, nil
			}
			if m.currentView() == workspace.Hunks {
				// The hunk header occupies the first content row; patch lines start at y=3.
				m.Hunks.SelectLine(m.Hunks.LineAt(v.Y - 3))
				return m, nil
			}
			if m.currentView() == workspace.Repositories {
				// Each repository occupies a name/state row followed by its path row.
				row := (v.Y - 3) / 2
				if v.Y >= 3 && row >= 0 && row < len(m.Repositories.Rows) {
					m.Repositories.Selected = row
				}
				return m, nil
			}
			if m.currentView() == workspace.Remotes {
				row := (v.Y - 4) / 3
				if v.Y >= 4 && row >= 0 && row < len(m.Remotes.Dashboard.Remotes) {
					m.Remotes.Selected = row
				}
				return m, nil
			}
			if m.currentView() == workspace.Stashes {
				row := v.Y - 3 // feature title, blank line, and "Stashes" header
				if row >= 0 && row < len(m.Stashes.Entries) {
					m.Stashes.Selected = row
					return m, m.previewSelectedStash()
				}
				return m, nil
			}
			statusLayout := m.statusLayout()
			if m.currentView() == workspace.Status && m.contextPaneFocused() && statusLayout.CommitTree.Contains(v.X, v.Y) {
				m.CommitTreeFocused, m.UnpushedFocused = m.showCommitTreePane(), m.showUnpushedPane()
				if m.showCommitTreePane() {
					line := v.Y - statusLayout.CommitTree.Y - 2 + m.CommitTreeOffset
					if line >= 0 && line < len(m.CommitTreeLines) {
						m.StatusCommitSelectedLine = line
						return m, m.inspectStatusCommit(line)
					}
				}
				return m, nil
			}
			if statusLayout.Mode != layout.Wide && m.DiffPath != "" {
				return m, nil
			}
			files := statusLayout.Files
			if statusLayout.Mode == layout.Wide {
				files.Width = max(1, files.Width-1)
			}
			files.Width = max(1, files.Width-1)
			hit := uimouse.HitMap{Files: files, RowTop: files.Y + 1 + m.statusFileHeaderRows(files.Width), RowHeight: 1, Offset: m.Files.Offset, RowHeights: m.statusFileRowHeights(files.Width), StageX: files.X + 1, StageWidth: 3, RowCount: len(m.Files.Visible)}
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
		m.Hunks.SetHeight(max(1, v.Height-8))
	case refreshResultMsg:
		if v.Coordinator != m.RefreshCoordinator {
			return m, nil
		}
		if !v.Open {
			return m, nil
		}
		if v.Result.Err != nil {
			m.State, m.Status = StateError, v.Result.Err.Error()
			m.recordActivity(history.RefreshError, "", v.Result.Err.Error())
		} else {
			m.applySnapshot(v.Result.Snapshot)
			m.State = StateReady
		}
		return m, tea.Batch(waitForRefresh(v.Coordinator), m.refreshStatusContextIfNeeded())
	case refreshRequestedMsg:
		if v.Coordinator != m.RefreshCoordinator {
			return m, nil
		}
		m.State = StateRefreshing
		v.Coordinator.Request(v.Context)
	case SnapshotMsg:
		m.applySnapshot(v.Snapshot)
		m.State = StateReady
		return m, m.refreshStatusContextIfNeeded()
	case RefreshStartedMsg:
		m.State = StateRefreshing
	case RefreshFinishedMsg:
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
			m.recordActivity(history.RefreshError, "", v.Err.Error())
		} else if m.State == StateRefreshing {
			m.State = StateReady
		}
		return m, m.refreshStatusContextIfNeeded()
	case TickMsg:
		if !m.Motion.Ticks() {
			return m, nil
		}
		m.WatchPulse++
		if m.currentView() == workspace.Log {
			m.HistoryPulse++
			m.History.SetPulse(m.HistoryPulse)
		}
		return m, m.tick()
	case watcherStartedMsg:
		if v.Generation != m.repositoryGeneration {
			if v.Manager != nil {
				if err := v.Manager.Close(); err != nil {
					m.Status = "stale watcher did not close cleanly: " + err.Error()
				}
			}
			return m, nil
		}
		if m.WatchManager != nil && m.WatchManager != v.Manager {
			if err := m.WatchManager.Close(); err != nil {
				m.Status = "superseded watcher did not close cleanly: " + err.Error()
			}
		}
		m.WatchManager = v.Manager
		if v.Manager == nil {
			m.WatchMode = ""
			m.Status = "watcher unavailable"
			if v.Warning != nil {
				m.Status += ": " + v.Warning.Error()
			}
			return m, nil
		}
		m.WatchMode = v.Manager.Mode()
		if v.Warning != nil {
			m.Status = "watcher fallback: " + v.Warning.Error()
			m.recordActivity(history.WatchFallback, "", v.Warning.Error())
		}
		return m, waitForWatcher(v.Manager)
	case watcherEventMsg:
		if v.Manager != m.WatchManager {
			return m, nil
		}
		if !v.Open {
			m.Status = "watcher stopped; restarting with polling"
			m.WatchRequested = watch.RequestedPoll
			m.WatchManager = nil
			return m, m.startWatcher()
		}
		if v.Event.Mode != "" {
			m.WatchMode = v.Event.Mode
		}
		if v.Event.Err != nil {
			m.Status = "watcher fallback: " + v.Event.Err.Error()
			m.recordActivity(history.WatchFallback, v.Event.Path, v.Event.Err.Error())
		}
		return m, tea.Batch(m.refresh(), waitForWatcher(v.Manager))
	case WatcherStateMsg:
		if mode, ok := watch.ParseMode(v.Mode); ok && mode != watch.RequestedAuto {
			m.WatchMode = watch.Mode(mode)
		}
		if v.Err != nil {
			m.Status = "watcher fallback: " + v.Err.Error()
		}
	case OperationStartedMsg:
		m.State = StateOperationPending
		m.Status = v.Name
	case OperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
			m.notify(notifications.JobComplete, notifications.Error, v.Name, v.Err.Error(), true)
			m.recordActivity(history.OperationFailure, "", v.Name+": "+v.Err.Error())
		} else {
			m.State = StateReady
			m.Status = v.Name + " complete"
			m.notify(notifications.JobComplete, notifications.Success, m.Status, "", false)
			m.recordActivity(history.OperationSuccess, "", m.Status)
		}
		return m, m.refresh()
	case RebaseFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Outcome.Paused {
			m.State = StateReady
			m.Status = "interactive rebase paused: " + v.Outcome.State.Phase().String()
			if v.Err != nil {
				m.Status += " (Git reported: " + v.Err.Error() + ")"
			}
			return m, m.refresh()
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, "", "interactive rebase: "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "interactive rebase complete"
			m.recordActivity(history.OperationSuccess, "", m.Status)
		}
		return m, m.refresh()
	case PartialOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.notify(notifications.JobComplete, notifications.Error, v.Name, v.Err.Error(), true)
			m.recordActivity(history.OperationFailure, "", v.Name+": "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, v.Name+" complete"
			m.recordActivity(history.OperationSuccess, "", m.Status)
		}
		return m, m.refresh()
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
		if err := m.shutdown(); err != nil {
			m.Status = "shutdown: " + err.Error()
		}
	case DiffReadyMsg:
		if v.Request != m.DiffRequest {
			return m, nil
		}
		m.DiffCancel, m.DiffLoading = nil, false
		m.DiffPath, m.DiffText, m.DiffStaged, m.DiffBinary, m.DiffAdded, m.DiffDeleted, m.DiffErr, m.DiffTruncated, m.Status = v.Path, v.Text, v.Staged, v.Binary, v.Added, v.Deleted, v.Err, v.Truncated, ""
		if v.Err != nil {
			m.Status = v.Err.Error()
		} else if m.currentView() == workspace.Hunks {
			m.beginHunks()
		}
	case CommitTreeReadyMsg:
		if v.Generation != m.repositoryGeneration || v.Request != m.CommitTreeRequest {
			return m, nil
		}
		m.CommitTreeCancel, m.CommitTreeLoading = nil, false
		if v.Err != nil {
			m.CommitTreeErr = v.Err
			m.Status = "commit tree: " + v.Err.Error()
		} else {
			m.CommitTreeLines, m.CommitTreeHead, m.CommitTreeErr = append([]string(nil), v.Tree.Lines...), v.Tree.Head, nil
			m.CommitTreeOffset = min(m.CommitTreeOffset, max(0, len(m.CommitTreeLines)-1))
		}
	case StatusCommitInspectorReadyMsg:
		if v.Generation != m.repositoryGeneration || v.Request != m.StatusCommitRequest {
			return m, nil
		}
		m.StatusCommitCancel, m.StatusCommitLoading = nil, false
		if v.Err != nil {
			m.StatusCommitErr = v.Err
			m.Status = "commit inspection: " + v.Err.Error()
			return m, nil
		}
		m.StatusCommitActive, m.StatusCommitInspector, m.StatusCommitSHA, m.StatusCommitErr = true, v.Inspector, v.Inspector.Commit.SHA, nil
		entries := make([]repo.Entry, 0, len(v.Inspector.Stats))
		for _, stat := range v.Inspector.Stats {
			kind := byte('M')
			if stat.Binary {
				kind = 'B'
			}
			entries = append(entries, repo.Entry{Path: repo.Path([]byte(stat.Path)), Kind: kind, Unstaged: true, Deleted: stat.Deleted > 0 && stat.Added == 0})
		}
		m.Files.SetEntries(entries)
		m.Files.Selected = 0
		m.CommitTreeFocused, m.UnpushedFocused = false, false
		m.Status = "inspecting commit " + v.Inspector.Commit.Short
	case UnpushedReadyMsg:
		if v.Generation != m.repositoryGeneration || v.Request != m.UnpushedRequest {
			return m, nil
		}
		m.UnpushedCancel, m.UnpushedLoading = nil, false
		if v.Err != nil {
			m.UnpushedErr = v.Err
			m.Status = "unpushed commits: " + v.Err.Error()
		} else {
			m.UnpushedLines = append([]string(nil), v.Commits.Lines...)
			m.UnpushedHead, m.UnpushedUpstream, m.UnpushedCount, m.UnpushedErr = v.Commits.Head, v.Commits.Upstream, v.Commits.Count, nil
			viewport := max(1, m.statusLayout().CommitTree.Height-2)
			m.UnpushedOffset = min(m.UnpushedOffset, max(0, len(m.UnpushedLines)-viewport))
		}
	case BranchesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if len(m.Branches.AllEntries) == 0 {
				m.Branches = branchview.New(v.Entries)
			} else {
				m.Branches.SetEntries(v.Entries)
			}
			m.State = StateReady
		}
	case StashesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if len(m.Stashes.Entries) == 0 {
				m.Stashes = stashview.New(v.Entries)
			} else {
				m.Stashes.SetEntries(v.Entries)
			}
			m.State = StateReady
		}
	case StashPreviewReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.StashPreview, m.StashPreviewRef, m.State, m.Status = v.Text, v.Ref, StateReady, ""
		}
	case StashOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Ref, v.Operation+": "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, v.Operation+" complete"
			m.recordActivity(history.OperationSuccess, v.Ref, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadStashes(), m.loadBranches())
	case CommitFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			if strings.TrimSpace(v.HookOutput) != "" {
				m.Status += "\nhook output:\n" + v.HookOutput
			}
			m.notify(notifications.HookFailure, notifications.Error, "commit hook failed", v.Err.Error(), true)
			m.recordActivity(history.OperationFailure, "", "commit: "+v.Err.Error())
		} else {
			m.CommitAmendConfirm, m.CommitAuthorMode = false, false
			m.State, m.Status = StateReady, "commit "+v.SHA
			m.Workspace.Back()
			m.recordActivity(history.OperationSuccess, "", m.Status)
		}
		return m, m.refresh()
	case CommitConfigReadyMsg:
		m.CommitConfig, m.CommitConfigReady = v.Config, true
		identity := strings.TrimSpace(strings.TrimSpace(v.Config.UserName) + " <" + strings.TrimSpace(v.Config.UserEmail) + ">")
		if strings.TrimSpace(v.Config.UserName) == "" && strings.TrimSpace(v.Config.UserEmail) == "" {
			identity = "Git user identity is not configured"
		}
		signing := "signing off"
		if v.Config.SignEnabled {
			signing = "configured signing: " + strings.TrimSpace(v.Config.SignFormat)
			if strings.TrimSpace(v.Config.SignFormat) == "" {
				signing = "configured signing"
			}
		}
		m.Composer.SetConfigSummary(platform.SafeText("identity: " + identity + "; " + signing))
	case GitHubReadyMsg:
		if v.Err != nil {
			m.GitHub.SetError(v.Repository, v.Branch, v.Err)
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			v.Pull.Checks = provider.Checks{Total: v.Checks.Passing + v.Checks.Failing + v.Checks.Pending, Passing: v.Checks.Passing, Failing: v.Checks.Failing, Pending: v.Checks.Pending}
			v.Pull.ReviewState = v.Review.State()
			m.GitHub.SetData(v.Repository, v.Branch, v.Pull, v.Checks)
			m.State, m.Status = StateReady, "GitHub data loaded"
		}
	case PluginsReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Plugins.SetEntries(v.Entries)
			m.State, m.Status = StateReady, "plugins loaded"
		}
	case PluginStateSavedMsg:
		if v.Err != nil {
			m.Status = "plugin state: " + v.Err.Error()
			m.notify(notifications.PluginFailure, notifications.Error, "plugin state", v.Err.Error(), true)
		}
	case BranchOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Name, v.Operation+": "+v.Err.Error())
		} else {
			operation := v.Operation
			if operation == "" {
				operation = "completed"
			}
			m.State, m.Status = StateReady, operation+" "+v.Name
			m.recordActivity(history.OperationSuccess, v.Name, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadBranches())
	case HistoryReadyMsg:
		m.HistoryCancel = nil
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
	case HistoryRefReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			found := false
			for i, row := range m.History.Rows {
				if row.Commit.SHA == v.SHA {
					m.History.Selected, found = i, true
					break
				}
			}
			m.State = StateReady
			if found {
				m.Status = "jumped to " + v.Ref
			} else {
				m.Status = "ref resolved outside loaded history: " + v.Ref
			}
		}
	case HistoryTagsReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.HistoryTags, m.State, m.Status = v.Tags, StateReady, ""
		}
	case HistoryActionFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Target, v.Action+": "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, v.Action+" "+v.Target
			m.Workspace.Back()
			m.recordActivity(history.OperationSuccess, v.Target, m.Status)
		}
		return m, m.refresh()
	case RemotesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Remotes, m.State = remoteview.New(v.Dashboard), StateReady
			for _, remote := range v.Dashboard.Remotes {
				if v.Dashboard.Stale(remote) {
					m.notify(notifications.RemoteStale, notifications.Warning, "stale remote", remote.Name, true)
				}
			}
		}
	case PushPreviewReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.RemotePushPreview, m.RemotePushConfirm, m.State = v.Preview, true, StateReady
			remoteSHA := v.Preview.RemoteSHA
			if remoteSHA == "" {
				remoteSHA = "(new branch)"
			}
			m.Status = fmt.Sprintf("push %s/%s: %s -> %s? (y/n)", v.Preview.Remote, v.Preview.Branch, remoteSHA, v.Preview.LocalSHA)
		}
	case WorktreesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if len(m.Worktrees.Entries) == 0 {
				m.Worktrees = worktreeview.New(v.Entries)
			} else {
				m.Worktrees.SetEntries(v.Entries)
			}
			m.State = StateReady
		}
	case RepositoriesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.RepositoryRegistry = append([]registry.Repository(nil), v.Repositories...)
			if len(m.Repositories.Rows) == 0 {
				m.Repositories = repoview.New(v.Rows)
			} else {
				m.Repositories.SetRows(v.Rows)
			}
			m.State = StateReady
		}
	case RepositoryOpenedMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if v.PersistenceErr != nil {
				m.Toast = ToastMsg{Text: "repository metadata was not saved: " + v.PersistenceErr.Error(), Error: true}
			}
			for i := range m.RepositoryRegistry {
				if m.RepositoryRegistry[i].Path == v.Path {
					m.RepositoryRegistry[i].LastOpened = time.Now()
				}
			}
			if err := m.setRepository(v.Discovery); err != nil {
				m.Toast = ToastMsg{Text: "previous repository watcher did not close cleanly: " + err.Error(), Error: true}
			}
			m.State, m.Status = StateReady, "opened "+v.Path
			m.Workspace.Navigate(workspace.Status, "Status")
			return m, tea.Batch(m.refresh(), waitForRefresh(m.RefreshCoordinator), m.startWatcher())
		}
	case WorktreeOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Target, v.Operation+": "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, v.Operation+" complete"
			m.recordActivity(history.OperationSuccess, v.Target, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadWorktrees(), m.loadBranches())
	case RemoteOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.RemoteSetUpstream, m.RemoteTag = false, ""
		if m.RemoteJobID != "" {
			for i := range m.Remotes.Dashboard.Jobs {
				if m.Remotes.Dashboard.Jobs[i].ID == m.RemoteJobID {
					job := &m.Remotes.Dashboard.Jobs[i]
					job.Finished = time.Now()
					if v.Err != nil {
						job.State, job.Error, job.Progress = remotes.JobFailed, v.Err.Error(), "failed"
						if errors.Is(v.Err, git.ErrCancelled) || errors.Is(v.Err, context.Canceled) {
							job.State, job.Progress = remotes.JobCanceled, "cancelled"
						}
					} else {
						job.State, job.Progress = remotes.JobSuccess, "complete"
					}
					job.Updated = job.Finished
				}
			}
			m.RemoteCancel, m.RemoteJobID = nil, ""
		}
		m.recordRemoteActivity(v.Operation, v.Remote, v.Err == nil)
		if v.Err != nil {
			kind := notifications.PushFailure
			if strings.HasPrefix(v.Operation, "pull") || v.Operation == "fetch" {
				kind = notifications.HookFailure
			}
			m.notify(kind, notifications.Error, v.Operation, v.Err.Error(), true)
			m.State = StateError
			if remoteConflict(v.Err) {
				m.Status = "conflict during " + v.Operation + ": resolve conflicts, then refresh"
			} else {
				m.Status = v.Err.Error()
			}
			m.recordActivity(history.OperationFailure, v.Remote, v.Operation+": "+v.Err.Error())
		} else {
			m.notify(notifications.JobComplete, notifications.Success, v.Operation, v.Remote, false)
			m.State, m.Status = StateReady, v.Operation+" complete: "+v.Remote
			m.recordActivity(history.OperationSuccess, v.Remote, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadRemotes())
	}
	return m, nil
}

func remoteConflict(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	var commandErr *git.CommandError
	if errors.As(err, &commandErr) {
		text += " " + strings.ToLower(string(commandErr.Result.Stderr))
	}
	return strings.Contains(text, "conflict") || strings.Contains(text, "would be overwritten") || strings.Contains(text, "non-fast-forward")
}

func (m *Model) shutdown() error {
	m.closeDiff()
	if m.cancel != nil {
		m.cancel()
	}
	if m.HistoryCancel != nil {
		m.HistoryCancel()
		m.HistoryCancel = nil
	}
	if m.RemoteCancel != nil {
		m.RemoteCancel()
		m.RemoteCancel = nil
	}
	if m.repositoryCancel != nil {
		m.repositoryCancel()
		m.repositoryCancel = nil
	}
	if m.RefreshCoordinator != nil {
		m.RefreshCoordinator.Close()
	}
	if m.WatchManager != nil {
		err := m.WatchManager.Close()
		m.WatchManager = nil
		return err
	}
	return nil
}

// Close stops repository watchers and in-flight context-aware work.
func (m *Model) Close() error { return m.shutdown() }

func (m *Model) notify(kind notifications.Kind, level notifications.Level, title, message string, attention bool) {
	if m.Notifications != nil {
		m.Notifications.Add(notifications.Notification{Kind: kind, Level: level, Title: title, Message: message, Attention: attention})
	}
	m.Toast = ToastMsg{Text: title + func() string {
		if message == "" {
			return ""
		}
		return ": " + message
	}(), Error: level == notifications.Error}
}

func (m Model) paletteView() tea.View {
	lines := []string{"gitwatch command palette", "", "Search: " + platform.SafeText(m.PaletteQuery), ""}
	if len(m.PaletteResults) == 0 {
		lines = append(lines, "  No matching commands")
	}
	for i, result := range m.PaletteResults {
		prefix := "  "
		if i == m.PaletteSelected {
			prefix = "> "
		}
		state := result.Shortcut
		if !result.Enabled {
			state += " — disabled: " + result.Reason
		}
		lines = append(lines, prefix+result.Label+" ["+state+"]")
	}
	lines = append(lines, "", "[j/k] move  [enter] run  [esc] close")
	v := tea.NewView(strings.Join(safeRenderLines(lines), "\n"))
	v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
	return v
}

func (m Model) View() tea.View {
	if m.PaletteMode {
		return m.paletteView()
	}
	if view := m.currentView(); view == workspace.Branches || view == workspace.Stashes || view == workspace.Log || view == workspace.Commit || view == workspace.Remotes || view == workspace.GitHub || view == workspace.Plugins || view == workspace.Hunks || view == workspace.Worktrees || view == workspace.Repositories || view == workspace.Rebase {
		return m.featureView(view)
	}
	if m.Modal == "help" {
		lines := append([]string{"gitwatch help", ""}, HelpLines()...)
		lines = append(lines, "", "Mouse: click a file row to open its diff; click [ ]/[S] to stage or unstage.")
		v := tea.NewView(strings.Join(safeRenderLines(lines), "\n"))
		v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
		return v
	}
	v := tea.NewView(m.statusView())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) featureView(view workspace.View) tea.View {
	title, content := "gitwatch", "Loading…"
	switch view {
	case workspace.Branches:
		title, content = "gitwatch · branches", m.Branches.View()
	case workspace.Stashes:
		title, content = "gitwatch · stashes", m.Stashes.View()
		if m.StashPreviewRef != "" {
			content += "\n\nPreview " + m.StashPreviewRef + ":\n" + m.StashPreview
		}
		if m.StashCreateMode {
			content += fmt.Sprintf("\n\nStash message: %s\nInclude untracked [u]: %t", m.StashCreateMessage, m.StashIncludeUntracked)
		}
		if m.StashConfirmAction != "" {
			content += "\n\n" + m.Status
		}
		if m.StashBranchMode {
			content += "\n\nBranch name: " + m.StashBranchName + "\n" + m.Status
		}
	case workspace.Log:
		title, content = "gitwatch · history", m.History.View()
		if m.HistorySearching {
			content = "Search: " + m.HistoryFilter + "\n\n" + content
		}
		if m.HistoryInspector.Commit.SHA != "" {
			content += "\n\n" + inspectorText(m.HistoryInspector)
		}
		if m.HistoryInspectorPathMode {
			content += "\n\n" + m.Status
		}
		if m.HistoryRefMode {
			content += "\n\n" + m.Status
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
	case workspace.Commit:
		title, content = "gitwatch · commit", m.Composer.View()
	case workspace.Remotes:
		title, content = "gitwatch · remotes", m.Remotes.View()
		if m.RemoteForceConfirm {
			content += "\n\n" + m.Status
		}
		if m.RemotePushConfirm {
			content += "\n\n" + m.Status
		}
	case workspace.GitHub:
		title, content = "gitwatch · GitHub", m.GitHub.View()
	case workspace.Plugins:
		title, content = "gitwatch · plugins", m.Plugins.View()
	case workspace.Hunks:
		title, content = "gitwatch · hunk selection", m.Hunks.View()
		if m.HunkDiscardConfirm {
			content += "\n\n" + m.Status + ": " + m.HunkDiscardInput
		}
	case workspace.Worktrees:
		title, content = "gitwatch · worktrees", m.Worktrees.View()
	case workspace.Repositories:
		title, content = "gitwatch · repositories", m.Repositories.View()
	case workspace.Rebase:
		title, content = "gitwatch · interactive rebase", m.Rebase.View()
	}
	title += " · watch:" + watchModeName(m.WatchMode)
	lines := []string{title, "", content, "", "──────────────────────────────────────────────────────────────", "[j/k] move  [1] status  [b] branches  [s] stashes  [l] history  [n] remotes  [esc] back  [q] quit"}
	if m.Notifications != nil && m.Notifications.Attention() > 0 {
		lines[len(lines)-1] += fmt.Sprintf("  [!] %d attention  [ctrl+n] dismiss", m.Notifications.Attention())
	}
	if view == workspace.Log {
		lines[len(lines)-1] = "[j/k] move  [enter] inspect  [/] search  [] more  [t] tags  [g] ref  [M] parent  [f] path  [y] copy SHA  [x] checkout  [B] branch  [R] revert  [1] status  [esc] back  [q] quit"
	}
	if view == workspace.Branches {
		lines[len(lines)-1] = "[j/k] move  [/] filter  [s] sort  [enter] checkout  [c] create  [R] rename  [u/N] upstream  [D/X] delete  [esc] back  [q] quit"
		if m.BranchSearching {
			lines[len(lines)-1] = "filter: " + platform.SafeText(m.Branches.Query) + "  [enter] apply  [esc] cancel"
		}
	}
	if view == workspace.Remotes {
		lines[len(lines)-1] = "[j/k] move  [f] fetch  [m] merge  [e] rebase  [o] ff-only  [p] push preview  [P] force-with-lease  [esc] back  [q] quit"
	}
	if view == workspace.GitHub {
		lines[len(lines)-1] = "[r] refresh  [esc] back  [q] quit"
	}
	if view == workspace.Plugins {
		lines[len(lines)-1] = "[j/k] move  [r] reload  [esc] back  [q] quit"
	}
	if view == workspace.Hunks {
		lines[len(lines)-1] = "[j/k] move  [n/p] hunk  [N/P] file  [c] context  [space] select  [a/A/i] hunk/all/invert  [s] stage  [d] discard  [esc] back  [q] quit"
	}
	if view == workspace.Commit {
		lines[len(lines)-1] = "[tab] subject/body  [ctrl+s] commit  [esc] back  [q] quit"
	}
	if view == workspace.Stashes {
		lines[len(lines)-1] = "[j/k] move  [C] create  [B] branch  [a] apply  [p] pop  [D] drop  [enter] preview  [esc] back  [q] quit"
	}
	if view == workspace.Worktrees {
		lines[len(lines)-1] = "[j/k] move  [A] add  [D] remove  [P] prune  [1] status  [esc] back  [q] quit"
		if m.WorktreeAddMode {
			content += "\n\n" + m.Status
		}
		if m.WorktreeConfirmAction != "" {
			content += "\n\n" + m.Status
		}
	}
	if view == workspace.Repositories {
		lines[len(lines)-1] = "[j/k] move  [/] filter  [s] sort  [v] refresh  [enter] open  [esc] back  [q] quit"
		if m.RepositorySearching {
			lines[len(lines)-1] = "filter: " + platform.SafeText(m.Repositories.Query) + "  [enter] apply  [esc] cancel"
		}
	}
	if view == workspace.Rebase {
		lines[len(lines)-1] = "[j/k] move  [b] choose base  [enter] start  [esc] cancel  [q] quit"
	}
	if m.Notifications != nil && m.Notifications.Attention() > 0 {
		lines[len(lines)-1] += fmt.Sprintf("  [!] %d attention  [ctrl+n] dismiss", m.Notifications.Attention())
	}
	if m.Toast.Text != "" {
		content += "\n\nNOTICE: " + platform.SafeText(m.Toast.Text)
	}
	lines[2] = content
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
	return v
}

func safeRenderLines(lines []string) []string {
	safe := make([]string, len(lines))
	for i, line := range lines {
		safe[i] = platform.SafeText(platform.RedactSecrets(line))
	}
	return safe
}

func inspectorText(inspector history.Inspector) string {
	lines := []string{"Selected commit: " + platform.SafeText(inspector.Summary()), "Files:"}
	if len(inspector.Commit.Parents) > 0 {
		parents := make([]string, len(inspector.Commit.Parents))
		for i, parent := range inspector.Commit.Parents {
			parents[i] = platform.SafeText(parent)
		}
		lines = append(lines, "Parents: "+strings.Join(parents, ", "))
	}
	if len(inspector.Commit.Refs) > 0 {
		refs := make([]string, len(inspector.Commit.Refs))
		for i, ref := range inspector.Commit.Refs {
			refs[i] = platform.SafeText(ref)
		}
		lines = append(lines, "Refs: "+strings.Join(refs, ", "))
	}
	if inspector.Parent != "" {
		lines[0] += " (parent " + platform.SafeText(inspector.Parent) + ")"
	}
	for _, stat := range inspector.Stats {
		if stat.Binary {
			lines = append(lines, "  "+platform.SafeText(stat.Path)+" [binary]")
		} else {
			lines = append(lines, fmt.Sprintf("  %s +%d -%d", platform.SafeText(stat.Path), stat.Added, stat.Deleted))
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

func watchModeName(mode watch.Mode) string {
	if mode == "" {
		return "starting"
	}
	return string(mode)
}
