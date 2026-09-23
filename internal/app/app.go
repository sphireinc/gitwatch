package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/bisect"
	"github.com/sphireinc/git-watch/internal/blame"
	"github.com/sphireinc/git-watch/internal/branches"
	"github.com/sphireinc/git-watch/internal/cherrypick"
	"github.com/sphireinc/git-watch/internal/commands"
	"github.com/sphireinc/git-watch/internal/commitmodel"
	compareops "github.com/sphireinc/git-watch/internal/compare"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/customcmd"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/manage"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
	"github.com/sphireinc/git-watch/internal/gitignore/recommend"
	"github.com/sphireinc/git-watch/internal/gitignore/security"
	"github.com/sphireinc/git-watch/internal/health"
	"github.com/sphireinc/git-watch/internal/history"
	mergeops "github.com/sphireinc/git-watch/internal/merge"
	"github.com/sphireinc/git-watch/internal/multirepo"
	"github.com/sphireinc/git-watch/internal/notifications"
	"github.com/sphireinc/git-watch/internal/operations"
	"github.com/sphireinc/git-watch/internal/patch"
	"github.com/sphireinc/git-watch/internal/pathhistory"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/rebase"
	redopolicy "github.com/sphireinc/git-watch/internal/redo"
	"github.com/sphireinc/git-watch/internal/reflog"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/remoteintel"
	"github.com/sphireinc/git-watch/internal/remotes"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
	"github.com/sphireinc/git-watch/internal/stash"
	"github.com/sphireinc/git-watch/internal/submodules"
	"github.com/sphireinc/git-watch/internal/tags"
	"github.com/sphireinc/git-watch/internal/ui/activityviz"
	"github.com/sphireinc/git-watch/internal/ui/blameview"
	"github.com/sphireinc/git-watch/internal/ui/branchview"
	"github.com/sphireinc/git-watch/internal/ui/committree"
	"github.com/sphireinc/git-watch/internal/ui/commitview"
	"github.com/sphireinc/git-watch/internal/ui/compareview"
	"github.com/sphireinc/git-watch/internal/ui/conflictview"
	"github.com/sphireinc/git-watch/internal/ui/details"
	"github.com/sphireinc/git-watch/internal/ui/filetree"
	"github.com/sphireinc/git-watch/internal/ui/githubview"
	"github.com/sphireinc/git-watch/internal/ui/gitignoreview"
	"github.com/sphireinc/git-watch/internal/ui/historyview"
	"github.com/sphireinc/git-watch/internal/ui/hunkview"
	"github.com/sphireinc/git-watch/internal/ui/layout"
	uimouse "github.com/sphireinc/git-watch/internal/ui/mouse"
	"github.com/sphireinc/git-watch/internal/ui/pathhistoryview"
	"github.com/sphireinc/git-watch/internal/ui/pluginview"
	"github.com/sphireinc/git-watch/internal/ui/rebaseview"
	"github.com/sphireinc/git-watch/internal/ui/reflogview"
	"github.com/sphireinc/git-watch/internal/ui/remoteview"
	"github.com/sphireinc/git-watch/internal/ui/repoview"
	"github.com/sphireinc/git-watch/internal/ui/stashview"
	"github.com/sphireinc/git-watch/internal/ui/table"
	"github.com/sphireinc/git-watch/internal/ui/theme"
	"github.com/sphireinc/git-watch/internal/ui/worktreeview"
	undopolicy "github.com/sphireinc/git-watch/internal/undo"
	"github.com/sphireinc/git-watch/internal/watch"
	"github.com/sphireinc/git-watch/internal/workspace"
	"github.com/sphireinc/git-watch/internal/worktrees"
)

type State uint8

const maxRepositoryParentDepth = 8
const maxBisectDisplayOutput = 64 << 10

const (
	StateLoading State = iota
	StateReady
	StateRefreshing
	StateOperationPending
	StateError
	StateModal
	StateShutdown
)

type SnapshotMsg struct {
	Generation uint64
	Snapshot   repo.Snapshot
}
type SubmodulesReadyMsg struct {
	Generation uint64
	Snapshot   submodules.Snapshot
	Err        error
}
type SubmoduleFinishedMsg struct {
	Generation uint64
	Outcome    submodules.Outcome
}
type SubmoduleOpenedMsg struct {
	Generation uint64
	Path       string
	Discovery  git.Discovery
	Err        error
}
type BulkSubmoduleFinishedMsg struct {
	Generation uint64
	Outcome    submodules.BulkOutcome
}
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
	Operation  *history.OperationRecord
	Err        error
}
type RebaseFinishedMsg struct {
	Repository uint64
	Outcome    git.RebaseOutcome
	Operation  *history.OperationRecord
	Err        error
}
type CherryPickFinishedMsg struct {
	Repository uint64
	Outcome    cherrypick.Outcome
	Operation  *history.OperationRecord
}
type UndoFinishedMsg struct {
	Repository uint64
	Outcome    undopolicy.Outcome
	Operation  *history.OperationRecord
}
type RedoFinishedMsg struct {
	Repository uint64
	Outcome    redopolicy.Outcome
	Operation  *history.OperationRecord
}
type RebaseContinueFinishedMsg struct {
	Repository uint64
	Result     git.Result
	Err        error
}
type RebaseAbortFinishedMsg struct {
	Repository uint64
	Result     git.Result
	Err        error
}
type TickMsg struct{ At time.Time }
type AutoFetchFinishedMsg struct {
	Results []remoteintel.Result
	Err     error
}
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
type CompareReadyMsg struct {
	Generation uint64
	Request    uint64
	Result     compareops.Result
	Err        error
}
type ComparePatchReadyMsg struct {
	Generation uint64
	Request    uint64
	Path       string
	Text       string
	Truncated  bool
	Err        error
}
type ConflictContentReadyMsg struct {
	Content    git.ConflictContent
	Generation uint64
	Request    uint64
	Err        error
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
type HistoricalPatchAppliedMsg struct {
	Repository uint64
	Err        error
}
type ExternalToolFinishedMsg struct {
	Name       string
	Repository uint64
	Err        error
}
type HistoricalToolReadyMsg struct {
	Name       string
	Repository uint64
	Tool       platform.ExternalTool
	Path       string
	Content    []byte
	Err        error
}
type CustomCommandFinishedMsg struct {
	Name       string
	Repository uint64
	Refresh    bool
	Output     customcmd.Output
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
type ReflogReadyMsg struct {
	Entries    []reflog.Entry
	Generation uint64
	Skip       int
	HasMore    bool
	Err        error
}
type BisectReadyMsg struct {
	Repository uint64
	State      bisect.State
	Err        error
}
type BisectFinishedMsg struct {
	Repository uint64
	Action     string
	Outcome    bisect.Outcome
}
type BisectOutputMsg struct {
	Repository uint64
	Chunk      git.OutputChunk
	Open       bool
	Output     <-chan git.OutputChunk
}
type ReflogCompareReadyMsg struct {
	Text       string
	Generation uint64
	Err        error
}
type HistoryReadyMsg struct {
	Commits []history.Commit
	Skip    int
	HasMore bool
	Err     error
}
type PathHistoryReadyMsg struct {
	Path       string
	Entries    []pathhistory.Entry
	Skip       int
	HasMore    bool
	Follow     bool
	Request    uint64
	Generation uint64
	Err        error
}
type BlameReadyMsg struct {
	Path       string
	Start      int
	Lines      []blame.Line
	HasMore    bool
	Truncated  bool
	Request    uint64
	Generation uint64
	Err        error
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
type TagsReadyMsg struct {
	Generation uint64
	Snapshot   tags.Snapshot
	Err        error
}
type TagSignatureReadyMsg struct {
	Generation uint64
	Name       string
	State      tags.SignatureState
	Err        error
}
type TagCheckoutFinishedMsg struct {
	Generation uint64
	Name       string
	Err        error
}
type TagCompareReadyMsg struct {
	Generation uint64
	Name       string
	Text       string
	Err        error
}
type TagWorktreeFinishedMsg struct {
	Generation uint64
	Name       string
	Path       string
	Err        error
}
type TagMutationFinishedMsg struct {
	Generation uint64
	Operation  string
	Name       string
	Err        error
}
type HistoryActionFinishedMsg struct {
	Action, Target string
	Repository     uint64
	Err            error
}
type RevertFinishedMsg struct {
	Repository uint64
	Result     git.Result
	Snapshot   *repo.Snapshot
	Operation  *history.OperationRecord
	Paused     bool
	Err        error
}
type CommitFinishedMsg struct {
	SHA        string
	HookOutput string
	Repository uint64
	Operation  *history.OperationRecord
	Err        error
}
type FixupFinishedMsg struct {
	SHA, Target string
	Repository  uint64
	Err         error
}
type CommitConfigReadyMsg struct{ Config git.CommitConfig }
type BranchOperationFinishedMsg struct {
	Operation  string
	Name       string
	Repository uint64
	Err        error
}

type MergeFinishedMsg struct {
	Repository uint64
	Outcome    mergeops.Outcome
	Operation  *history.OperationRecord
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
type RepositoryBatchFinishedMsg struct {
	Results []multirepo.Result
	Err     error
}
type RepositoryBatchProgressMsg struct {
	Path      string
	Status    string
	Completed int
	Total     int
	Events    <-chan tea.Msg
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
	Journal           *history.OperationRecord
	Err               error
}
type RemoteTrackingReadyMsg struct {
	Remote     string
	Branches   []remotes.TrackingBranch
	Repository uint64
	Err        error
}
type RemotePrunePreviewMsg struct {
	Remote     string
	Text       string
	Repository uint64
	Err        error
}
type PushPreviewReadyMsg struct {
	Preview remotes.RefMovement
	Err     error
}
type GitHubReadyMsg struct {
	Generation    uint64
	Repository    provider.Repository
	Branch        string
	Pull          provider.PullRequest
	Pulls         []provider.PullRequest
	Issues        []provider.Issue
	Releases      []provider.Release
	Detail        *provider.PullRequestDetail
	Comments      []provider.ReviewComment
	Checks        provider.ChecksSnapshot
	Review        provider.ReviewSnapshot
	ProviderStale bool
	Err           error
}

type providerCIAttention struct {
	State, Attention string
	Stale            bool
}
type GitHubPullRequestCreatedMsg struct {
	Pull provider.PullRequest
	Err  error
}
type GitHubMergeFinishedMsg struct {
	Result provider.MergeResult
	Err    error
}
type GitHubBranchDeleteFinishedMsg struct {
	Branch string
	Err    error
}
type GitHubReviewFinishedMsg struct {
	Result provider.ReviewSubmissionResult
	Err    error
}
type GitHubReviewCommentFinishedMsg struct {
	Comment provider.ReviewComment
	Err     error
}
type GitHubCheckActionFinishedMsg struct {
	Action string
	Err    error
}
type GitHubIssueCreatedMsg struct {
	Issue provider.Issue
	Err   error
}
type PluginsReadyMsg struct {
	Entries []plugins.Entry
	Err     error
}
type GitignoreReadyMsg struct {
	Model      gitignoreview.RepositoryModel
	Err        error
	Missing    bool
	ReadOnly   bool
	Reload     bool
	Generation uint64
}
type GitignoreCreatePreviewMsg struct {
	Plan       domain.MutationPlan
	Text       string
	Err        error
	Repository uint64
}
type GitignoreMutationFinishedMsg struct {
	Action     string
	Repository uint64
	Err        error
}
type GitignoreCatalogReadyMsg struct {
	Source     catalog.Source
	Generation uint64
	Err        error
}
type PluginStateSavedMsg struct{ Err error }

type Model struct {
	State                     State
	Width, Height             int
	Focus, Modal, Status      string
	Motion                    Motion
	Keymap                    map[string]string
	Toast                     ToastMsg
	Notifications             *notifications.Model
	Snapshot                  repo.Snapshot
	Submodules                submodules.Snapshot
	SubmodulesLoading         bool
	SubmodulesGeneration      uint64
	SubmodulesErr             error
	SubmoduleAction           string
	SubmodulePath             string
	SubmoduleInput            string
	SubmoduleURL              string
	BulkSubmoduleAction       string
	BulkSubmodulePaths        []string
	BulkSubmoduleOutcome      *submodules.BulkOutcome
	BulkSubmoduleCancel       context.CancelFunc
	Discovery                 git.Discovery
	Files                     table.Model
	FileTree                  filetree.Model
	StatusTreeMode            bool
	FileFilterMode            bool
	FileFilterInput           string
	FileConflictOnly          bool
	Theme                     theme.Roles
	PanelSplit                layout.Split
	DetailsCache              *details.Cache
	ActivityLog               *history.Log
	ctx                       context.Context
	cancel                    context.CancelFunc
	repositoryCtx             context.Context
	repositoryCancel          context.CancelFunc
	RefreshInterval           time.Duration
	ReconciliationInterval    time.Duration
	WatchDebounce             time.Duration
	WatchRequested            watch.RequestedMode
	WatchMode                 watch.Mode
	WatchManager              *watch.Manager
	RefreshCoordinator        *git.RefreshCoordinator
	repositoryGeneration      uint64
	DiffPath, DiffText        string
	DiffBinary                bool
	DiffStaged                bool
	DiffLoading               bool
	DiffErr                   error
	DiffOffset                int
	DiffAdded                 int
	DiffDeleted               int
	DiffRequest               uint64
	DiffCancel                context.CancelFunc
	DiffAutoPreviewed         bool
	DiffSearchMode            bool
	DiffSearchInput           string
	DiffSearchMatch           int
	DiffTruncated             bool
	Compare                   compareview.Model
	CompareLeft               string
	CompareRight              string
	CompareAssignSide         string
	CompareLoading            bool
	CompareErr                error
	CompareRequest            uint64
	CompareCancel             context.CancelFunc
	CompareGeneration         uint64
	ComparePatchLoading       bool
	ComparePatchRequest       uint64
	ComparePatchCancel        context.CancelFunc
	DiffMaxBytes              int64
	DiffMaxLines              int
	EditorTool                platform.ExternalTool
	OpenerTool                platform.ExternalTool
	Difftool                  platform.ExternalTool
	CommitTreeEnabled         bool
	CommitTreeMaxCommits      int
	CommitTreeLines           []string
	CommitTreeHead            string
	CommitTreeOffset          int
	CommitTreeFocused         bool
	CommitTreeLoading         bool
	CommitTreeErr             error
	CommitTreeRequest         uint64
	CommitTreeCancel          context.CancelFunc
	LowerPane                 string
	UnpushedLines             []string
	UnpushedHead              string
	UnpushedUpstream          string
	UnpushedCount             int
	UnpushedOffset            int
	UnpushedFocused           bool
	UnpushedLoading           bool
	UnpushedErr               error
	UnpushedRequest           uint64
	UnpushedCancel            context.CancelFunc
	StatusCommitActive        bool
	StatusCommitInspector     history.Inspector
	StatusCommitSHA           string
	StatusCommitSelectedLine  int
	StatusCommitLoading       bool
	StatusCommitErr           error
	StatusCommitRequest       uint64
	StatusCommitCancel        context.CancelFunc
	Restore                   Confirmation
	RestoreInput              string
	HunkContext               int
	Workspace                 *workspace.Model
	Branches                  branchview.Model
	BranchSearching           bool
	BranchCreateMode          bool
	BranchRenameMode          bool
	BranchRenameOld           string
	BranchUpstreamMode        bool
	BranchMergeMode           bool
	BranchMergeTarget         string
	BranchMutationInput       string
	BranchDeleteMode          bool
	BranchDeleteTarget        branches.Branch
	BranchDeleteForce         bool
	RemoteBranchAction        string
	RemoteBranchTarget        branches.Branch
	RemoteBranchInput         string
	RemoteBranchConfirm       bool
	BranchRecoveryAction      string
	BranchRecoveryTarget      string
	BranchRecoveryConfirm     bool
	BranchResetPrompt         bool
	BranchResetInput          string
	Stashes                   stashview.Model
	Reflog                    reflogview.Model
	ReflogSkip                int
	ReflogLoading             bool
	ReflogCompare             string
	ReflogCompareLoading      bool
	JournalOffset             int
	JournalFilterMode         bool
	JournalFilterInput        string
	JournalDetailMode         bool
	JournalCancelID           string
	JournalCancelConfirm      bool
	JournalRetryID            string
	JournalRetryConfirm       bool
	Bisect                    bisect.State
	BisectLoading             bool
	BisectResetConfirm        bool
	BisectStartMode           string
	BisectStartBad            string
	BisectStartGood           string
	BisectStartInput          string
	BisectStartConfirm        bool
	BisectRunMode             string
	BisectRunExecutable       string
	BisectRunInput            string
	BisectRunArgs             []string
	BisectRunConfirm          bool
	BisectRunOutput           string
	UndoConfirm               bool
	UndoRecord                *history.OperationRecord
	RedoConfirm               bool
	RedoRecord                *history.OperationRecord
	History                   historyview.Model
	Rebase                    rebaseview.Model
	Conflict                  conflictview.Model
	ConflictContentLoading    bool
	ConflictContentRequest    uint64
	ConflictContentCancel     context.CancelFunc
	RebaseConfirmAction       rebase.Action
	RebaseAutosquashConfirm   bool
	HistoricalRebaseAction    rebase.Action
	HistoricalRebaseTarget    string
	HistoryCommits            []history.Commit
	HistorySkip               int
	HistoryHasMore            bool
	HistoryCancel             context.CancelFunc
	HistoryPulse              uint8
	WatchPulse                uint8
	HistoryFilter             string
	HistorySearching          bool
	HistoryRangeAnchor        int
	HistoryRangeAnchorSet     bool
	HistoryInspector          history.Inspector
	HistoryInspectorParent    string
	HistoryInspectorPathMode  bool
	HistoryInspectorPath      string
	PathHistory               pathhistoryview.Model
	PathHistoryLoading        bool
	PathHistoryErr            error
	PathHistoryRequest        uint64
	PathHistoryCancel         context.CancelFunc
	PathHistoryGeneration     uint64
	Blame                     blameview.Model
	BlameLoading              bool
	BlameErr                  error
	BlameRequest              uint64
	BlameCancel               context.CancelFunc
	BlameGeneration           uint64
	HistoryRefMode            bool
	HistoryRefInput           string
	HistoryTags               []history.Ref
	TagSnapshot               tags.Snapshot
	TagsLoading               bool
	TagsErr                   error
	TagsSelected              int
	TagsFilter                string
	TagsFilterMode            bool
	TagsSort                  string
	TagsSortDesc              bool
	TagSignatureChecking      string
	TagCreateMode             string
	TagCreateKind             tags.CreateKind
	TagCreateName             string
	TagCreateTarget           string
	TagCreateMessage          string
	TagCreateInput            string
	TagDeleteMode             bool
	TagDeleteTarget           string
	TagDeleteInput            string
	TagCheckoutConfirm        bool
	TagCheckoutTarget         string
	TagCompare                string
	TagCompareLoading         bool
	TagWorktreeMode           bool
	TagWorktreePath           string
	HistoryActionConfirm      bool
	HistoryActionTarget       string
	HistoryBranchCreating     bool
	HistoryBranchTarget       string
	HistoryBranchName         string
	HistoryRevertConfirm      bool
	HistoryRevertTarget       string
	HistoryRevertCommits      []string
	HistoryRevertInput        string
	HistoryRevertInvalid      bool
	HistoryRevertParentMode   bool
	HistoryRevertParentInput  string
	HistoryRevertParent       int
	HistoryRevertParentMax    int
	HistoryRevertRunning      bool
	CherryPickConfirm         bool
	CherryPickCommits         []string
	Composer                  commitview.Composer
	Hunks                     hunkview.Model
	HunkDiscardConfirm        bool
	HunkDiscardInput          string
	HistoricalPatchMode       bool
	HistoricalPatch           []byte
	HistoricalPatchTarget     string
	HistoricalPatchPath       string
	CommitConfig              git.CommitConfig
	CommitConfigReady         bool
	CommitAmendConfirm        bool
	CommitAuthorMode          bool
	StashPreview              string
	StashPreviewRef           string
	StashCreateMode           bool
	StashCreateMessage        string
	StashIncludeUntracked     bool
	StashConfirmAction        string
	StashConfirmRef           string
	StashBranchMode           bool
	StashBranchRef            string
	StashBranchName           string
	Remotes                   remoteview.Model
	Worktrees                 worktreeview.Model
	WorktreeAddMode           bool
	WorktreeAddPath           string
	WorktreeAddCommit         string
	WorktreeConfirmAction     string
	WorktreeConfirmTarget     string
	RemoteForceConfirm        bool
	RemotePushConfirm         bool
	RemotePushPreview         remotes.RefMovement
	RemoteSetUpstream         bool
	RemoteTag                 string
	RemoteTagMode             bool
	RemoteTagDeleteMode       bool
	RemoteTagDeleteConfirm    bool
	RemoteMutationMode        string
	RemoteMutationRemote      string
	RemoteMutationInput       string
	RemoteMutationURL         string
	RemoteMutationNewName     string
	RemoteMutationImpact      []remotes.TrackingBranch
	RemoteMutationConfirm     bool
	RemotePrunePreview        string
	RemotePruneConfirm        bool
	RemoteCancel              context.CancelFunc
	RemoteJobID               string
	GitHub                    githubview.Model
	GitHubEnabled             bool
	GitHubTokenEnv            string
	GitHubCache               *provider.PullRequestCache
	GitHubPullsCache          *provider.Cache[[]provider.PullRequest]
	GitHubDetailsCache        *provider.Cache[provider.PullRequestDetail]
	GitHubCommentsCache       *provider.Cache[[]provider.ReviewComment]
	GitHubChecksCache         *provider.Cache[provider.ChecksSnapshot]
	GitHubReviewsCache        *provider.Cache[provider.ReviewSnapshot]
	GitHubIssuesCache         *provider.Cache[[]provider.Issue]
	GitHubReleasesCache       *provider.Cache[[]provider.Release]
	ProviderCI                map[string]providerCIAttention
	GitHubCreateMode          bool
	GitHubCreateField         int
	GitHubCreateTitle         string
	GitHubCreateBody          string
	GitHubCreateBase          string
	GitHubCreateConfirm       bool
	GitHubMergeMode           bool
	GitHubMergeMethod         provider.MergeMethod
	GitHubMergeRefresh        bool
	GitHubMergeConfirm        bool
	GitHubBranchDeleteConfirm bool
	GitHubBranchDeleteTarget  string
	GitHubReviewMode          bool
	GitHubReviewEvent         provider.ReviewEvent
	GitHubReviewBody          string
	GitHubReplyCommentID      int64
	GitHubReviewConfirm       bool
	GitHubCheckAction         string
	GitHubCheckActionRunID    int64
	GitHubCheckActionConfirm  bool
	GitHubIssueMode           bool
	GitHubIssueField          int
	GitHubIssueTitle          string
	GitHubIssueBody           string
	GitHubIssueLabels         string
	GitHubIssueConfirm        bool
	Plugins                   pluginview.Model
	Gitignore                 gitignoreview.RepositoryModel
	GitignoreMissing          bool
	GitignoreCreateConfirm    bool
	GitignoreCreatePlan       domain.MutationPlan
	GitignoreMutationAction   string
	GitignoreReturnToStatus   bool
	GitignoreReadOnly         bool
	GitignoreMaxBytes         int64
	GitignoreCatalog          *catalog.Catalog
	GitignoreCatalogSource    catalog.SourceKind
	PluginsEnabled            bool
	PluginDirectories         []string
	PluginOutputLimit         int64
	PluginStatePath           string
	Repositories              repoview.Model
	RepositorySearching       bool
	RepositoryRoots           []string
	RepositoryGroups          map[string][]string
	RepositoryGroup           string
	RepositoryMaxDepth        int
	RepositoryMaxCount        int
	RepositoryIgnoreDirs      []string
	RepositoryBatchConfirm    bool
	RepositoryBatchRetry      bool
	RepositoryBatchAction     multirepo.Action
	RepositoryBatchStrategy   string
	RepositoryBatchCancel     context.CancelFunc
	RepositoryBatchResults    []multirepo.Result
	RepositoryRegistry        []registry.Repository
	RepositoryRegistryPath    string
	RepositoryEngine          *registry.Engine
	AutoFetchScheduler        *remoteintel.Scheduler
	AutoFetchEnabled          bool
	AutoFetchRunning          bool
	AutoFetchResults          map[string]remoteintel.Result
	OperationEngine           *operations.Engine
	CustomCommands            []customcmd.Definition
	CustomCommandForm         *customcmd.Form
	CustomCommandPending      string
	CustomCommandPromptValues map[string]string
	PaletteMode               bool
	PaletteQuery              string
	PaletteSelected           int
	PaletteResults            []commands.Match
	PaletteActions            []commands.Action
	PaletteCommands           map[string]func() tea.Cmd
	PaletteMaxResults         int
	StatusOverscan            int
	repositoryParents         []repositoryParent
}

type repositoryParent struct {
	Discovery git.Discovery
	Label     string
}

func New() Model {
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		State: StateLoading, Focus: "files", Motion: MotionFull,
		Keymap: config.DefaultKeymap(), GitHub: githubview.New(),
		GitHubCache:         provider.NewPullRequestCache(2 * time.Minute),
		GitHubPullsCache:    provider.NewCache[[]provider.PullRequest](2 * time.Minute),
		GitHubDetailsCache:  provider.NewCache[provider.PullRequestDetail](2 * time.Minute),
		GitHubCommentsCache: provider.NewCache[[]provider.ReviewComment](2 * time.Minute),
		GitHubChecksCache:   provider.NewCache[provider.ChecksSnapshot](2 * time.Minute),
		GitHubReviewsCache:  provider.NewCache[provider.ReviewSnapshot](2 * time.Minute),
		GitHubIssuesCache:   provider.NewCache[[]provider.Issue](2 * time.Minute),
		GitHubReleasesCache: provider.NewCache[[]provider.Release](2 * time.Minute),
		Plugins:             pluginview.New(nil), Theme: theme.New(theme.Auto, false), PanelSplit: layout.DefaultSplit(),
		DetailsCache: details.NewCache(), ActivityLog: history.New(100),
		ctx: ctx, cancel: cancel, RefreshInterval: 2 * time.Second,
		ReconciliationInterval: 30 * time.Second, WatchDebounce: 75 * time.Millisecond,
		DiffMaxBytes: 4 << 20, DiffMaxLines: 20_000, GitignoreMaxBytes: security.DefaultMaxDocumentBytes, CommitTreeMaxCommits: config.DefaultCommitTreeCommits,
		WatchRequested: watch.RequestedAuto, Workspace: workspace.New(), Conflict: conflictview.New(), Gitignore: gitignoreview.RepositoryModel{RepositoryID: domain.RepositoryID(""), Width: 80, Height: 24},
		Notifications: notifications.New(100, false), OperationEngine: operations.New(4), Reflog: reflogview.New("HEAD"),
		TagsSort: "name",
	}
}

func (m Model) paletteActions() []commands.Action {
	paletteIndexLimit := m.PaletteMaxResults
	if paletteIndexLimit < 1 {
		paletteIndexLimit = 200
	}
	actions := []commands.Action{
		{ID: "status", Label: "Show status", Shortcut: "1", Enabled: m.Discovery.Root != ""},
		{ID: "gitignore", Label: "Open gitignore catalog", Shortcut: "I", Enabled: m.Discovery.Root != ""},
		{ID: "branches", Label: "Open branches", Shortcut: "b", Enabled: m.Discovery.Root != ""},
		{ID: "stashes", Label: "Open stashes", Shortcut: "s", Enabled: m.Discovery.Root != ""},
		{ID: "history", Label: "Open history", Shortcut: "l", Enabled: m.Discovery.Root != ""},
		{ID: "tags", Label: "Open tags", Shortcut: "t", Enabled: m.Discovery.Root != ""},
		{ID: "reflog", Label: "Open reflog recovery points", Shortcut: "R", Enabled: m.Discovery.Root != ""},
		{ID: "journal", Label: "Open operation journal", Shortcut: "J", Enabled: m.Discovery.Root != "" && m.ActivityLog != nil},
		{ID: "bisect", Label: "Open bisect state", Shortcut: "", Enabled: m.Discovery.Root != ""},
		{ID: "clear_commit_basket", Label: fmt.Sprintf("Clear commit basket (%d)", m.History.Basket.Count()), Shortcut: "C", Enabled: m.History.Basket.Count() > 0},
		{ID: "rebase", Label: "Open interactive rebase", Shortcut: "I", Enabled: m.Discovery.Root != "" && len(m.HistoryCommits) > 0},
		{ID: "cherry_pick_recovery", Label: "Reopen active cherry-pick", Shortcut: "C", Enabled: m.Snapshot.Operation != nil && m.Snapshot.Operation.Kind() == sequencer.KindCherryPick},
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
	if operation := m.Snapshot.Operation; operation != nil {
		if _, ok := recoveryWorkspaceRoute(operation.Kind()); ok {
			actions = append(actions, commands.Action{ID: "operation_recovery", Label: "Reopen active " + recoveryWorkspaceLabel(operation.Kind()), Category: "recovery", Enabled: true})
		}
	}
	if m.GitHub.Repository.Owner != "" && m.GitHub.Repository.Name != "" {
		if m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
			actions = append(actions, commands.Action{ID: "github_commit_selected", Label: "Open selected commit on GitHub", Category: "provider", Enabled: true})
		}
		if m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
			actions = append(actions, commands.Action{ID: "github_branch_selected", Label: "Open selected branch on GitHub", Category: "provider", Enabled: true})
		}
		if _, ok := m.selectedTag(); ok {
			actions = append(actions, commands.Action{ID: "github_tag_selected", Label: "Open selected tag on GitHub", Category: "provider", Enabled: true})
		}
	}
	for index, branch := range m.Branches.Entries {
		if index >= paletteIndexLimit {
			break
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_branch_%d", index), Label: platform.SafeText("Open branch: " + branch.Name), Category: "branch", Enabled: true})
	}
	for index, row := range m.History.Rows {
		if index >= paletteIndexLimit {
			break
		}
		label := "Open commit: " + row.Commit.Short + " " + row.Commit.Subject
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_commit_%d", index), Label: platform.SafeText(label), Category: "commit", Enabled: true})
	}
	for index, visible := range m.Files.Visible {
		if index >= paletteIndexLimit || visible < 0 || visible >= len(m.Files.Entries) {
			break
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_file_%d", index), Label: platform.SafeText("Open file: " + string(m.Files.Entries[visible].Path)), Category: "file", Enabled: true})
	}
	for index, pull := range m.GitHub.Pulls {
		if index >= paletteIndexLimit {
			break
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_pr_%d", index), Label: platform.SafeText(fmt.Sprintf("Open pull request #%d: %s", pull.Number, pull.Title)), Category: "provider", Enabled: true})
	}
	for index, issue := range m.GitHub.Issues {
		if index >= paletteIndexLimit {
			break
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_issue_%d", index), Label: platform.SafeText(fmt.Sprintf("Open issue #%d: %s", issue.Number, issue.Title)), Category: "provider", Enabled: true})
	}
	for index, release := range m.GitHub.Releases {
		if index >= paletteIndexLimit {
			break
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_release_%d", index), Label: platform.SafeText("Open release: " + release.TagName + " " + release.Name), Category: "provider", Enabled: true})
	}
	for index, entry := range m.Plugins.Entries {
		if index >= paletteIndexLimit {
			break
		}
		name := entry.Manifest.Name
		if name == "" {
			name = entry.Manifest.ID
		}
		actions = append(actions, commands.Action{ID: fmt.Sprintf("palette_plugin_%d", index), Label: platform.SafeText("Open plugin: " + name), Category: "plugin", Enabled: true})
	}
	for index, row := range m.Repositories.Rows {
		label := "Open repository: " + row.Repository.Name
		if row.Repository.Path != "" {
			label += " (" + row.Repository.Path + ")"
		}
		if row.NeedsAttention() {
			label = "Open repository attention: " + row.Repository.Name + " (" + row.Repository.Path + ")"
		}
		id := fmt.Sprintf("repository_open_%d", index)
		if row.NeedsAttention() {
			id = fmt.Sprintf("repository_attention_%d", index)
		}
		actions = append(actions, commands.Action{ID: id, Label: platform.SafeText(label), Category: "repository", Shortcut: "v", Enabled: row.Repository.Path != ""})
	}
	for _, definition := range m.CustomCommands {
		label := definition.Label
		if label == "" {
			label = definition.Name
		}
		actions = append(actions, commands.Action{ID: "customcmd:" + definition.Name, Label: "Run custom command: " + platform.SafeText(label), Shortcut: definition.Binding, Enabled: m.Discovery.Root != ""})
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

func (m *Model) reindexPalette() {
	if !m.PaletteMode {
		return
	}
	m.PaletteResults = commands.Search(m.paletteActions(), m.PaletteQuery)
	if m.PaletteSelected >= len(m.PaletteResults) {
		m.PaletteSelected = max(0, len(m.PaletteResults)-1)
	}
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

func (m Model) customCommandContext() customcmd.Context {
	selectedPath := string(m.Files.SelectedPath())
	if m.StatusTreeMode {
		if entry, ok := m.selectedStatusTreeEntry(); ok {
			selectedPath = string(entry.Path)
		} else {
			selectedPath = ""
		}
	}
	selectedSHA := ""
	if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
		selectedSHA = m.History.Rows[m.History.Selected].Commit.SHA
	}
	providerURL := ""
	if m.currentView() == workspace.GitHub {
		providerURL = m.GitHub.Pull.URL
	}
	options := map[string][]string{}
	for _, entry := range m.Files.Entries {
		options["paths"] = append(options["paths"], string(entry.Path))
	}
	for _, commit := range m.HistoryCommits {
		options["commits"] = append(options["commits"], commit.SHA)
	}
	for _, branch := range m.Branches.Entries {
		options["branches"] = append(options["branches"], branch.Name)
	}
	for _, tag := range m.TagSnapshot.Tags {
		options["tags"] = append(options["tags"], tag.Name)
	}
	for _, remote := range m.Remotes.Dashboard.Remotes {
		options["remotes"] = append(options["remotes"], remote.Name)
	}
	return customcmd.Context{RepositoryRoot: m.Discovery.Root, SelectedPath: selectedPath, SelectedSHA: selectedSHA, Branch: m.Snapshot.Branch.Name, ProviderURL: providerURL, PromptValues: m.CustomCommandPromptValues, OptionValues: options}
}

func (m *Model) runCustomCommand(name string) tea.Cmd {
	var definition *customcmd.Definition
	for index := range m.CustomCommands {
		if m.CustomCommands[index].Name == name {
			definition = &m.CustomCommands[index]
			break
		}
	}
	if definition == nil {
		m.Status = "custom command not found: " + platform.SafeText(name)
		return nil
	}
	if len(definition.Prompts) > 0 && m.CustomCommandPromptValues == nil {
		resolvedPrompts, resolveErr := customcmd.ResolvePrompts(definition.Prompts, m.customCommandContext())
		if resolveErr != nil {
			m.Status = "custom command form: " + platform.SafeText(resolveErr.Error())
			return nil
		}
		form, formErr := customcmd.NewForm(resolvedPrompts)
		if formErr != nil {
			m.Status = "custom command form: " + platform.SafeText(formErr.Error())
			return nil
		}
		m.CustomCommandForm, m.CustomCommandPending, m.State = &form, name, StateModal
		label := definition.Label
		if label == "" {
			label = definition.Name
		}
		m.Status = "custom command prompts: " + platform.SafeText(label)
		return nil
	}
	if len(definition.Contexts) > 0 {
		current := "status"
		switch m.currentView() {
		case workspace.Log:
			current = "history"
		case workspace.Compare:
			current = "compare"
		case workspace.GitHub:
			current = "github"
		}
		allowed := false
		for _, contextName := range definition.Contexts {
			if contextName == "any" || contextName == current {
				allowed = true
				break
			}
		}
		if !allowed {
			m.Status = "custom command unavailable in " + current + " context"
			return nil
		}
	}
	invocation, err := definition.Expand(m.customCommandContext())
	if err != nil {
		m.Status = "custom command: " + platform.SafeText(err.Error())
		return nil
	}
	if invocation.Confirm {
		if m.CustomCommandPromptValues == nil || m.CustomCommandPromptValues["__confirm"] != "true" {
			form, formErr := customcmd.NewForm([]customcmd.Prompt{{ID: "__confirm", Label: "Run this custom command?", Kind: customcmd.PromptConfirm}})
			if formErr != nil {
				m.Status = "custom command confirmation: " + platform.SafeText(formErr.Error())
				return nil
			}
			m.CustomCommandForm, m.CustomCommandPending, m.State = &form, name, StateModal
			m.Status = "confirm custom command: " + platform.SafeText(invocation.Label) + " (y/n)"
			return nil
		}
	}
	m.CustomCommandPromptValues = nil
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	operationID := fmt.Sprintf("customcmd-%s-%d", name, time.Now().UnixNano())
	generation, root, parent := m.repositoryGeneration, m.Discovery.Root, m.commandContext()
	var output customcmd.Output
	work := func(ctx context.Context) error {
		var runErr error
		output, runErr = customcmd.Run(ctx, invocation, int(m.DiffMaxBytes))
		return runErr
	}
	command := m.OperationEngine.Command(parent, operationID, root, "custom command "+name, invocation.Timeout, work)
	return func() tea.Msg {
		result := command()
		return CustomCommandFinishedMsg{Name: name, Repository: generation, Refresh: invocation.Refresh, Output: output, Err: result.Result.Err}
	}
}

func (m *Model) updateCustomCommandForm(key string) tea.Cmd {
	if m.CustomCommandForm == nil {
		return nil
	}
	event, err := m.CustomCommandForm.Handle(key)
	if err != nil {
		m.Status = "custom command form: " + platform.SafeText(err.Error())
		return nil
	}
	switch event {
	case customcmd.FormCancelled:
		m.CustomCommandForm, m.CustomCommandPending, m.CustomCommandPromptValues = nil, "", nil
		m.State, m.Status = StateReady, "custom command cancelled"
	case customcmd.FormSubmitted:
		name := m.CustomCommandPending
		values := m.CustomCommandForm.Values()
		m.CustomCommandForm, m.CustomCommandPending, m.CustomCommandPromptValues = nil, "", values
		m.State, m.Status = StateReady, "custom command ready"
		return m.runCustomCommand(name)
	default:
		position, total := m.CustomCommandForm.Progress()
		m.Status = fmt.Sprintf("custom command prompt %d/%d", position, total)
	}
	return nil
}

func (m *Model) executePaletteAction(id string) tea.Cmd {
	if command := m.PaletteCommands[id]; command != nil {
		return command()
	}
	if strings.HasPrefix(id, "repository_open_") || strings.HasPrefix(id, "repository_attention_") {
		prefix := "repository_open_"
		if strings.HasPrefix(id, "repository_attention_") {
			prefix = "repository_attention_"
		}
		index, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
		if err == nil && index >= 0 && index < len(m.Repositories.Rows) {
			m.Repositories.Selected = index
			if prefix == "repository_open_" {
				m.State, m.Status = StateOperationPending, "opening repository"
				return m.openSelectedRepository()
			}
			return m.navigate(workspace.Repositories, "Repositories")
		}
		return nil
	}
	if strings.HasPrefix(id, "customcmd:") {
		return m.runCustomCommand(strings.TrimPrefix(id, "customcmd:"))
	}
	for prefix, route := range map[string]workspace.View{"palette_branch_": workspace.Branches, "palette_commit_": workspace.Log, "palette_file_": workspace.Status} {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
		if err != nil || index < 0 {
			return nil
		}
		switch route {
		case workspace.Branches:
			if index >= len(m.Branches.Entries) {
				return nil
			}
			m.Branches.Selected = index
		case workspace.Log:
			if index >= len(m.History.Rows) {
				return nil
			}
			m.History.Selected = index
		case workspace.Status:
			if index >= len(m.Files.Visible) {
				return nil
			}
			m.Files.Selected = index
		}
		return m.navigate(route, string(route))
	}
	for prefix := range map[string]bool{"palette_pr_": true, "palette_issue_": true, "palette_release_": true, "palette_plugin_": true} {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
		if err != nil || index < 0 {
			return nil
		}
		switch prefix {
		case "palette_pr_":
			if index >= len(m.GitHub.Pulls) {
				return nil
			}
			m.GitHub.Pull = m.GitHub.Pulls[index]
			m.Status = fmt.Sprintf("selected GitHub pull request #%d", m.GitHub.Pull.Number)
			return m.navigate(workspace.GitHub, "GitHub")
		case "palette_issue_":
			if index >= len(m.GitHub.Issues) {
				return nil
			}
			m.GitHub.SelectedIssue = index
			m.Status = fmt.Sprintf("selected GitHub issue #%d", m.GitHub.Issues[index].Number)
			return m.navigate(workspace.GitHub, "GitHub")
		case "palette_release_":
			if index >= len(m.GitHub.Releases) {
				return nil
			}
			m.GitHub.SelectedRelease = index
			m.Status = "selected GitHub release " + platform.SafeText(m.GitHub.Releases[index].TagName)
			return m.navigate(workspace.GitHub, "GitHub")
		case "palette_plugin_":
			if index >= len(m.Plugins.Entries) {
				return nil
			}
			m.Plugins.Selected = index
			return m.navigate(workspace.Plugins, "Plugins")
		}
	}
	switch id {
	case "operation_recovery":
		operation := m.Snapshot.Operation
		if operation == nil {
			return nil
		}
		view, ok := recoveryWorkspaceRoute(operation.Kind())
		if !ok {
			return nil
		}
		if view == workspace.Bisect {
			return m.openBisectWorkspace()
		}
		return m.navigate(view, recoveryWorkspaceLabel(operation.Kind()))
	case "github_commit_selected":
		if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
			return nil
		}
		return m.openGitHubResource("commit", m.History.Rows[m.History.Selected].Commit.SHA)
	case "github_branch_selected":
		if m.Branches.Selected < 0 || m.Branches.Selected >= len(m.Branches.Entries) {
			return nil
		}
		return m.openGitHubResource("tree", m.Branches.Entries[m.Branches.Selected].Name)
	case "github_tag_selected":
		if selected, ok := m.selectedTag(); ok {
			return m.openGitHubResource("releases/tag", selected.Name)
		}
		return nil
	case "status":
		m.Workspace.Navigate(workspace.Status, "Status")
	case "gitignore":
		m.Workspace.Navigate(workspace.Gitignore, "Gitignore catalog")
		return m.openGitignore()
	case "branches":
		return m.navigate(workspace.Branches, "Branches")
	case "stashes":
		return m.navigate(workspace.Stashes, "Stashes")
	case "history":
		return m.navigate(workspace.Log, "History")
	case "tags":
		return m.navigate(workspace.Tags, "Tags")
	case "reflog":
		return m.navigate(workspace.Reflog, "Reflog")
	case "journal":
		m.JournalOffset = 0
		return m.navigate(workspace.Journal, "Operation journal")
	case "bisect":
		return m.openBisectWorkspace()
	case "clear_commit_basket":
		m.History.ClearBasket()
		return nil
	case "rebase":
		return m.openRebaseWorkspace()
	case "cherry_pick_recovery":
		m.Workspace.Navigate(workspace.CherryPick, "Cherry-pick progress")
		if len(m.Snapshot.Conflicts) > 0 {
			return m.loadConflictContent()
		}
		return nil
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

func (m *Model) openGitHubResource(kind, ref string) tea.Cmd {
	if m.GitHub.Repository.Owner == "" || m.GitHub.Repository.Name == "" || ref == "" {
		m.Status = "GitHub URL unavailable"
		return nil
	}
	host := m.GitHub.Repository.Host
	if host == "" {
		host = "github.com"
	}
	resourceURL := url.URL{Scheme: "https", Host: host, Path: "/" + m.GitHub.Repository.Owner + "/" + m.GitHub.Repository.Name + "/" + kind + "/" + url.PathEscape(ref)}
	command, err := platform.OpenURLCommand(resourceURL.String())
	if err != nil {
		m.Status = "GitHub URL unavailable: " + platform.SafeText(err.Error())
		return nil
	}
	m.Status = "opening GitHub " + kind + " " + platform.SafeText(ref)
	return tea.ExecProcess(command, nil)
}

func (m Model) activeGitignoreCatalog() (*catalog.Catalog, error) {
	if m.GitignoreCatalog != nil {
		return m.GitignoreCatalog, nil
	}
	return catalog.Default()
}

func (m Model) openGitignore() tea.Cmd {
	root := m.Discovery.Root
	generation := m.repositoryGeneration
	return func() tea.Msg {
		cat, err := m.activeGitignoreCatalog()
		if err != nil {
			return GitignoreReadyMsg{Err: err, Generation: generation}
		}
		content, missing, readErr := security.ReadDocument(root, m.GitignoreMaxBytes)
		if readErr != nil {
			return GitignoreReadyMsg{Err: readErr, ReadOnly: true, Generation: generation}
		}
		doc, parseErr := document.Parse(content)
		if parseErr != nil {
			return GitignoreReadyMsg{Err: parseErr, Generation: generation}
		}
		results := match.Match(doc, cat)
		model := gitignoreview.New(domain.RepositoryID(root), cat, results)
		report, recommendErr := recommend.Recommend(root, cat, recommend.Options{})
		if recommendErr == nil {
			model.SetRecommendations(report.Recommendations)
		}
		model.SetSize(m.Width, m.Height)
		return GitignoreReadyMsg{Model: model, Missing: missing, Generation: generation}
	}
}

func (m Model) refreshGitignoreCatalog() tea.Cmd {
	generation := m.repositoryGeneration
	return func() tea.Msg {
		cachePath, err := catalog.DefaultCacheDir()
		if err != nil {
			return GitignoreCatalogReadyMsg{Generation: generation, Err: err}
		}
		result, err := catalog.Refresh(m.commandContext(), catalog.RefreshConfig{CachePath: cachePath})
		return GitignoreCatalogReadyMsg{Source: result.Source, Generation: generation, Err: err}
	}
}

func (m Model) previewGitignoreCreate() tea.Cmd {
	root, ids, generation := m.Discovery.Root, selectedGitignoreIDs(m.Gitignore), m.repositoryGeneration
	return func() tea.Msg {
		cat, err := catalog.Default()
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		snapshot, err := domain.NewDocumentSnapshot(domain.RepositoryID(root), root, ".gitignore", nil, 0644)
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		plan, err := manage.PlanCreateTemplates(snapshot, cat, ids)
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		preview := manage.PreviewPlan(plan)
		return GitignoreCreatePreviewMsg{Plan: plan, Text: preview.Diff + "\nselected templates: " + strings.Join(templateIDStrings(ids), ", "), Repository: generation}
	}
}

func (m Model) previewGitignoreMutation(action string) tea.Cmd {
	root, ids, generation := m.Discovery.Root, selectedGitignoreIDs(m.Gitignore), m.repositoryGeneration
	return func() tea.Msg {
		cat, err := catalog.Default()
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		path, targetErr := security.Target(root)
		if targetErr != nil {
			return GitignoreCreatePreviewMsg{Err: targetErr, Repository: generation}
		}
		info, err := os.Lstat(path)
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return GitignoreCreatePreviewMsg{Err: domain.ErrUnsafeTarget, Repository: generation}
		}
		content, missing, err := security.ReadDocument(root, m.GitignoreMaxBytes)
		if err != nil || missing {
			if err == nil {
				err = domain.ErrConcurrentModification
			}
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		snapshot, err := domain.NewDocumentSnapshot(domain.RepositoryID(root), root, ".gitignore", content, uint32(info.Mode().Perm()))
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		var plan domain.MutationPlan
		switch action {
		case "add":
			plan, err = manage.PlanAddTemplates(snapshot, cat, ids)
		case "remove":
			plan, err = manage.PlanRemoveTemplates(snapshot, cat, ids)
		case "update":
			plan, err = manage.PlanUpdateTemplates(snapshot, cat, ids)
		case "adopt":
			if len(ids) != 1 {
				err = errors.New("adopt requires exactly one selected template")
			} else {
				plan, err = manage.PlanAdoptTemplate(snapshot, cat, ids[0])
			}
		default:
			err = errors.New("unknown gitignore mutation")
		}
		if err != nil {
			return GitignoreCreatePreviewMsg{Err: err, Repository: generation}
		}
		preview := manage.PreviewPlan(plan)
		return GitignoreCreatePreviewMsg{Plan: plan, Text: preview.Diff + "\nselected templates: " + strings.Join(templateIDStrings(ids), ", "), Repository: generation}
	}
}

func (m Model) executeGitignoreMutation(plan domain.MutationPlan, action string) tea.Cmd {
	generation := m.repositoryGeneration
	return func() tea.Msg {
		return GitignoreMutationFinishedMsg{Action: action, Repository: generation, Err: manage.Apply(plan)}
	}
}

func selectedGitignoreIDs(model gitignoreview.RepositoryModel) []domain.TemplateID {
	entries := model.SelectedEntries()
	ids := make([]domain.TemplateID, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.Template.ID)
	}
	return ids
}

func templateIDStrings(ids []domain.TemplateID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
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
	m.CustomCommands = append([]customcmd.Definition(nil), c.CustomCommands...)
	m.EditorTool = platform.ExternalTool{Executable: c.Tools.Editor.Executable, Args: append([]string(nil), c.Tools.Editor.Args...)}
	m.OpenerTool = platform.ExternalTool{Executable: c.Tools.Opener.Executable, Args: append([]string(nil), c.Tools.Opener.Args...)}
	m.Difftool = platform.ExternalTool{Executable: c.Tools.Difftool.Executable, Args: append([]string(nil), c.Tools.Difftool.Args...)}
	m.GitignoreMaxBytes = c.GitignoreMaxBytes
	if m.GitignoreMaxBytes <= 0 {
		m.GitignoreMaxBytes = security.DefaultMaxDocumentBytes
	}
	m.CommitTreeEnabled, m.CommitTreeMaxCommits = c.ShowCommitTree, c.CommitTree.MaxCommits
	m.PaletteMaxResults, m.StatusOverscan = c.Workspace.PaletteMaxResults, c.Workspace.StatusOverscan
	if requested, ok := watch.ParseMode(c.Watch); ok {
		m.WatchRequested = requested
	}
	m.Keymap = mergeKeymap(config.EffectiveKeymap(c))
	m.GitHubEnabled, m.GitHubTokenEnv = c.GitHub.Enabled, c.GitHub.TokenEnv
	m.GitHubCache = provider.NewPullRequestCache(c.GitHub.CacheTTL)
	m.GitHubPullsCache = provider.NewCache[[]provider.PullRequest](c.GitHub.CacheTTL)
	m.GitHubDetailsCache = provider.NewCache[provider.PullRequestDetail](c.GitHub.CacheTTL)
	m.GitHubCommentsCache = provider.NewCache[[]provider.ReviewComment](c.GitHub.CacheTTL)
	m.GitHubChecksCache = provider.NewCache[provider.ChecksSnapshot](c.GitHub.CacheTTL)
	m.GitHubReviewsCache = provider.NewCache[provider.ReviewSnapshot](c.GitHub.CacheTTL)
	m.GitHubIssuesCache = provider.NewCache[[]provider.Issue](c.GitHub.CacheTTL)
	m.GitHubReleasesCache = provider.NewCache[[]provider.Release](c.GitHub.CacheTTL)
	m.PluginsEnabled, m.PluginDirectories = c.Plugins.Enabled, append([]string(nil), c.Plugins.Directories...)
	m.PluginOutputLimit = c.Plugins.MaxOutput
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
	m.History.SetASCII(m.Theme.Colorless)
	m.PanelSplit = layout.Split{FilesPercent: c.Layout.FilesPercent, DetailsPercent: c.Layout.DetailsPercent}
	m.RepositoryRoots = append([]string(nil), c.Repositories.Roots...)
	m.RepositoryGroups = cloneGroups(c.Repositories.Groups)
	m.RepositoryMaxDepth, m.RepositoryMaxCount = c.Repositories.MaxDepth, c.Repositories.MaxRepositories
	m.RepositoryIgnoreDirs = append([]string(nil), c.Repositories.IgnoreDirs...)
	if path, err := registry.StatePath(); err == nil {
		m.RepositoryRegistryPath = path
	}
	m.RepositoryEngine = registry.NewEngine(c.Remote.Workers)
	autoFetchEnabled, autoFetchInterval, autoFetchJitter := c.Remote.AutoFetch, c.Remote.AutoFetchInterval, c.Remote.AutoFetchJitter
	autoFetchBackoff, autoFetchBackoffMax := c.Remote.AutoFetchBackoff, c.Remote.AutoFetchBackoffMax
	if profile, ok := c.Remote.AutoFetchProfiles[c.Profile]; ok {
		autoFetchEnabled, autoFetchInterval, autoFetchJitter = profile.Enabled, profile.Interval, profile.Jitter
		autoFetchBackoff, autoFetchBackoffMax = profile.Backoff, profile.BackoffMax
	}
	m.AutoFetchEnabled = autoFetchEnabled
	m.AutoFetchScheduler = remoteintel.New(remoteintel.Config{
		Enabled:        autoFetchEnabled,
		Interval:       autoFetchInterval,
		Jitter:         autoFetchJitter,
		BackoffBase:    autoFetchBackoff,
		BackoffMax:     autoFetchBackoffMax,
		Workers:        c.Remote.Workers,
		GroupIntervals: cloneRefreshPolicies(c.Repositories.GroupAutoFetch),
	})
	m.AutoFetchResults = make(map[string]remoteintel.Result)
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
	m.DiffAutoPreviewed = false
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
	m.TagSnapshot, m.TagsLoading, m.TagsErr, m.TagsSelected, m.TagsFilter, m.TagsFilterMode = tags.Snapshot{}, false, nil, 0, "", false
	m.Submodules, m.SubmodulesLoading, m.SubmodulesGeneration, m.SubmodulesErr = submodules.Snapshot{}, false, 0, nil
	m.SubmoduleAction, m.SubmodulePath, m.SubmoduleInput, m.SubmoduleURL = "", "", "", ""
	if m.BulkSubmoduleCancel != nil {
		m.BulkSubmoduleCancel()
	}
	m.BulkSubmoduleAction, m.BulkSubmodulePaths, m.BulkSubmoduleOutcome, m.BulkSubmoduleCancel = "", nil, nil, nil
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
	previousOperation := m.Snapshot.Operation != nil
	previousOperationKind := sequencer.KindUnknown
	if m.Snapshot.Operation != nil {
		previousOperationKind = m.Snapshot.Operation.Kind()
	}
	previousConflicted := m.Snapshot.Counts.Conflicted
	wasConflictView := m.recoveryWorkspace()
	if m.ActivityLog != nil && !m.Snapshot.ObservedAt.IsZero() {
		for _, event := range history.Diff(m.Snapshot, snapshot) {
			m.ActivityLog.Add(event)
		}
	}
	if m.DiffPath != "" && !m.StatusCommitActive && !snapshotContainsPath(snapshot.Entries, m.DiffPath) {
		m.closeDiff()
	}
	m.Snapshot = snapshot
	if m.ConflictContentCancel != nil {
		m.ConflictContentCancel()
		m.ConflictContentCancel = nil
	}
	m.ConflictContentLoading = false
	operationKind, operationTarget := sequencer.KindUnknown, ""
	if snapshot.Operation != nil {
		operationKind, operationTarget = snapshot.Operation.Kind(), snapshot.Operation.Target()
	}
	m.Conflict.SetSnapshot(operationKind, operationTarget, snapshot.Conflicts)
	m.Conflict.SetOperationState(snapshot.Operation)
	m.Conflict.SetStagedCount(snapshot.Counts.Staged)
	if m.HistoryRevertRunning && operationKind == sequencer.KindRevert {
		if view, ok := recoveryWorkspaceRoute(operationKind); ok {
			m.Workspace.Navigate(view, recoveryWorkspaceLabel(operationKind))
		}
		m.Status = "revert paused for conflict recovery"
	}
	if snapshot.Operation == nil {
		m.HistoryRevertRunning = false
	}
	if previousOperation && wasConflictView && snapshot.Operation == nil && len(snapshot.Conflicts) == 0 {
		m.Workspace.Navigate(workspace.Status, "Status")
		m.Status = "sequencer operation completed or was aborted externally"
	}
	if err := m.History.SetScope(snapshot.Root, snapshot.Branch.Name, m.repositoryGeneration); err != nil {
		m.Status = "history selection: " + err.Error()
	}
	if !m.StatusCommitActive {
		m.Files.SetEntries(snapshot.Entries)
		m.rebuildStatusFileTree()
	}
	if snapshot.Counts.Conflicted > 0 && (previousConflicted == 0 || previousOperationKind != operationKind) {
		m.notify(notifications.Conflict, notifications.Error, "repository conflicts", fmt.Sprintf("%d conflicted file(s)", snapshot.Counts.Conflicted), true)
	}
}

func recoverableOperation(kind sequencer.Kind) bool {
	_, ok := recoveryWorkspaceRoute(kind)
	return ok
}

func recoveryWorkspaceRoute(kind sequencer.Kind) (workspace.View, bool) {
	route, ok := sequencer.RouteFor(kind)
	if !ok {
		return "", false
	}
	switch route.View {
	case sequencer.RecoveryCherryPickView:
		return workspace.CherryPick, true
	case sequencer.RecoveryBisectView:
		return workspace.Bisect, true
	default:
		return workspace.Conflict, true
	}
}

func recoveryWorkspaceLabel(kind sequencer.Kind) string {
	if route, ok := sequencer.RouteFor(kind); ok {
		return route.Label
	}
	return "Recovery"
}

func (m *Model) updateBisectKey(key string) tea.Cmd {
	if m.BisectRunMode != "" || m.BisectRunConfirm {
		switch key {
		case "esc":
			m.BisectRunMode, m.BisectRunInput, m.BisectRunConfirm = "", "", false
			m.BisectRunArgs = nil
			m.Status = "automated bisect run cancelled"
		case "backspace":
			m.BisectRunInput = removeLastRune(m.BisectRunInput)
		case "space", " ":
			if m.BisectRunMode != "" {
				m.BisectRunInput += " "
			}
		case "enter":
			if m.BisectRunMode == "executable" {
				if strings.TrimSpace(m.BisectRunInput) == "" {
					m.Status = "an executable is required"
				} else {
					m.BisectRunExecutable, m.BisectRunInput, m.BisectRunMode = m.BisectRunInput, "", "arg"
					m.Status = "argument 1 (enter blank to run): "
				}
			} else if m.BisectRunMode == "arg" {
				if m.BisectRunInput == "" {
					m.BisectRunMode, m.BisectRunConfirm = "", true
					m.Status = "run " + m.BisectRunExecutable + " with " + fmt.Sprintf("%d", len(m.BisectRunArgs)) + " argument(s)? (y/n)"
				} else {
					m.BisectRunArgs = append(m.BisectRunArgs, m.BisectRunInput)
					m.BisectRunInput = ""
					m.Status = "argument " + fmt.Sprintf("%d", len(m.BisectRunArgs)+1) + " (enter blank to run): "
				}
			} else if m.BisectRunConfirm {
				m.Status = "confirm automated bisect run with y/n"
			}
		case "y", "Y":
			if m.BisectRunConfirm {
				m.BisectRunConfirm, m.State, m.Status, m.BisectRunOutput = false, StateOperationPending, "running automated bisect", ""
				return m.bisectRun()
			}
			if m.BisectRunMode != "" {
				m.BisectRunInput += key
			}
		case "n", "N":
			if m.BisectRunConfirm {
				m.BisectRunMode, m.BisectRunInput, m.BisectRunConfirm = "", "", false
				m.BisectRunArgs = nil
				m.Status = "automated bisect run cancelled"
			} else {
				m.BisectRunInput += key
			}
		default:
			if m.BisectRunMode != "" && len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
				m.BisectRunInput += key
			}
		}
		if m.BisectRunMode == "executable" {
			m.Status = "executable: " + m.BisectRunInput
		}
		return nil
	}

	if m.BisectStartMode != "" || m.BisectStartConfirm {
		switch key {
		case "esc":
			m.BisectStartMode, m.BisectStartInput, m.BisectStartConfirm = "", "", false
			m.Status = "bisect start cancelled"
		case "backspace":
			m.BisectStartInput = removeLastRune(m.BisectStartInput)
		case "enter":
			ref := strings.TrimSpace(m.BisectStartInput)
			if m.BisectStartMode != "" {
				if ref == "" {
					m.Status = "a ref is required"
				} else if m.BisectStartMode == "bad" {
					m.BisectStartBad, m.BisectStartInput, m.BisectStartMode = ref, "", "good"
					m.Status = "known-good ref: "
				} else {
					m.BisectStartGood, m.BisectStartInput, m.BisectStartMode, m.BisectStartConfirm = ref, "", "", true
					m.Status = "start bisect bad=" + m.BisectStartBad + " good=" + m.BisectStartGood + "? (y/n)"
				}
			} else if m.BisectStartConfirm {
				m.Status = "confirm start bisect with y/n"
			}
		case "y", "Y":
			if m.BisectStartConfirm {
				m.BisectStartConfirm, m.State, m.Status = false, StateOperationPending, "starting bisect"
				return m.bisectStart()
			}
		case "n", "N":
			m.BisectStartMode, m.BisectStartInput, m.BisectStartConfirm = "", "", false
			m.Status = "bisect start cancelled"
		default:
			if m.BisectStartMode != "" && len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00 ") {
				m.BisectStartInput += key
			}
		}
		if m.BisectStartMode != "" {
			m.Status = "known-" + m.BisectStartMode + " ref: " + m.BisectStartInput
		}
		return nil
	}
	switch key {
	case "g":
		m.State, m.Status = StateOperationPending, "marking candidate good"
		return m.bisectAction(bisect.Good)
	case "b":
		m.State, m.Status = StateOperationPending, "marking candidate bad"
		return m.bisectAction(bisect.Bad)
	case "s":
		m.State, m.Status = StateOperationPending, "skipping candidate"
		return m.bisectAction(bisect.Skip)
	case "x":
		m.BisectResetConfirm = true
		m.Status = "reset bisect and return to the original branch? (y/n)"
	case "S":
		m.BisectStartMode, m.BisectStartInput = "bad", ""
		m.Status = "known-bad ref: "
	case "A":
		m.BisectRunMode, m.BisectRunInput, m.BisectRunArgs = "executable", "", nil
		m.Status = "executable: "
	case "y":
		if m.BisectResetConfirm {
			m.BisectResetConfirm, m.State, m.Status = false, StateOperationPending, "resetting bisect"
			return m.bisectReset()
		}
	case "n", "esc":
		if m.BisectResetConfirm {
			m.BisectResetConfirm = false
			m.Status = "bisect reset cancelled"
		} else if key == "esc" {
			m.Workspace.Back()
		}
	case "1":
		m.Workspace.Navigate(workspace.Status, "Status")
	case "r":
		m.State, m.Status = StateOperationPending, "refreshing bisect state"
		return m.loadBisectState()
	case "i":
		if m.Bisect.Candidate == "" {
			m.Status = "no bisect candidate is available"
			return nil
		}
		m.State, m.Status = StateOperationPending, "loading bisect candidate details"
		return m.inspectCommit(history.Commit{SHA: m.Bisect.Candidate, Short: shortSHA(m.Bisect.Candidate)}, "", "")
	}
	return nil
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
	m.recordActivityWithOperation(kind, path, message, nil)
}

func attachLatestRecoveryPoint(ctx context.Context, runner git.Runner, operation *history.OperationRecord) {
	if operation == nil {
		return
	}
	entries, err := reflog.Load(ctx, runner, reflog.Request{Ref: "HEAD", Limit: 1})
	if err != nil || len(entries) == 0 {
		return
	}
	entry := entries[0]
	operation.RecoverySHA = entry.SHA
	operation.RecoveryRef = entry.Selector
	operation.RecoverySubject = platform.SafeText(entry.Subject)
}

func (m *Model) recordActivityWithOperation(kind history.Kind, path, message string, operation *history.OperationRecord) {
	if m.ActivityLog != nil {
		event := history.Event{At: time.Now(), Kind: kind, Path: path, Message: message}
		if kind == history.OperationSuccess || kind == history.OperationFailure {
			outcome := "failure"
			if kind == history.OperationSuccess {
				outcome = "success"
			}
			if operation == nil {
				operation = &history.OperationRecord{Kind: string(kind), Target: path}
			} else {
				copy := *operation
				operation = &copy
			}
			if operation.Repository == "" {
				operation.Repository = m.Discovery.Root
			}
			if operation.Kind == "" {
				operation.Kind = string(kind)
			}
			if operation.Target == "" {
				operation.Target = path
			}
			operation.Args = history.RedactArgs(operation.Args)
			operation.Outcome = outcome
			event.Operation = operation
		}
		m.ActivityLog.Add(event)
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
	commands := make([]tea.Cmd, 0, 6)
	if m.Discovery.Root != "" {
		commands = append(commands, m.refresh(), m.tick(), m.startWatcher())
	}
	if m.AutoFetchEnabled {
		commands = append(commands, m.autoFetchTick())
	}
	if len(commands) == 0 {
		return nil
	}
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

func (m Model) autoFetchTick() tea.Cmd {
	if !m.AutoFetchEnabled || m.AutoFetchScheduler == nil {
		return nil
	}
	interval := m.AutoFetchScheduler.Config().Interval
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return AutoFetchTickMsg{At: t} })
}

type AutoFetchTickMsg struct{ At time.Time }

func (m *Model) runAutoFetch() tea.Cmd {
	if !m.AutoFetchEnabled || m.AutoFetchScheduler == nil || m.AutoFetchRunning {
		return nil
	}
	repositories := append([]registry.Repository(nil), m.RepositoryRegistry...)
	if len(repositories) == 0 && m.Discovery.Root != "" {
		repositories = []registry.Repository{{Path: m.Discovery.Root, Name: filepath.Base(m.Discovery.Root)}}
	}
	if len(repositories) == 0 {
		return nil
	}
	requests := make([]remoteintel.Repository, 0, len(repositories))
	for _, repository := range repositories {
		requests = append(requests, remoteintel.Repository{Path: repository.Path, Groups: append([]string(nil), repository.Groups...), ActiveOperation: repository.Path == m.Discovery.Root && m.Snapshot.Operation != nil})
	}
	ctx := m.commandContext()
	scheduler := m.AutoFetchScheduler
	m.AutoFetchRunning = true
	return func() tea.Msg {
		results := scheduler.RunOnce(ctx, requests, time.Now(), func(ctx context.Context, path string) error {
			discovery, err := git.Discover(ctx, path)
			if err != nil {
				return err
			}
			snapshot, err := git.Snapshot(ctx, discovery, 0)
			if err != nil {
				return err
			}
			if snapshot.Operation != nil {
				return remoteintel.ErrActiveOperation
			}
			entries, err := remotes.List(ctx, git.NewRunner(discovery.Root))
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				return errors.New("repository has no configured remote")
			}
			_, err = remotes.Fetch(ctx, git.NewRunner(discovery.Root), entries[0].Name)
			return err
		})
		return AutoFetchFinishedMsg{Results: results}
	}
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
	generation := m.repositoryGeneration
	return func() tea.Msg {
		snapshot, err := git.Snapshot(ctx, discovery, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Generation: generation, Snapshot: snapshot}
	}
}

// loadSubmodules runs independently of the authoritative status refresh so a
// slow nested repository cannot delay the core worktree snapshot.
func (m *Model) loadSubmodules(generation uint64) tea.Cmd {
	if generation == 0 || m.Discovery.Root == "" || m.SubmodulesLoading {
		return nil
	}
	if generation == m.SubmodulesGeneration && m.SubmodulesGeneration != 0 {
		return nil
	}
	m.SubmodulesLoading = true
	m.SubmodulesGeneration = generation
	root, ctx := m.Discovery.Root, m.commandContext()
	runner := git.NewRunner(root)
	return func() tea.Msg {
		snapshot, err := submodules.Load(ctx, runner, submodules.LoadRequest{Repository: root})
		return SubmodulesReadyMsg{Generation: generation, Snapshot: snapshot, Err: err}
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
	if m.StatusTreeMode {
		if _, ok := m.selectedStatusTreeEntry(); !ok {
			return nil
		}
	}
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
	if m.StatusTreeMode {
		if _, ok := m.selectedStatusTreeEntry(); !ok {
			m.Status = "select a file row before restoring"
			return
		}
	}
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
		m.rebuildStatusFileTree()
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
	m.rebuildStatusFileTree()
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
	if m.StatusTreeMode {
		if _, ok := m.selectedStatusTreeEntry(); !ok {
			return nil
		}
	}
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	return m.openDiffMode(e.Staged && !e.Unstaged)
}

// previewSelectedStatusDiff opens the first status selection once per
// repository. Later refreshes do not reopen a diff the user explicitly
// closed, while keyboard selection continues to preview every changed file.
func (m *Model) previewSelectedStatusDiff() tea.Cmd {
	if m.DiffAutoPreviewed || m.currentView() != workspace.Status || m.StatusCommitActive || len(m.Files.Visible) == 0 {
		return nil
	}
	m.DiffAutoPreviewed = true
	return m.openDiff()
}

func (m *Model) openDiffMode(staged bool) tea.Cmd {
	if m.StatusTreeMode {
		if _, ok := m.selectedStatusTreeEntry(); !ok {
			return nil
		}
	}
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

func (m *Model) openExternalTool(name string, tool platform.ExternalTool, values map[string]string) tea.Cmd {
	command, err := tool.CommandValues(values, m.Discovery.Root, nil)
	if err != nil {
		m.Status = name + ": " + err.Error()
		return nil
	}
	return m.openExternalProcess(name, command)
}

func (m *Model) openExternalProcess(name string, command *platform.Command) tea.Cmd {
	return m.openExternalProcessWithCleanup(name, command, nil)
}

func (m *Model) openExternalProcessWithCleanup(name string, command *platform.Command, cleanup func()) tea.Cmd {
	m.State, m.Status = StateOperationPending, name+" active"
	generation := m.repositoryGeneration
	return tea.ExecProcess(command, func(processErr error) tea.Msg {
		if cleanup != nil {
			cleanup()
		}
		return ExternalToolFinishedMsg{Name: name, Repository: generation, Err: processErr}
	})
}

func (m *Model) prepareCompareExternalTool(name string, tool platform.ExternalTool) tea.Cmd {
	if tool.Executable == "" {
		m.Status = name + ": configure a tool before opening historical content"
		return nil
	}
	if m.Compare.Selected < 0 || m.Compare.Selected >= len(m.Compare.Result.Changes) {
		m.Status = name + ": select a changed path first"
		return nil
	}
	change := m.Compare.Result.Changes[m.Compare.Selected]
	path := change.NewPath
	if path == "" {
		path = change.OldPath
	}
	if path == "" {
		m.Status = name + ": selected change has no path"
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	commit := m.Compare.Result.Left.SHA
	ctx := m.commandContext()
	m.State, m.Status = StateOperationPending, "loading historical file for "+name
	return func() tea.Msg {
		content, err := runner.ShowPath(ctx, commit, []byte(path), int(m.DiffMaxBytes))
		return HistoricalToolReadyMsg{Name: name, Repository: generation, Tool: tool, Path: path, Content: content, Err: err}
	}
}

func (m *Model) openSelectedExternalTool(name string, tool platform.ExternalTool) tea.Cmd {
	if m.StatusTreeMode {
		entry, ok := m.selectedStatusTreeEntry()
		if !ok {
			m.Status = name + ": select a file row"
			return nil
		}
		return m.openExternalTool(name, tool, map[string]string{"path": string(entry.Path), "repo": m.Discovery.Root})
	}
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		m.Status = name + ": no file selected"
		return nil
	}
	path := string(m.Files.Entries[m.Files.Visible[m.Files.Selected]].Path)
	return m.openExternalTool(name, tool, map[string]string{"path": path, "repo": m.Discovery.Root})
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
	m.rebuildStatusFileTree()
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

func (m Model) selectedCompareRef() (string, bool) {
	switch m.currentView() {
	case workspace.Log:
		if m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
			return m.History.Rows[m.History.Selected].Commit.SHA, true
		}
	case workspace.Tags:
		if selected, ok := m.selectedTag(); ok {
			return selected.Name, true
		}
	case workspace.Branches:
		if m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
			return m.Branches.Entries[m.Branches.Selected].Name, true
		}
	case workspace.Reflog:
		if selected, ok := m.Reflog.SelectedEntry(); ok {
			return selected.SHA, true
		}
	case workspace.Remotes:
		if m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) && m.Snapshot.Branch.Name != "" {
			remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
			if m.CompareLeft == "" {
				return m.Snapshot.Branch.Name, true
			}
			return remote + "/" + m.Snapshot.Branch.Name, true
		}
	}
	return "", false
}

func (m *Model) assignCompareSelection() tea.Cmd {
	ref, ok := m.selectedCompareRef()
	if !ok {
		m.Status = "select a revision before assigning comparison side"
		return nil
	}
	if m.CompareLeft == "" {
		m.CompareLeft = ref
		m.Status = "comparison A set to " + platform.SafeText(ref) + "; select B and press Y"
		return nil
	}
	if m.CompareRight == "" && ref != m.CompareLeft {
		m.CompareRight = ref
		return m.startCompare()
	}
	m.CompareLeft, m.CompareRight = ref, ""
	m.Status = "comparison A reset to " + platform.SafeText(ref) + "; select B and press Y"
	return nil
}

func (m *Model) startCompare() tea.Cmd {
	if m.CompareLeft == "" || m.CompareRight == "" {
		return nil
	}
	if m.CompareCancel != nil {
		m.CompareCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.CompareCancel = cancel
	m.CompareRequest++
	m.CompareGeneration = m.repositoryGeneration
	request, generation := m.CompareRequest, m.CompareGeneration
	left, right := m.CompareLeft, m.CompareRight
	runner := git.NewRunner(m.Discovery.Root)
	m.CompareLoading, m.CompareErr, m.State, m.Status = true, nil, StateOperationPending, "comparing revisions"
	m.Workspace.Navigate(workspace.Compare, "Comparison")
	return func() tea.Msg {
		result, err := compareops.Compare(ctx, runner, compareops.Request{Left: left, Right: right, MaxFiles: compareops.DefaultMaxFiles, MaxPatchBytes: int(m.DiffMaxBytes)})
		return CompareReadyMsg{Generation: generation, Request: request, Result: result, Err: err}
	}
}

func (m *Model) selectCompareRemote() bool {
	refs := []string{m.CompareLeft, m.CompareRight}
	for index, remote := range m.Remotes.Dashboard.Remotes {
		for _, ref := range refs {
			if strings.HasPrefix(ref, remote.Name+"/") {
				m.Remotes.Selected = index
				return true
			}
		}
	}
	return false
}

func (m *Model) loadComparePatch() tea.Cmd {
	if m.Compare.Selected < 0 || m.Compare.Selected >= len(m.Compare.Result.Changes) {
		m.Status = "select a changed path first"
		return nil
	}
	if m.ComparePatchCancel != nil {
		m.ComparePatchCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.ComparePatchCancel = cancel
	m.ComparePatchRequest++
	request, generation := m.ComparePatchRequest, m.repositoryGeneration
	change := m.Compare.Result.Changes[m.Compare.Selected]
	path := change.NewPath
	if path == "" {
		path = change.OldPath
	}
	result := m.Compare.Result
	runner := git.NewRunner(m.Discovery.Root)
	m.ComparePatchLoading, m.Status = true, "loading comparison patch for "+platform.SafeText(path)
	return func() tea.Msg {
		text, truncated, err := compareops.FilePatch(ctx, runner, result, change, int(m.DiffMaxBytes))
		return ComparePatchReadyMsg{Generation: generation, Request: request, Path: path, Text: text, Truncated: truncated, Err: err}
	}
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

func (m *Model) beginHistoricalHunks() {
	if m.HistoryInspector.Commit.SHA == "" || m.HistoryInspector.Diff == "" {
		m.Status = "selected commit has no editable patch"
		return
	}
	files, err := patch.Parse(m.HistoryInspector.Diff)
	if err != nil || len(files) == 0 {
		m.Status = "historical commit is not a selectable patch"
		return
	}
	m.Hunks = hunkview.New(files)
	m.HistoricalPatchMode = true
	m.HistoricalPatchTarget = m.HistoryInspector.Commit.SHA
	m.HistoricalPatchPath = ""
	if len(files) == 1 {
		m.HistoricalPatchPath = files[0].NewPath
	}
	m.Workspace.Navigate(workspace.Hunks, "Historical patch edit")
	m.Status = "select historical lines to remove, then press Enter"
}

func (m *Model) applyHistoricalPatch() tea.Cmd {
	if len(m.HistoricalPatch) == 0 {
		m.Status = "select historical lines before starting the edit"
		return nil
	}
	patchBytes := append([]byte(nil), m.HistoricalPatch...)
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	m.State, m.Status = StateOperationPending, "checking and applying historical patch edit"
	return func() tea.Msg {
		_, err := runner.ApplyReversePatchToWorktreeAndIndex(m.commandContext(), git.PartialPatch{Patch: patchBytes})
		return HistoricalPatchAppliedMsg{Repository: generation, Err: err}
	}
}

func (m *Model) beginHistoricalAmend() tea.Cmd {
	commitSubject := ""
	for _, commit := range m.HistoryCommits {
		if commit.SHA == m.HistoricalRebaseTarget {
			commitSubject = commit.Subject
			break
		}
	}
	cmd := m.beginCommit()
	m.Composer.Draft.Amend = true
	m.Composer.Draft.Subject = commitSubject
	m.Status = "rebase paused; review the historical patch edit (ctrl+x aborts)"
	return cmd
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

func (m Model) remoteBranchMutation(operation string, branch branches.Branch, localName, confirmation string) tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		var err error
		switch operation {
		case "checked out tracking":
			_, err = branches.CheckoutRemote(m.commandContext(), runner, branch.RemoteName, branch.RemoteBranch, localName)
		case "checked out detached":
			_, err = branches.CheckoutRemoteDetached(m.commandContext(), runner, branch.RemoteName, branch.RemoteBranch)
		case "deleted remote branch":
			_, err = branches.DeleteRemote(m.commandContext(), runner, branch.RemoteName, branch.RemoteBranch, confirmation)
		default:
			err = fmt.Errorf("unknown remote branch operation: %s", operation)
		}
		return BranchOperationFinishedMsg{Operation: operation, Name: branch.Name, Repository: generation, Err: err}
	}
}

func (m *Model) updateRemoteBranchKey(key string) tea.Cmd {
	if m.RemoteBranchAction != "track" {
		return nil
	}
	if key == "esc" {
		m.RemoteBranchAction, m.RemoteBranchInput, m.RemoteBranchTarget = "", "", branches.Branch{}
		m.Status = "remote branch checkout cancelled"
		return nil
	}
	switch key {
	case "backspace":
		m.RemoteBranchInput = removeLastRune(m.RemoteBranchInput)
	case "space":
		m.RemoteBranchInput += " "
	case "enter":
		name := strings.TrimSpace(m.RemoteBranchInput)
		if name == "" {
			m.Status = "local tracking branch name is required"
			return nil
		}
		branch := m.RemoteBranchTarget
		m.RemoteBranchAction, m.RemoteBranchInput, m.RemoteBranchTarget = "", "", branches.Branch{}
		m.State, m.Status = StateOperationPending, "checking out tracking branch "+platform.SafeText(name)
		return m.remoteBranchMutation("checked out tracking", branch, name, "")
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			m.RemoteBranchInput += key
		}
	}
	m.Status = "local tracking branch name: " + platform.SafeText(m.RemoteBranchInput)
	return nil
}

func (m Model) checkoutSelectedBranch() tea.Cmd {
	if m.Branches.Selected < 0 || m.Branches.Selected >= len(m.Branches.Entries) {
		return nil
	}
	branch := m.Branches.Entries[m.Branches.Selected]
	generation := m.repositoryGeneration
	if branch.Remote {
		return func() tea.Msg {
			return BranchOperationFinishedMsg{Name: branch.Name, Repository: generation, Err: fmt.Errorf("remote branch requires a local tracking name: %s", branch.Name)}
		}
	}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := branches.Checkout(m.commandContext(), runner, branch.Name)
		return BranchOperationFinishedMsg{Operation: "checked out", Name: branch.Name, Repository: generation, Err: err}
	}
}

func (m Model) githubCheckoutBranch() (branches.Branch, error) {
	if err := provider.ValidateCheckoutRef(m.GitHub.Pull.Head); err != nil {
		return branches.Branch{}, err
	}
	for _, remote := range m.Remotes.Dashboard.Remotes {
		candidate, ok := provider.ParseGitHubRemote(remote.FetchURL)
		if !ok || candidate.Owner != m.GitHub.Repository.Owner || candidate.Name != m.GitHub.Repository.Name {
			continue
		}
		return branches.Branch{Name: "remotes/" + remote.Name + "/" + m.GitHub.Pull.Head, Remote: true, RemoteName: remote.Name, RemoteBranch: m.GitHub.Pull.Head}, nil
	}
	return branches.Branch{}, errors.New("no matching GitHub remote is configured")
}

func (m Model) branchMutation(operation, name string, work func(context.Context, git.Runner) error) tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		return BranchOperationFinishedMsg{Operation: operation, Name: name, Repository: generation, Err: work(ctx, runner)}
	}
}

func (m Model) mergeSelectedBranch(strategy mergeops.Strategy) tea.Cmd {
	if m.BranchMergeTarget == "" || m.Snapshot.Branch.Name == "" {
		return nil
	}
	request := mergeops.Request{Repository: m.Discovery.Root, Generation: m.repositoryGeneration, Source: m.BranchMergeTarget, Strategy: strategy}
	runner := git.NewRunner(m.Discovery.Root)
	generation, target := m.repositoryGeneration, m.BranchMergeTarget
	mergeArgs := []string{"merge"}
	switch strategy {
	case mergeops.FastForwardOnly:
		mergeArgs = append(mergeArgs, "--ff-only")
	case mergeops.NoFastForward:
		mergeArgs = append(mergeArgs, "--no-ff")
	case mergeops.Squash:
		mergeArgs = append(mergeArgs, "--squash")
	}
	mergeArgs = append(mergeArgs, "--", target)
	operation := &history.OperationRecord{
		Repository: m.Discovery.Root,
		Kind:       "merge",
		Args:       history.RedactArgs(mergeArgs),
		Target:     target,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{m.Snapshot.Branch.Name, target},
	}
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	operationID := fmt.Sprintf("merge-%d-%s", generation, target)
	ctx := m.commandContext()
	var outcome mergeops.Outcome
	command := m.OperationEngine.Command(ctx, operationID, m.Discovery.Root, "merge "+target, 5*time.Minute, func(ctx context.Context) error {
		outcome = mergeops.Engine{Runner: runner, Repository: m.Discovery.Root, Generation: generation}.Execute(ctx, request)
		return outcome.Err
	})
	// Run the typed engine behind operations.Engine so repository serialization
	// and cancellation apply while retaining its rich paused/snapshot outcome.
	return func() tea.Msg {
		started := time.Now()
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		completed := *operation
		completed.Duration = time.Since(started)
		if outcome.Snapshot != nil {
			completed.NewHead = outcome.Snapshot.Branch.OID
		}
		attachLatestRecoveryPoint(ctx, runner, &completed)
		return MergeFinishedMsg{Repository: generation, Outcome: outcome, Operation: &completed}
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

func (m Model) fastForwardBranch(upstream string) tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := branches.FastForward(ctx, runner, upstream)
		return BranchOperationFinishedMsg{Operation: "fast-forwarded", Name: upstream, Repository: generation, Err: err}
	}
}

func (m Model) resetBranch(mode branches.ResetMode, target string) tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	label := "soft"
	if mode == branches.ResetMixed {
		label = "mixed"
	}
	return func() tea.Msg {
		_, err := branches.Reset(ctx, runner, mode, target)
		return BranchOperationFinishedMsg{Operation: "reset (" + label + ")", Name: target, Repository: generation, Err: err}
	}
}

func (m *Model) updateBranchResetKey(key string) tea.Cmd {
	if key == "esc" {
		m.BranchResetPrompt, m.BranchResetInput = false, ""
		m.Status = "branch reset cancelled"
		return nil
	}
	if key == "backspace" {
		m.BranchResetInput = removeLastRune(m.BranchResetInput)
	} else if key == "space" {
		m.BranchResetInput += " "
	} else if key == "enter" {
		parts := strings.Fields(m.BranchResetInput)
		if len(parts) != 2 {
			m.Status = "reset format: soft <ref> or mixed <ref>"
			return nil
		}
		var mode branches.ResetMode
		switch parts[0] {
		case "soft":
			mode = branches.ResetSoft
		case "mixed":
			mode = branches.ResetMixed
		default:
			m.Status = "reset mode must be soft or mixed"
			return nil
		}
		target := parts[1]
		m.BranchResetPrompt, m.BranchResetInput = false, ""
		m.State, m.Status = StateOperationPending, "resetting ("+parts[0]+") to "+platform.SafeText(target)
		return m.resetBranch(mode, target)
	} else if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
		m.BranchResetInput += key
	} else {
		return nil
	}
	m.Status = "reset: " + platform.SafeText(m.BranchResetInput) + " (soft <ref> or mixed <ref>)"
	return nil
}

func (m Model) deleteBranch(branch branches.Branch, force bool, input string) tea.Cmd {
	return m.branchMutation("deleted", branch.Name, func(ctx context.Context, r git.Runner) error {
		_, err := branches.Delete(ctx, r, branch, branches.DeletePrompt(branch.Name, force), input)
		return err
	})
}

func (m *Model) updateBranchMutationKey(key string) tea.Cmd {
	if !m.BranchCreateMode && !m.BranchRenameMode && !m.BranchUpstreamMode && !m.BranchDeleteMode && !m.BranchMergeMode {
		return nil
	}
	if key == "esc" {
		m.BranchCreateMode, m.BranchRenameMode, m.BranchUpstreamMode, m.BranchDeleteMode, m.BranchMergeMode = false, false, false, false, false
		m.BranchMutationInput, m.BranchRenameOld, m.BranchMergeTarget = "", "", ""
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
		renameMode, upstreamMode, mergeMode := m.BranchRenameMode, m.BranchUpstreamMode, m.BranchMergeMode
		m.BranchCreateMode, m.BranchRenameMode, m.BranchUpstreamMode, m.BranchDeleteMode, m.BranchMergeMode = false, false, false, false, false
		m.State = StateOperationPending
		switch {
		case mergeMode:
			strategy, ok := mergeStrategy(input)
			if !ok {
				m.BranchMergeMode = true
				m.State, m.Status = StateReady, "merge strategy must be merge, ff-only, no-ff, or squash"
				return nil
			}
			target := m.BranchMergeTarget
			m.BranchMutationInput, m.Status = "", "merging "+target
			return m.mergeSelectedBranch(strategy)
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
	} else if m.BranchMergeMode {
		label = "merge strategy"
	} else if m.BranchDeleteTarget.Name != "" {
		label = "type " + m.BranchDeleteTarget.Name + " to confirm"
	}
	m.Status = label + ": " + m.BranchMutationInput
	return nil
}

func mergeStrategy(value string) (mergeops.Strategy, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "merge", "regular":
		return mergeops.Regular, true
	case "ff-only", "ffonly":
		return mergeops.FastForwardOnly, true
	case "no-ff", "noff":
		return mergeops.NoFastForward, true
	case "squash":
		return mergeops.Squash, true
	default:
		return 0, false
	}
}

func (m Model) loadStashes() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := stash.List(m.commandContext(), r)
		return StashesReadyMsg{Entries: entries, Err: err}
	}
}

func (m *Model) loadReflog(appendPage bool) tea.Cmd {
	if m.Discovery.Root == "" || m.ReflogLoading {
		return nil
	}
	if !appendPage {
		m.ReflogSkip = 0
	}
	ref, skip, generation := m.Reflog.Ref, m.ReflogSkip, m.repositoryGeneration
	if ref == "" {
		ref = "HEAD"
	}
	m.ReflogLoading = true
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := reflog.Load(m.commandContext(), runner, reflog.Request{Ref: ref, Skip: skip})
		return ReflogReadyMsg{Entries: entries, Generation: generation, Skip: skip, HasMore: len(entries) == reflog.DefaultLimit, Err: err}
	}
}

func (m *Model) loadBisectState() tea.Cmd {
	if m.Discovery.Root == "" || m.BisectLoading {
		return nil
	}
	m.BisectLoading = true
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		state, err := bisect.Load(m.commandContext(), runner, m.Discovery.Root, generation)
		return BisectReadyMsg{Repository: generation, State: state, Err: err}
	}
}

func (m *Model) openBisectWorkspace() tea.Cmd {
	if m.Workspace == nil {
		m.Workspace = workspace.New()
	}
	m.Workspace.Navigate(workspace.Bisect, "Bisect")
	return m.loadBisectState()
}

func (m Model) bisectAction(action bisect.Mark) tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	id := fmt.Sprintf("bisect-%s-%d", action, generation)
	var outcome bisect.Outcome
	command := m.OperationEngine.Command(ctx, id, m.Discovery.Root, "bisect "+string(action), 5*time.Minute, func(ctx context.Context) error {
		outcome = bisect.MarkCandidate(ctx, runner, bisect.Request{Repository: m.Discovery.Root, Generation: generation, Mark: action})
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		return BisectFinishedMsg{Repository: generation, Action: string(action), Outcome: outcome}
	}
}

func (m Model) bisectStart() tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	request := bisect.StartRequest{Repository: m.Discovery.Root, Generation: generation, Bad: m.BisectStartBad, Good: m.BisectStartGood}
	var outcome bisect.Outcome
	command := m.OperationEngine.Command(ctx, fmt.Sprintf("bisect-start-%d", generation), m.Discovery.Root, "bisect start", 5*time.Minute, func(ctx context.Context) error {
		outcome = bisect.Start(ctx, runner, request)
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		return BisectFinishedMsg{Repository: generation, Action: "start", Outcome: outcome}
	}
}

func (m Model) bisectRun() tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	output := make(chan git.OutputChunk, 32)
	request := bisect.RunRequest{
		Repository: m.Discovery.Root, Generation: generation,
		Executable: m.BisectRunExecutable, Args: append([]string(nil), m.BisectRunArgs...),
		OnOutput: func(chunk git.OutputChunk) {
			select {
			case output <- chunk:
			default:
				// Display output is intentionally lossy and bounded; the final
				// typed result remains authoritative for the operation.
			}
		},
	}
	var outcome bisect.Outcome
	command := m.OperationEngine.Command(ctx, fmt.Sprintf("bisect-run-%d", generation), m.Discovery.Root, "bisect run", 30*time.Minute, func(ctx context.Context) error {
		outcome = bisect.RunCommand(ctx, runner, request)
		return outcome.Err
	})
	commandResult := func() tea.Msg {
		result := command()
		close(output)
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		return BisectFinishedMsg{Repository: generation, Action: "run", Outcome: outcome}
	}
	return tea.Batch(commandResult, waitBisectOutput(generation, output))
}

func waitBisectOutput(repository uint64, output <-chan git.OutputChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, open := <-output
		return BisectOutputMsg{Repository: repository, Chunk: chunk, Open: open, Output: output}
	}
}

func appendBisectDisplayOutput(current string, chunk git.OutputChunk) string {
	prefix := ""
	if chunk.Stream == git.StderrStream {
		prefix = "[stderr] "
	}
	value := current + prefix + platform.SafeText(string(chunk.Data))
	runes := []rune(value)
	if len(runes) > maxBisectDisplayOutput {
		runes = runes[len(runes)-maxBisectDisplayOutput:]
	}
	return string(runes)
}

func (m Model) bisectReset() tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	var outcome bisect.Outcome
	command := m.OperationEngine.Command(ctx, fmt.Sprintf("bisect-reset-%d", generation), m.Discovery.Root, "bisect reset", 5*time.Minute, func(ctx context.Context) error {
		outcome = bisect.Reset(ctx, runner, bisect.Request{Repository: m.Discovery.Root, Generation: generation})
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		return BisectFinishedMsg{Repository: generation, Action: "reset", Outcome: outcome}
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

func (m *Model) openPathHistory(path string, follow bool) tea.Cmd {
	if path == "" {
		m.Status = "select a file before opening path history"
		return nil
	}
	if m.PathHistoryCancel != nil {
		m.PathHistoryCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.PathHistoryCancel = cancel
	m.PathHistoryRequest++
	m.PathHistoryGeneration = m.repositoryGeneration
	request, generation := m.PathHistoryRequest, m.PathHistoryGeneration
	m.PathHistoryLoading, m.PathHistoryErr = true, nil
	m.PathHistory.SetPage(path, nil, false)
	m.PathHistory.Follow = follow
	m.State, m.Status = StateOperationPending, "loading history for "+platform.SafeText(path)
	m.Workspace.Navigate(workspace.PathHistory, "Path history: "+platform.SafeText(path))
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := pathhistory.LoadPage(ctx, runner, pathhistory.Request{Path: path, Follow: follow, Limit: pathhistory.DefaultLimit})
		return PathHistoryReadyMsg{Path: path, Entries: page.Entries, HasMore: page.HasMore, Follow: follow, Request: request, Generation: generation, Err: err}
	}
}

func (m *Model) loadPathHistoryPage(skip int) tea.Cmd {
	if m.PathHistory.Path == "" || m.PathHistoryLoading {
		return nil
	}
	if m.PathHistoryCancel != nil {
		m.PathHistoryCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.PathHistoryCancel = cancel
	m.PathHistoryRequest++
	m.PathHistoryGeneration = m.repositoryGeneration
	request, generation := m.PathHistoryRequest, m.PathHistoryGeneration
	m.PathHistoryLoading = true
	path, follow := m.PathHistory.Path, m.PathHistory.Follow
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := pathhistory.LoadPage(ctx, runner, pathhistory.Request{Path: path, Follow: follow, Skip: skip, Limit: pathhistory.DefaultLimit})
		return PathHistoryReadyMsg{Path: path, Entries: page.Entries, Skip: skip, HasMore: page.HasMore, Follow: follow, Request: request, Generation: generation, Err: err}
	}
}

func (m Model) inspectSelectedPathHistory() tea.Cmd {
	entry, ok := m.PathHistory.SelectedEntry()
	if !ok {
		return nil
	}
	path := entry.NewPath
	if path == "" {
		path = entry.OldPath
	}
	m.State, m.Status = StateOperationPending, "loading path commit details"
	m.Workspace.Navigate(workspace.Log, "History")
	return m.inspectCommit(entry.Commit, "", path)
}

func (m *Model) compareSelectedPathHistory() tea.Cmd {
	entry, ok := m.PathHistory.SelectedEntry()
	if !ok || entry.Commit.SHA == "" {
		m.Status = "select a path-history entry first"
		return nil
	}
	m.CompareLeft, m.CompareRight = entry.Commit.SHA, "HEAD"
	return m.startCompare()
}

func (m *Model) openBlame(path string, start int) tea.Cmd {
	if path == "" {
		m.Status = "select a file before opening blame"
		return nil
	}
	if start < 1 {
		start = 1
	}
	if m.BlameCancel != nil {
		m.BlameCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.BlameCancel = cancel
	m.BlameRequest++
	m.BlameGeneration = m.repositoryGeneration
	request, generation := m.BlameRequest, m.BlameGeneration
	m.BlameLoading, m.BlameErr = true, nil
	m.Blame.SetPage(path, start, nil, false)
	m.State, m.Status = StateOperationPending, "loading blame for "+platform.SafeText(path)
	m.Workspace.Navigate(workspace.Blame, "Blame: "+platform.SafeText(path))
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := blame.LoadPage(ctx, runner, blame.Request{Path: path, Start: start, Limit: blame.DefaultLimit})
		return BlameReadyMsg{Path: path, Start: page.Start, Lines: page.Lines, HasMore: page.HasMore, Truncated: page.Truncated, Request: request, Generation: generation, Err: err}
	}
}

func (m *Model) loadBlamePage() tea.Cmd {
	if m.Blame.Path == "" || m.BlameLoading || !m.Blame.HasMore {
		return nil
	}
	if m.BlameCancel != nil {
		m.BlameCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.BlameCancel = cancel
	m.BlameRequest++
	m.BlameGeneration = m.repositoryGeneration
	request, generation := m.BlameRequest, m.BlameGeneration
	start := m.Blame.Start + len(m.Blame.Lines)
	m.BlameLoading = true
	path := m.Blame.Path
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		page, err := blame.LoadPage(ctx, runner, blame.Request{Path: path, Start: start, Limit: blame.DefaultLimit})
		return BlameReadyMsg{Path: path, Start: page.Start, Lines: page.Lines, HasMore: page.HasMore, Truncated: page.Truncated, Request: request, Generation: generation, Err: err}
	}
}

func (m *Model) inspectSelectedBlame() tea.Cmd {
	line, ok := m.Blame.SelectedLine()
	if !ok || line.FinalSHA == "" {
		return nil
	}
	path := line.Filename
	if path == "" {
		path = m.Blame.Path
	}
	sha := line.FinalSHA
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.commandContext()
	m.State, m.Status = StateOperationPending, "loading blame origin commit"
	m.Workspace.Navigate(workspace.Log, "History")
	return func() tea.Msg {
		commit, err := history.LoadCommit(ctx, runner, sha)
		if err != nil {
			return HistoryInspectorReadyMsg{Err: err}
		}
		parent := ""
		if len(commit.Parents) > 0 {
			parent = commit.Parents[0]
		}
		inspector, inspectErr := history.InspectPath(ctx, runner, sha, parent, path)
		inspector.Commit = commit
		return HistoryInspectorReadyMsg{Inspector: inspector, Err: inspectErr}
	}
}

func (m Model) inspectSelectedCommit() tea.Cmd {
	if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
		return nil
	}
	commit := m.History.Rows[m.History.Selected].Commit
	return m.inspectCommit(commit, m.HistoryInspectorParent, m.HistoryInspectorPath)
}

func (m Model) inspectSelectedReflog() tea.Cmd {
	entry, ok := m.Reflog.SelectedEntry()
	if !ok {
		return nil
	}
	short := entry.SHA
	if len(short) > 12 {
		short = short[:12]
	}
	return m.inspectCommit(history.Commit{SHA: entry.SHA, Short: short, Author: entry.Actor, Subject: entry.Subject, Unix: entry.Timestamp}, "", "")
}

func (m *Model) compareSelectedReflog() tea.Cmd {
	entry, ok := m.Reflog.SelectedEntry()
	if !ok {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	m.ReflogCompare, m.ReflogCompareLoading = "", true
	return func() tea.Msg {
		inspector, err := history.InspectPath(ctx, runner, entry.SHA, "HEAD", "")
		return ReflogCompareReadyMsg{Text: inspector.Diff, Generation: generation, Err: err}
	}
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

func (m Model) loadTags() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		snapshot, err := tags.Load(m.commandContext(), runner, tags.LoadRequest{Repository: m.Discovery.Root})
		return TagsReadyMsg{Generation: generation, Snapshot: snapshot, Err: err}
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
	commits := append([]string(nil), m.HistoryRevertCommits...)
	mainline := m.HistoryRevertParent
	var revertResult git.Result
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	operationID := fmt.Sprintf("revert-%d-%s", generation, target)
	command := m.OperationEngine.Command(ctx, operationID, m.Discovery.Root, "revert "+target, 5*time.Minute, func(ctx context.Context) error {
		if len(commits) > 0 {
			result, err := history.RevertSelection(ctx, runner, confirmation, input, history.RevertPlan{Commits: commits, Mainline: mainline})
			revertResult = result
			return err
		}
		result, err := history.Revert(ctx, runner, confirmation, input)
		revertResult = result
		return err
	})
	return func() tea.Msg {
		started := time.Now()
		result := command()
		var snapshot *repo.Snapshot
		if current, snapshotErr := git.Snapshot(ctx, m.Discovery, generation); snapshotErr == nil {
			snapshot = &current
		}
		operation := &history.OperationRecord{
			Repository: m.Discovery.Root, Kind: "revert", Args: history.RedactArgs(revertResult.Args),
			Target: target, OldHead: m.Snapshot.Branch.OID, Refs: []string{m.Snapshot.Branch.Name}, Duration: time.Since(started),
		}
		if head, headErr := runner.Run(ctx, "rev-parse", "HEAD"); headErr == nil {
			operation.NewHead = strings.TrimSpace(string(head.Stdout))
		}
		attachLatestRecoveryPoint(ctx, runner, operation)
		paused := snapshot != nil && snapshot.Operation != nil && snapshot.Operation.Kind() == sequencer.KindRevert
		return RevertFinishedMsg{Repository: generation, Result: revertResult, Snapshot: snapshot, Operation: operation, Paused: paused, Err: result.Result.Err}
	}
}

func (m *Model) prepareCherryPick() bool {
	if m.History.Basket.Count() > 0 {
		m.CherryPickCommits = m.History.Basket.SHAs()
	} else if m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
		m.CherryPickCommits = []string{m.History.Rows[m.History.Selected].Commit.SHA}
	} else {
		m.Status = "select commits before cherry-picking"
		return false
	}
	for _, sha := range m.CherryPickCommits {
		for _, row := range m.History.Rows {
			if row.Commit.SHA == sha && len(row.Commit.Parents) > 1 {
				m.Status = "cherry-pick merge commits require explicit mainline selection"
				m.CherryPickCommits = nil
				return false
			}
		}
	}
	m.CherryPickConfirm = true
	suffix := "s"
	if len(m.CherryPickCommits) == 1 {
		suffix = ""
	}
	m.Status = fmt.Sprintf("confirm cherry-pick %d commit%s? (y/n)", len(m.CherryPickCommits), suffix)
	return true
}

func (m Model) cherryPickSelectedHistory() tea.Cmd {
	if len(m.CherryPickCommits) == 0 {
		return nil
	}
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	commits := append([]string(nil), m.CherryPickCommits...)
	args := append([]string{"cherry-pick"}, commits...)
	operation := &history.OperationRecord{
		Repository: m.Discovery.Root,
		Kind:       "cherry-pick",
		Args:       history.RedactArgs(args),
		Target:     m.Snapshot.Branch.Name,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{m.Snapshot.Branch.Name},
	}
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	id := fmt.Sprintf("cherry-pick-%d-%s", generation, strings.Join(commits, ","))
	var outcome cherrypick.Outcome
	command := m.OperationEngine.Command(ctx, id, m.Discovery.Root, "cherry-pick", 5*time.Minute, func(ctx context.Context) error {
		outcome = (cherrypick.Engine{Runner: runner, Discovery: m.Discovery, Repository: m.Discovery.Root, Generation: generation}).Execute(ctx, cherrypick.Request{
			Repository: m.Discovery.Root, Generation: generation, SHAs: commits,
		})
		return outcome.Err
	})
	return func() tea.Msg {
		started := time.Now()
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		completed := *operation
		completed.Duration = time.Since(started)
		if head, err := runner.Run(ctx, "rev-parse", "HEAD"); err == nil {
			completed.NewHead = strings.TrimSpace(string(head.Stdout))
		}
		if vSnapshot := outcome.Snapshot; vSnapshot != nil {
			completed.PostSnapshotFingerprint = undopolicy.SnapshotFingerprint(*vSnapshot)
		}
		attachLatestRecoveryPoint(ctx, runner, &completed)
		return CherryPickFinishedMsg{Repository: generation, Outcome: outcome, Operation: &completed}
	}
}

func (m Model) selectedUndoRecord() *history.OperationRecord {
	if m.ActivityLog == nil {
		return nil
	}
	events := m.filteredJournalEvents()
	index := len(events) - 1 - m.JournalOffset
	if index < 0 || index >= len(events) || events[index].Operation == nil {
		return nil
	}
	record := *events[index].Operation
	if record.Repository != m.Discovery.Root || record.Kind != "commit" || record.Outcome != "success" || record.Target == "" || record.OldHead == "" || record.NewHead == "" || record.PostSnapshotFingerprint == "" {
		return nil
	}
	return &record
}

func (m Model) selectedRedoRecord() *history.OperationRecord {
	if m.ActivityLog == nil {
		return nil
	}
	events := m.filteredJournalEvents()
	index := len(events) - 1 - m.JournalOffset
	if index < 0 || index >= len(events) || events[index].Operation == nil {
		return nil
	}
	record := *events[index].Operation
	if record.Repository != m.Discovery.Root || record.Kind != "undo commit" || record.Outcome != "success" || record.Target == "" || record.OldHead == "" || record.NewHead == "" || record.PostSnapshotFingerprint == "" {
		return nil
	}
	return &record
}

func (m Model) selectedJournalEvent() *history.Event {
	if m.ActivityLog == nil {
		return nil
	}
	events := m.filteredJournalEvents()
	index := len(events) - 1 - m.JournalOffset
	if index < 0 || index >= len(events) {
		return nil
	}
	event := events[index]
	return &event
}

func (m Model) activeJournalOperations() []operations.Result {
	if m.OperationEngine == nil {
		return nil
	}
	all := m.OperationEngine.Snapshot()
	active := make([]operations.Result, 0, len(all))
	for _, result := range all {
		if result.Repo == m.Discovery.Root && (result.State == operations.Pending || result.State == operations.Running) {
			active = append(active, result)
		}
	}
	return active
}

func (m Model) retryableJournalOperation() *operations.Result {
	if m.OperationEngine == nil {
		return nil
	}
	all := m.OperationEngine.Snapshot()
	for i := len(all) - 1; i >= 0; i-- {
		result := all[i]
		if result.Repo == m.Discovery.Root && result.Retryable && (result.State == operations.Failed || result.State == operations.Cancelled || result.State == operations.TimedOut) {
			return &result
		}
	}
	return nil
}

func (m Model) undoJournalOperation() tea.Cmd {
	if m.UndoRecord == nil {
		return nil
	}
	record := *m.UndoRecord
	ctx, generation := m.commandContext(), m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	id := fmt.Sprintf("undo-commit-%d-%s", generation, record.NewHead)
	var outcome undopolicy.Outcome
	command := m.OperationEngine.Command(ctx, id, m.Discovery.Root, "undo commit", 5*time.Minute, func(ctx context.Context) error {
		outcome = undopolicy.Execute(ctx, runner, undopolicy.Request{
			Repository: m.Discovery.Root, Kind: record.Kind, Ref: record.Target,
			OldHead: record.OldHead, NewHead: record.NewHead,
			PostSnapshotHash: record.PostSnapshotFingerprint, Discovery: m.Discovery, Generation: generation,
		})
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		completed := &history.OperationRecord{
			Repository: m.Discovery.Root, Kind: "undo commit", Args: []string{"reset", "--soft", record.OldHead},
			Target: record.Target, OldHead: record.NewHead, NewHead: record.OldHead,
			Refs: append([]string(nil), record.Refs...), Duration: result.Result.Finished.Sub(result.Result.Started),
		}
		if outcome.Snapshot.Root != "" {
			completed.PostSnapshotFingerprint = undopolicy.SnapshotFingerprint(outcome.Snapshot)
		}
		attachLatestRecoveryPoint(ctx, runner, completed)
		return UndoFinishedMsg{Repository: generation, Outcome: outcome, Operation: completed}
	}
}

func (m Model) redoJournalOperation() tea.Cmd {
	if m.RedoRecord == nil {
		return nil
	}
	record := *m.RedoRecord
	ctx, generation := m.commandContext(), m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	id := fmt.Sprintf("redo-commit-%d-%s", generation, record.OldHead)
	var outcome redopolicy.Outcome
	command := m.OperationEngine.Command(ctx, id, m.Discovery.Root, "redo commit", 5*time.Minute, func(ctx context.Context) error {
		outcome = redopolicy.Execute(ctx, runner, redopolicy.Request{
			Repository: m.Discovery.Root, Kind: record.Kind, Ref: record.Target,
			OldHead: record.OldHead, NewHead: record.NewHead,
			PostSnapshotHash: record.PostSnapshotFingerprint, Discovery: m.Discovery, Generation: generation,
		})
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		completed := &history.OperationRecord{
			Repository: m.Discovery.Root, Kind: "redo commit", Args: []string{"reset", "--soft", record.OldHead},
			Target: record.Target, OldHead: record.NewHead, NewHead: record.OldHead,
			Refs: append([]string(nil), record.Refs...), Duration: result.Result.Finished.Sub(result.Result.Started),
		}
		if outcome.Snapshot.Root != "" {
			completed.PostSnapshotFingerprint = undopolicy.SnapshotFingerprint(outcome.Snapshot)
		}
		attachLatestRecoveryPoint(ctx, runner, completed)
		return RedoFinishedMsg{Repository: generation, Outcome: outcome, Operation: completed}
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

func (m Model) loadRemoteTracking(remote string) tea.Cmd {
	generation := m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		branches, err := remotes.TrackingBranches(m.commandContext(), runner, remote)
		return RemoteTrackingReadyMsg{Remote: remote, Branches: branches, Repository: generation, Err: err}
	}
}

func (m Model) previewRemotePrune(remote string) tea.Cmd {
	generation := m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		result, err := remotes.Prune(m.commandContext(), runner, remote, true)
		return RemotePrunePreviewMsg{Remote: remote, Text: platform.SafeText(string(result.Stdout)), Repository: generation, Err: err}
	}
}

func (m Model) loadGitHub() tea.Cmd {
	generation := m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	branch := m.Snapshot.Branch.Name
	tokenEnv := m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	return func() tea.Msg {
		entries, err := remotes.List(m.commandContext(), runner)
		if err != nil {
			return GitHubReadyMsg{Generation: generation, Branch: branch, Err: err}
		}
		var repository provider.Repository
		for _, remote := range entries {
			if candidate, ok := provider.ParseGitHubRemote(remote.FetchURL); ok {
				repository = candidate
				break
			}
		}
		if repository.Owner == "" {
			return GitHubReadyMsg{Generation: generation, Branch: branch, Err: fmt.Errorf("no GitHub remote detected")}
		}
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		cache := m.GitHubCache
		if cache == nil {
			cache = provider.NewPullRequestCache(2 * time.Minute)
		}
		pull, err := cache.Get(m.commandContext(), client, repository, branch)
		if err != nil {
			return GitHubReadyMsg{Generation: generation, Repository: repository, Branch: branch, Err: err}
		}
		checksCache := m.GitHubChecksCache
		if checksCache == nil {
			checksCache = provider.NewCache[provider.ChecksSnapshot](2 * time.Minute)
		}
		checks, err := checksCache.Get(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"@"+branch, func(ctx context.Context) (provider.ChecksSnapshot, error) {
			return client.Checks(ctx, repository, branch)
		})
		if err != nil {
			return GitHubReadyMsg{Generation: generation, Repository: repository, Branch: branch, Pull: pull, Err: err}
		}
		pullsCache := m.GitHubPullsCache
		if pullsCache == nil {
			pullsCache = provider.NewCache[[]provider.PullRequest](2 * time.Minute)
		}
		providerStale := false
		pulls, pullsStale, _ := pullsCache.GetWithStale(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"/open", func(ctx context.Context) ([]provider.PullRequest, error) {
			return client.ListPullRequests(ctx, repository, 1, 25)
		})
		providerStale = providerStale || pullsStale
		var detail *provider.PullRequestDetail
		detailsCache := m.GitHubDetailsCache
		if detailsCache == nil {
			detailsCache = provider.NewCache[provider.PullRequestDetail](2 * time.Minute)
		}
		if loaded, stale, detailErr := detailsCache.GetWithStale(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"/pull/"+fmt.Sprint(pull.Number), func(ctx context.Context) (provider.PullRequestDetail, error) {
			return client.PullRequestDetail(ctx, repository, pull.Number)
		}); detailErr == nil || loaded.Number == pull.Number {
			detail = &loaded
			providerStale = providerStale || stale
		}
		commentsCache := m.GitHubCommentsCache
		if commentsCache == nil {
			commentsCache = provider.NewCache[[]provider.ReviewComment](2 * time.Minute)
		}
		comments, commentsStale, _ := commentsCache.GetWithStale(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"/pull/"+fmt.Sprint(pull.Number)+"/comments", func(ctx context.Context) ([]provider.ReviewComment, error) {
			return client.ListReviewComments(ctx, repository, pull.Number)
		})
		providerStale = providerStale || commentsStale
		issuesCache := m.GitHubIssuesCache
		if issuesCache == nil {
			issuesCache = provider.NewCache[[]provider.Issue](2 * time.Minute)
		}
		issues, issuesStale, _ := issuesCache.GetWithStale(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"/issues/open", func(ctx context.Context) ([]provider.Issue, error) {
			return client.ListIssues(ctx, repository, "open", 1, provider.MaxIssues)
		})
		providerStale = providerStale || issuesStale
		releasesCache := m.GitHubReleasesCache
		if releasesCache == nil {
			releasesCache = provider.NewCache[[]provider.Release](2 * time.Minute)
		}
		releases, releasesStale, _ := releasesCache.GetWithStale(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"/releases", func(ctx context.Context) ([]provider.Release, error) {
			return client.ListReleases(ctx, repository, 1, provider.MaxReleases)
		})
		providerStale = providerStale || releasesStale
		reviewsCache := m.GitHubReviewsCache
		if reviewsCache == nil {
			reviewsCache = provider.NewCache[provider.ReviewSnapshot](2 * time.Minute)
		}
		review, err := reviewsCache.Get(m.commandContext(), repository.Host+"/"+repository.Owner+"/"+repository.Name+"#"+fmt.Sprint(pull.Number), func(ctx context.Context) (provider.ReviewSnapshot, error) {
			return client.Reviews(ctx, repository, pull.Number)
		})
		return GitHubReadyMsg{Generation: generation, Repository: repository, Branch: branch, Pull: pull, Pulls: pulls, Issues: issues, Releases: releases, Detail: detail, Comments: comments, Checks: checks, Review: review, ProviderStale: providerStale, Err: err}
	}
}

func (m *Model) startGitHubCreate() tea.Cmd {
	if m.GitHub.Repository.Owner == "" || m.GitHub.Repository.Name == "" {
		m.Status = "GitHub repository is not available"
		return nil
	}
	if strings.TrimSpace(m.Snapshot.Branch.Name) == "" {
		m.Status = "PR creation requires a checked-out branch"
		return nil
	}
	base := m.GitHub.Pull.Base
	if strings.TrimSpace(base) == "" {
		base = m.Snapshot.Branch.Upstream
	}
	if strings.Contains(base, "/") {
		base = base[strings.LastIndexByte(base, '/')+1:]
	}
	if base == "" {
		base = "main"
	}
	m.GitHubCreateMode, m.GitHubCreateField = true, 0
	m.GitHubCreateTitle, m.GitHubCreateBody, m.GitHubCreateBase = "", "", base
	m.Status = "PR title: "
	return nil
}

func (m *Model) startGitHubIssue() tea.Cmd {
	if m.GitHub.Repository.Owner == "" || m.GitHub.Repository.Name == "" {
		m.Status = "GitHub repository is not available"
		return nil
	}
	m.GitHubIssueMode, m.GitHubIssueField = true, 0
	m.GitHubIssueTitle, m.GitHubIssueBody, m.GitHubIssueLabels = "", "", ""
	m.Status = "issue title: "
	return nil
}

func (m *Model) updateGitHubIssueKey(key string) tea.Cmd {
	if key == "esc" {
		m.GitHubIssueMode = false
		m.Status = "GitHub issue creation cancelled"
		return nil
	}
	if key == "tab" || key == "enter" {
		if m.GitHubIssueField < 2 {
			m.GitHubIssueField++
			m.Status = m.githubIssuePrompt()
			return nil
		}
		request := m.githubIssueRequest()
		if err := request.Validate(); err != nil {
			m.Status = "GitHub issue: " + err.Error()
			return nil
		}
		m.GitHubIssueMode, m.GitHubIssueConfirm = false, true
		m.Status = "create GitHub issue " + platform.SafeText(request.Title) + "? (y/n)"
		return nil
	}
	var value *string
	switch m.GitHubIssueField {
	case 0:
		value = &m.GitHubIssueTitle
	case 1:
		value = &m.GitHubIssueBody
	default:
		value = &m.GitHubIssueLabels
	}
	switch key {
	case "backspace":
		*value = removeLastRune(*value)
	case "space":
		*value += " "
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			*value += key
		}
	}
	m.Status = m.githubIssuePrompt()
	return nil
}

func (m Model) githubIssuePrompt() string {
	switch m.GitHubIssueField {
	case 0:
		return "issue title: " + platform.SafeText(m.GitHubIssueTitle)
	case 1:
		return "issue body: " + platform.SafeText(m.GitHubIssueBody)
	default:
		return "issue labels (comma-separated): " + platform.SafeText(m.GitHubIssueLabels)
	}
}

func (m Model) githubIssueRequest() provider.IssueCreateRequest {
	labels := make([]string, 0)
	for _, label := range strings.Split(m.GitHubIssueLabels, ",") {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			labels = append(labels, trimmed)
		}
	}
	return provider.IssueCreateRequest{Title: strings.TrimSpace(m.GitHubIssueTitle), Body: m.GitHubIssueBody, Labels: labels}
}

func (m Model) createGitHubIssue() tea.Cmd {
	repository, request, tokenEnv := m.GitHub.Repository, m.githubIssueRequest(), m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		issue, err := client.CreateIssue(ctx, repository, request)
		return GitHubIssueCreatedMsg{Issue: issue, Err: err}
	}
}

func (m *Model) updateGitHubCreateKey(key string) tea.Cmd {
	if key == "esc" {
		m.GitHubCreateMode = false
		m.Status = "GitHub PR creation cancelled"
		return nil
	}
	if key == "tab" || key == "enter" {
		if m.GitHubCreateField < 2 {
			m.GitHubCreateField++
			m.Status = m.githubCreatePrompt()
			return nil
		}
		request := provider.PullRequestCreateRequest{Title: strings.TrimSpace(m.GitHubCreateTitle), Body: m.GitHubCreateBody, Head: m.Snapshot.Branch.Name, Base: strings.TrimSpace(m.GitHubCreateBase)}
		if strings.TrimSpace(request.Title) == "" || strings.TrimSpace(request.Head) == "" || strings.TrimSpace(request.Base) == "" {
			m.Status = "PR title, current branch, and base are required"
			return nil
		}
		m.GitHubCreateMode, m.GitHubCreateConfirm = false, true
		m.Status = "create GitHub PR " + platform.SafeText(request.Title) + " from " + platform.SafeText(request.Head) + " to " + platform.SafeText(request.Base) + "? (y/n)"
		return nil
	}
	var value *string
	switch m.GitHubCreateField {
	case 0:
		value = &m.GitHubCreateTitle
	case 1:
		value = &m.GitHubCreateBody
	default:
		value = &m.GitHubCreateBase
	}
	switch key {
	case "backspace":
		*value = removeLastRune(*value)
	case "space":
		*value += " "
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			*value += key
		}
	}
	m.Status = m.githubCreatePrompt()
	return nil
}

func (m Model) githubCreatePrompt() string {
	switch m.GitHubCreateField {
	case 0:
		return "PR title: " + platform.SafeText(m.GitHubCreateTitle)
	case 1:
		return "PR body: " + platform.SafeText(m.GitHubCreateBody)
	default:
		return "PR base: " + platform.SafeText(m.GitHubCreateBase)
	}
}

func (m Model) createGitHubPullRequest() tea.Cmd {
	request := provider.PullRequestCreateRequest{Title: strings.TrimSpace(m.GitHubCreateTitle), Body: m.GitHubCreateBody, Head: m.Snapshot.Branch.Name, Base: strings.TrimSpace(m.GitHubCreateBase)}
	repository, tokenEnv := m.GitHub.Repository, m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		pull, err := client.CreatePullRequest(ctx, repository, request)
		return GitHubPullRequestCreatedMsg{Pull: pull, Err: err}
	}
}

func (m *Model) startGitHubMerge() tea.Cmd {
	if m.GitHub.Pull.Number < 1 {
		m.Status = "no pull request is loaded"
		return nil
	}
	m.GitHubMergeMode = true
	m.GitHubMergeMethod = provider.MergeMethodMerge
	m.Status = "merge method: [m] merge  [s] squash  [r] rebase  [enter] refresh  [esc] cancel"
	return nil
}

func (m *Model) updateGitHubMergeKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.GitHubMergeMode = false
		m.Status = "GitHub merge cancelled"
	case "m":
		m.GitHubMergeMethod = provider.MergeMethodMerge
	case "s":
		m.GitHubMergeMethod = provider.MergeMethodSquash
	case "r":
		m.GitHubMergeMethod = provider.MergeMethodRebase
	case "enter":
		m.GitHubMergeMode, m.GitHubMergeRefresh = false, true
		m.State, m.Status = StateOperationPending, "refreshing GitHub mergeability, checks, and review state"
		return m.loadGitHub()
	}
	if m.GitHubMergeMode {
		m.Status = "merge method: " + string(m.GitHubMergeMethod) + "  [enter] refresh  [esc] cancel"
	}
	return nil
}

func (m *Model) startGitHubReview(event provider.ReviewEvent) tea.Cmd {
	if m.GitHub.Pull.Number < 1 {
		m.Status = "no pull request is loaded"
		return nil
	}
	m.GitHubReviewEvent, m.GitHubReviewBody = event, ""
	if event != provider.ReviewEventComment {
		m.GitHubReplyCommentID = 0
	}
	if event == provider.ReviewEventApprove {
		m.GitHubReviewConfirm = true
		m.Status = "approve GitHub PR #" + fmt.Sprint(m.GitHub.Pull.Number) + "? (y/n)"
		return nil
	}
	m.GitHubReviewMode = true
	m.Status = m.githubReviewPrompt()
	return nil
}

func (m *Model) updateGitHubReviewKey(key string) tea.Cmd {
	if key == "esc" {
		m.GitHubReviewMode, m.GitHubReviewBody = false, ""
		m.Status = "GitHub review cancelled"
		return nil
	}
	if key == "enter" {
		if m.GitHubReviewEvent == provider.ReviewEventRequestChanges && strings.TrimSpace(m.GitHubReviewBody) == "" {
			m.Status = "request-changes review requires a reason"
			return nil
		}
		m.GitHubReviewMode, m.GitHubReviewConfirm = false, true
		m.Status = "submit GitHub " + strings.ToLower(string(m.GitHubReviewEvent)) + " review? (y/n)"
		return nil
	}
	switch key {
	case "backspace":
		m.GitHubReviewBody = removeLastRune(m.GitHubReviewBody)
	case "space":
		m.GitHubReviewBody += " "
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			m.GitHubReviewBody += key
		}
	}
	m.Status = m.githubReviewPrompt()
	return nil
}

func (m Model) githubReviewPrompt() string {
	return strings.ToLower(string(m.GitHubReviewEvent)) + " review: " + platform.SafeText(m.GitHubReviewBody)
}

func (m Model) submitGitHubReview() tea.Cmd {
	repository, number, submission, tokenEnv := m.GitHub.Repository, m.GitHub.Pull.Number, provider.ReviewSubmission{Event: m.GitHubReviewEvent, Body: m.GitHubReviewBody, CommitID: m.GitHub.Pull.HeadSHA}, m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		if m.GitHubReplyCommentID > 0 && m.GitHubReviewEvent == provider.ReviewEventComment {
			comment, err := client.CreateReviewComment(ctx, repository, number, provider.ReviewCommentRequest{Body: m.GitHubReviewBody, CommitID: m.GitHub.Pull.HeadSHA, InReplyTo: m.GitHubReplyCommentID})
			return GitHubReviewCommentFinishedMsg{Comment: comment, Err: err}
		}
		result, err := client.SubmitReview(ctx, repository, number, submission)
		return GitHubReviewFinishedMsg{Result: result, Err: err}
	}
}

func (m *Model) startGitHubCheckAction(action string) tea.Cmd {
	if len(m.GitHub.Checks.Runs) == 0 || m.GitHub.SelectedRun < 0 || m.GitHub.SelectedRun >= len(m.GitHub.Checks.Runs) {
		m.Status = "no GitHub check run is selected"
		return nil
	}
	run := m.GitHub.Checks.Runs[m.GitHub.SelectedRun]
	if run.ID < 1 {
		m.Status = "selected check run has no provider action ID"
		return nil
	}
	if action == "rerun" {
		if run.Status != "completed" {
			m.Status = "selected check run is still running"
			return nil
		}
		if run.Conclusion == "success" || run.Conclusion == "neutral" || run.Conclusion == "skipped" {
			m.Status = "selected check run did not fail"
			return nil
		}
	}
	if action == "cancel" && run.Status == "completed" {
		m.Status = "selected check run is already completed"
		return nil
	}
	m.GitHubCheckAction, m.GitHubCheckActionRunID, m.GitHubCheckActionConfirm = action, run.ID, true
	m.Status = "confirm GitHub " + action + " for check " + platform.SafeText(run.Name) + "? (y/n)"
	return nil
}

func (m Model) runGitHubCheckAction() tea.Cmd {
	repository, runID, action, tokenEnv := m.GitHub.Repository, m.GitHubCheckActionRunID, m.GitHubCheckAction, m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		var err error
		if action == "rerun" {
			err = client.RerunFailedJobs(ctx, repository, runID)
		} else {
			err = client.CancelRun(ctx, repository, runID)
		}
		return GitHubCheckActionFinishedMsg{Action: action, Err: err}
	}
}

func (m Model) mergeGitHubPullRequest() tea.Cmd {
	repository, number, method, expectedSHA, tokenEnv := m.GitHub.Repository, m.GitHub.Pull.Number, m.GitHubMergeMethod, m.GitHub.Pull.HeadSHA, m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		result, err := client.MergePullRequest(ctx, repository, number, provider.MergeRequest{Method: method, ExpectedSHA: expectedSHA})
		return GitHubMergeFinishedMsg{Result: result, Err: err}
	}
}

func (m Model) deleteGitHubBranch() tea.Cmd {
	repository, branch, tokenEnv := m.GitHub.Repository, m.GitHubBranchDeleteTarget, m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}
	ctx := m.commandContext()
	return func() tea.Msg {
		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		err := client.DeleteBranch(ctx, repository, branch)
		return GitHubBranchDeleteFinishedMsg{Branch: branch, Err: err}
	}
}

func (m Model) loadPlugins() tea.Cmd {
	directories := append([]string(nil), m.PluginDirectories...)
	statePath := m.PluginStatePath
	outputLimit := m.PluginOutputLimit
	commandContext := m.commandContext()
	return func() tea.Msg {
		entries, err := plugins.Discover(commandContext, directories, 128)
		if err == nil && statePath != "" {
			state, stateErr := plugins.LoadState(statePath)
			if stateErr != nil {
				return PluginsReadyMsg{Err: stateErr}
			}
			entries = plugins.ApplyState(entries, state)
		}
		if err == nil {
			host := plugins.Runtime{OutputLimit: outputLimit}
			for index := range entries {
				entries[index] = plugins.Probe(commandContext, host, entries[index], plugins.DefaultCapabilities)
			}
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
	registryEntries := append([]registry.Repository(nil), m.RepositoryRegistry...)
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
		for index := range repositories {
			for _, entry := range registryEntries {
				if entry.Path == repositories[index].Path {
					repositories[index].LastAutoFetch = entry.LastAutoFetch
					repositories[index].LastAutoFetchStatus = entry.LastAutoFetchStatus
					repositories[index].LastAutoFetchError = entry.LastAutoFetchError
					repositories[index].LastAutoFetchMillis = entry.LastAutoFetchMillis
					break
				}
			}
		}
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

func (m Model) applyAutoFetchResults(rows []registry.Row) []registry.Row {
	if len(m.AutoFetchResults) == 0 {
		return rows
	}
	updated := append([]registry.Row(nil), rows...)
	for index := range updated {
		result, ok := m.AutoFetchResults[updated[index].Repository.Path]
		if !ok {
			continue
		}
		updated[index].RemoteFetchStatus = result.Status
		updated[index].RemoteFetchAt = result.Finished
		updated[index].RemoteFetchError = result.FailureClass
		if result.Status == "failed" {
			warning := "auto-fetch: " + result.FailureClass
			updated[index].Warnings = append(updated[index].Warnings, warning)
			if updated[index].Attention == "" {
				updated[index].Attention = warning
			}
			if updated[index].Health.Severity != health.SeverityCritical {
				updated[index].Health.Severity = health.SeverityWarning
			}
			updated[index].Health.Attention = append(updated[index].Health.Attention, warning)
		}
	}
	return updated
}

func (m Model) applyProviderCIAttention(rows []registry.Row) []registry.Row {
	if len(m.ProviderCI) == 0 {
		return rows
	}
	updated := append([]registry.Row(nil), rows...)
	for index := range updated {
		status, ok := m.ProviderCI[updated[index].Repository.Path]
		if !ok {
			continue
		}
		updated[index].ProviderCIState = status.State
		updated[index].ProviderCIStale = status.Stale
		updated[index].ProviderCIAttention = status.Attention
		if status.Attention != "" && updated[index].Attention == "" {
			updated[index].Attention = "ci:" + status.Attention
		}
	}
	return updated
}

func (m Model) applyCommitActivity(rows []registry.Row) []registry.Row {
	if m.Discovery.Root == "" || len(m.HistoryCommits) == 0 {
		return rows
	}
	commits := make([]int64, 0, len(m.HistoryCommits))
	for _, commit := range m.HistoryCommits {
		commits = append(commits, commit.Unix)
	}
	activity := activityviz.CommitBuckets(commits, time.Now(), 24*time.Hour, 8)
	updated := append([]registry.Row(nil), rows...)
	for index := range updated {
		if updated[index].Repository.Path == m.Discovery.Root {
			updated[index].Activity = append([]int(nil), activity...)
		}
	}
	return updated
}

func (m *Model) startRepositoryBatchFetch() tea.Cmd {
	if len(m.Repositories.Rows) == 0 {
		m.Status = "no repositories are available for batch fetch"
		return nil
	}
	m.RepositoryBatchRetry, m.RepositoryBatchConfirm = false, true
	m.RepositoryBatchAction, m.RepositoryBatchStrategy = multirepo.ActionFetch, ""
	m.RepositoryBatchCancel = nil
	m.RepositoryBatchResults = nil
	m.Status = fmt.Sprintf("fetch %d discovered repositories? (y/n)", len(m.Repositories.Rows))
	return nil
}

func (m *Model) startRepositoryBatchPull() tea.Cmd {
	if len(m.Repositories.Rows) == 0 {
		m.Status = "no repositories are available for batch pull"
		return nil
	}
	// Batch pull is deliberately limited to fast-forward-only until a caller
	// explicitly supplies per-repository merge/rebase policy.
	m.RepositoryBatchRetry, m.RepositoryBatchConfirm = false, true
	m.RepositoryBatchAction, m.RepositoryBatchStrategy = multirepo.ActionPull, "ff-only"
	m.RepositoryBatchCancel = nil
	m.RepositoryBatchResults = nil
	m.Status = fmt.Sprintf("pull %d discovered repositories with ff-only? (y/n)", len(m.Repositories.Rows))
	return nil
}

func (m *Model) startRepositoryBatchRetry() tea.Cmd {
	failed := 0
	for _, result := range m.RepositoryBatchResults {
		if result.Status == "failed" {
			failed++
		}
	}
	if failed == 0 {
		m.Status = "no failed batch repositories to retry"
		return nil
	}
	if len(m.RepositoryBatchResults) > 0 {
		for _, result := range m.RepositoryBatchResults {
			if result.Status == "failed" {
				m.RepositoryBatchAction = result.Request.Action
				m.RepositoryBatchStrategy = result.Request.Strategy
				break
			}
		}
	}
	m.RepositoryBatchRetry, m.RepositoryBatchConfirm = true, true
	action := "fetch"
	if m.RepositoryBatchAction == multirepo.ActionPull {
		action = "pull " + m.RepositoryBatchStrategy
	}
	m.Status = fmt.Sprintf("retry %s for %d failed repositories? (y/n)", action, failed)
	return nil
}

func (m *Model) runRepositoryBatchFetch() tea.Cmd {
	rows := append([]registry.Row(nil), m.Repositories.Rows...)
	if m.RepositoryBatchRetry {
		failed := make(map[string]struct{})
		for _, result := range m.RepositoryBatchResults {
			if result.Status == "failed" {
				failed[result.Request.Repository.Root] = struct{}{}
			}
		}
		filtered := rows[:0]
		for _, row := range rows {
			if _, ok := failed[row.Repository.Path]; ok {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.RepositoryBatchCancel = cancel
	workers := 2
	if m.RepositoryEngine != nil && m.RepositoryEngine.Workers > 0 {
		workers = m.RepositoryEngine.Workers
	}
	return func() tea.Msg {
		events := make(chan tea.Msg, len(rows)*3+1)
		go func() {
			defer close(events)
			requests := make([]multirepo.Request, len(rows))
			for index, row := range rows {
				requests[index] = multirepo.Request{Repository: multirepo.Repository{ID: domain.RepositoryID(row.Repository.Path), Root: row.Repository.Path}, Remote: "origin", Branch: row.Branch, Strategy: m.RepositoryBatchStrategy, Action: m.RepositoryBatchAction}
				if requests[index].Action == "" {
					requests[index].Action = multirepo.ActionFetch
				}
				events <- RepositoryBatchProgressMsg{Path: row.Repository.Path, Status: "queued", Total: len(rows), Events: events}
			}
			var progressMu sync.Mutex
			completed := 0
			results := multirepo.Run(ctx, requests, workers, func(ctx context.Context, request multirepo.Request) error {
				progressMu.Lock()
				started := completed
				progressMu.Unlock()
				events <- RepositoryBatchProgressMsg{Path: request.Repository.Root, Status: "running", Completed: started, Total: len(requests), Events: events}
				discovery, err := git.Discover(ctx, request.Repository.Root)
				if err == nil {
					entries, listErr := remotes.List(ctx, git.NewRunner(discovery.Root))
					err = listErr
					remote := request.Remote
					if len(entries) > 0 {
						remote = entries[0].Name
					}
					if err == nil && remote == "" {
						err = errors.New("repository has no configured remote")
					}
					if err == nil {
						if request.Action == multirepo.ActionPull {
							if request.Branch == "" {
								err = errors.New("repository has no checked-out branch")
							} else {
								_, err = remotes.Pull(ctx, git.NewRunner(discovery.Root), remote, request.Branch, request.Strategy)
							}
						} else {
							_, err = remotes.Fetch(ctx, git.NewRunner(discovery.Root), remote)
						}
					}
				}
				progressMu.Lock()
				completed++
				finished := completed
				progressMu.Unlock()
				status := "succeeded"
				if err != nil {
					status = "failed"
				}
				events <- RepositoryBatchProgressMsg{Path: request.Repository.Root, Status: status, Completed: finished, Total: len(requests), Events: events}
				return err
			})
			events <- RepositoryBatchFinishedMsg{Results: results}
		}()
		return batchProgressCommand(events)()
	}
}

func batchProgressCommand(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		if progress, ok := msg.(RepositoryBatchProgressMsg); ok {
			progress.Events = events
			return progress
		}
		return msg
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
	commit := strings.TrimSpace(m.WorktreeAddCommit)
	runner := git.NewRunner(m.Discovery.Root)
	ctx, generation := m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := worktrees.AddWithCommit(ctx, runner, path, "", commit)
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
		m.WorktreeAddMode, m.WorktreeAddPath, m.WorktreeAddCommit, m.Status = false, "", "", "worktree creation cancelled"
	case "backspace":
		m.WorktreeAddPath = removeLastRune(m.WorktreeAddPath)
	case "enter":
		if strings.TrimSpace(m.WorktreeAddPath) == "" {
			m.Status = "worktree path is required"
		} else {
			m.WorktreeAddMode, m.State, m.Status = false, StateOperationPending, "adding worktree"
			cmd := m.addWorktree()
			m.WorktreeAddCommit = ""
			return cmd
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
	remoteArgs := append(strings.Fields(operation), remote)
	journal := &history.OperationRecord{
		Repository: repoRoot,
		Kind:       operation,
		Args:       history.RedactArgs(remoteArgs),
		Target:     remote,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{remote},
	}
	command := m.OperationEngine.CommandWithOptions(ctx, id, repoRoot, operation, 5*time.Minute, work, operations.Options{Retryable: operation == "fetch"})
	return func() tea.Msg {
		started := time.Now()
		result := command()
		completed := *journal
		completed.Duration = time.Since(started)
		if operation == "fetch" || operation == "push" {
			completed.NewHead = completed.OldHead
		}
		attachLatestRecoveryPoint(ctx, git.NewRunner(repoRoot), &completed)
		return RemoteOperationFinishedMsg{Operation: operation, Remote: remote, Repository: generation, Journal: &completed, Err: result.Result.Err}
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

func (m *Model) deleteSelectedRemoteTag() tea.Cmd {
	if m.Remotes.Selected < 0 || m.Remotes.Selected >= len(m.Remotes.Dashboard.Remotes) || strings.TrimSpace(m.RemoteTag) == "" {
		return nil
	}
	remote, tag := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name, strings.TrimSpace(m.RemoteTag)
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.startRemoteJob("delete remote tag", remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = "deleting tag"
	return m.remoteCommand(ctx, "delete remote tag "+tag, remote, func(ctx context.Context) error {
		_, err := remotes.DeleteTag(ctx, runner, remote, tag)
		return err
	})
}

func (m *Model) executeRemoteLifecycle() tea.Cmd {
	remote := m.RemoteMutationRemote
	mode := m.RemoteMutationMode
	urlValue := m.RemoteMutationURL
	newName := m.RemoteMutationNewName
	runner := git.NewRunner(m.Discovery.Root)
	var operation string
	var work operations.Work
	switch mode {
	case "add":
		operation = "add remote"
		work = func(ctx context.Context) error {
			_, err := remotes.Add(ctx, runner, remote, urlValue)
			return err
		}
	case "rename", "rename-confirm":
		operation = "rename remote"
		work = func(ctx context.Context) error {
			_, err := remotes.Rename(ctx, runner, remote, newName)
			return err
		}
	case "set-url", "set-url-confirm":
		operation = "set remote URL"
		work = func(ctx context.Context) error {
			_, err := remotes.SetURL(ctx, runner, remote, urlValue)
			return err
		}
	case "remove":
		operation = "remove remote"
		work = func(ctx context.Context) error {
			_, err := remotes.Remove(ctx, runner, remote)
			return err
		}
	case "prune":
		operation = "prune remote"
		work = func(ctx context.Context) error {
			_, err := remotes.Prune(ctx, runner, remote, false)
			return err
		}
	default:
		return nil
	}
	ctx := m.startRemoteJob(operation, remote)
	m.Remotes.Dashboard.Jobs[len(m.Remotes.Dashboard.Jobs)-1].Progress = operation
	return m.remoteCommand(ctx, operation, remote, work)
}

func (m *Model) resetRemoteMutation() {
	m.RemoteMutationMode, m.RemoteMutationRemote, m.RemoteMutationInput = "", "", ""
	m.RemoteMutationURL, m.RemoteMutationNewName = "", ""
	m.RemoteMutationImpact, m.RemoteMutationConfirm = nil, false
	m.RemotePrunePreview, m.RemotePruneConfirm = "", false
}

func (m *Model) updateRemoteMutationKey(key string) tea.Cmd {
	if !remoteMutationInputMode(m.RemoteMutationMode) {
		return nil
	}
	if key == "esc" {
		m.resetRemoteMutation()
		m.Status = "remote lifecycle action cancelled"
		return nil
	}
	switch key {
	case "backspace":
		m.RemoteMutationInput = removeLastRune(m.RemoteMutationInput)
	case "space":
		m.RemoteMutationInput += " "
	case "enter":
		value := strings.TrimSpace(m.RemoteMutationInput)
		if value == "" {
			m.Status = "remote input is required"
			return nil
		}
		switch m.RemoteMutationMode {
		case "add-name":
			m.RemoteMutationRemote, m.RemoteMutationInput, m.RemoteMutationMode = value, "", "add-url"
			m.Status = "URL for remote " + platform.SafeText(value) + ": "
			return nil
		case "add-url":
			m.RemoteMutationURL, m.RemoteMutationInput, m.RemoteMutationMode = value, "", "add"
			m.State, m.Status = StateOperationPending, "adding remote"
			return m.executeRemoteLifecycle()
		case "set-url":
			m.RemoteMutationURL, m.RemoteMutationInput, m.RemoteMutationMode = value, "", "set-url-confirm"
			m.State, m.Status = StateOperationPending, "setting remote URL"
			return m.executeRemoteLifecycle()
		case "rename":
			m.RemoteMutationNewName, m.RemoteMutationInput, m.RemoteMutationMode = value, "", "rename-confirm"
			m.RemoteMutationConfirm = true
			m.Status = remoteImpactStatus("rename "+m.RemoteMutationRemote+" to "+value, m.RemoteMutationImpact)
			return nil
		}
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			m.RemoteMutationInput += key
		}
	}
	if m.RemoteMutationMode != "" {
		if m.RemoteMutationMode == "add-url" || m.RemoteMutationMode == "set-url" {
			m.Status = "remote URL input: [hidden]"
		} else {
			m.Status = "remote " + m.RemoteMutationMode + ": " + platform.SafeText(m.RemoteMutationInput)
		}
	}
	return nil
}

func remoteMutationInputMode(mode string) bool {
	switch mode {
	case "add-name", "add-url", "set-url", "rename":
		return true
	default:
		return false
	}
}

func remoteImpactStatus(action string, impact []remotes.TrackingBranch) string {
	if len(impact) == 0 {
		return "confirm " + action + " (no local tracking branches found)? (y/n)"
	}
	parts := make([]string, 0, len(impact))
	for _, branch := range impact {
		parts = append(parts, branch.Local+" -> "+branch.Upstream)
	}
	return "confirm " + action + "; affects " + strings.Join(parts, ", ") + "? (y/n)"
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
	case workspace.Reflog:
		m.Reflog.Ref = "HEAD"
		return m.loadReflog(false)
	case workspace.Log:
		return m.loadHistory()
	case workspace.Tags:
		m.TagsLoading, m.TagsErr = true, nil
		return m.loadTags()
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

func (m *Model) openRebaseWorkspaceWithChoices(choices []rebaseview.Base) tea.Cmd {
	if len(choices) == 0 {
		m.Status = "interactive rebase requires an explicit base"
		return nil
	}
	view, err := rebaseview.New(m.Snapshot.Branch.Name, m.Snapshot.Branch.Upstream, choices, m.HistoryCommits)
	if err != nil {
		m.Status = "rebase plan: " + err.Error()
		return nil
	}
	view.SetDivergence(m.Snapshot.Branch.Ahead, m.Snapshot.Branch.Behind, m.Snapshot.Branch.Ahead)
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
	return m.openRebaseWorkspaceWithChoices(choices)
}

func (m *Model) openSelectedBranchRebase() tea.Cmd {
	if m.Branches.Selected < 0 || m.Branches.Selected >= len(m.Branches.Entries) {
		return nil
	}
	branch := m.Branches.Entries[m.Branches.Selected]
	if branch.Current {
		m.Status = "cannot rebase the checked-out branch onto itself"
		return nil
	}
	if m.Snapshot.Counts.Staged > 0 || m.Snapshot.Counts.Unstaged > 0 || m.Snapshot.Counts.Untracked > 0 {
		m.Status = "rebase requires a clean worktree; stash explicitly first"
		return nil
	}
	if len(m.HistoryCommits) == 0 {
		m.Status = "rebase history is still loading"
		return nil
	}
	return m.openRebaseWorkspaceWithChoices([]rebaseview.Base{{Label: branch.Name, Ref: branch.Name}})
}

func (m *Model) startRebase(autosquash bool) tea.Cmd {
	if !m.Rebase.StartEnabled() {
		m.Status = "rebase plan is loading, invalid, or has no explicit base"
		return nil
	}
	request := git.RebaseRequest{Base: m.Rebase.Base.Ref, Autosquash: autosquash, Plan: m.Rebase.Plan}
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	args := []string{"rebase", "--interactive", request.Base}
	if autosquash {
		args = append(args, "--autosquash")
	}
	operation := &history.OperationRecord{
		Repository: m.Discovery.Root,
		Kind:       "rebase",
		Args:       history.RedactArgs(args),
		Target:     request.Base,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{m.Snapshot.Branch.Name},
	}
	return func() tea.Msg {
		started := time.Now()
		outcome, err := runner.StartInteractiveRebase(m.commandContext(), request)
		completed := *operation
		completed.Duration = time.Since(started)
		attachLatestRecoveryPoint(m.commandContext(), runner, &completed)
		return RebaseFinishedMsg{Repository: generation, Outcome: outcome, Operation: &completed, Err: err}
	}
}

func (m *Model) createFixup() tea.Cmd {
	if m.Snapshot.Counts.Staged == 0 {
		m.Status = "stage changes before creating a fixup commit"
		return nil
	}
	if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
		m.Status = "select a commit for the fixup"
		return nil
	}
	target := m.History.Rows[m.History.Selected].Commit.SHA
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		result, err := runner.Commit(m.commandContext(), git.CommitOptions{FixupSHA: target})
		return FixupFinishedMsg{SHA: result.SHA, Target: target, Repository: generation, Err: err}
	}
}

func (m *Model) openHistoricalRebase(action rebase.Action) tea.Cmd {
	if m.History.Selected < 0 || m.History.Selected >= len(m.History.Rows) {
		m.Status = "select a historical commit first"
		return nil
	}
	selected := m.History.Rows[m.History.Selected].Commit
	if len(selected.Parents) == 0 {
		m.Status = "the root commit requires root-rebase support"
		return nil
	}
	choices := []rebaseview.Base{{Label: "selected commit parent", Ref: selected.Parents[0]}}
	view, err := rebaseview.New(m.Snapshot.Branch.Name, m.Snapshot.Branch.Upstream, choices, m.HistoryCommits)
	if err != nil {
		m.Status = "rebase plan: " + err.Error()
		return nil
	}
	entries := view.Plan.Entries()
	laterCommits := 0
	targetSeen := false
	for index, entry := range entries {
		if targetSeen && entry.Kind() == rebase.CommitEntry {
			laterCommits++
		}
		if entry.Kind() == rebase.CommitEntry && entry.SHA() == selected.SHA {
			view.Plan, err = view.Plan.ChangeAction(index, action)
			if err != nil {
				m.Status = "rebase plan: " + err.Error()
				return nil
			}
			targetSeen = true
		}
	}
	for _, ref := range selected.Refs {
		if strings.Contains(ref, "origin/") {
			view.Published, view.ReachableRemote = true, true
		}
	}
	m.Rebase, m.HistoricalRebaseAction, m.HistoricalRebaseTarget = view, action, selected.SHA
	m.Workspace.Navigate(workspace.Rebase, "Historical "+string(action))
	if len(m.HistoricalPatch) > 0 {
		m.Status = fmt.Sprintf("historical patch preview: %d later commit(s) will replay; Enter starts the guarded rewrite", laterCommits)
	}
	return nil
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
	args := []string{"commit"}
	if draft.Amend {
		args = append(args, "--amend")
	}
	if draft.NoEdit {
		args = append(args, "--no-edit")
	}
	if draft.Signoff {
		args = append(args, "--signoff")
	}
	if draft.Sign {
		args = append(args, "--gpg-sign")
	}
	operation := &history.OperationRecord{
		Repository: m.Discovery.Root,
		Kind:       "commit",
		Args:       history.RedactArgs(args),
		Target:     m.Snapshot.Branch.Name,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{m.Snapshot.Branch.Name},
	}
	return func() tea.Msg {
		started := time.Now()
		result, err := runner.Commit(ctx, git.CommitOptions{
			Message: []byte(draft.Message()), Amend: draft.Amend, NoEdit: draft.NoEdit,
			Signoff: draft.Signoff, Sign: draft.Sign, Author: draft.Author,
		})
		completed := *operation
		completed.Duration = time.Since(started)
		completed.NewHead = result.SHA
		if snapshot, snapshotErr := git.Snapshot(ctx, m.Discovery, generation); snapshotErr == nil {
			completed.PostSnapshotFingerprint = undopolicy.SnapshotFingerprint(snapshot)
		}
		attachLatestRecoveryPoint(ctx, runner, &completed)
		return CommitFinishedMsg{SHA: result.SHA, HookOutput: platform.SafeText(string(append(result.Result.Stdout, result.Result.Stderr...))), Repository: generation, Operation: &completed, Err: err}
	}
}

func (m *Model) continueHistoricalRebase() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		result, err := runner.ContinueRebase(m.commandContext())
		return RebaseContinueFinishedMsg{Repository: generation, Result: result, Err: err}
	}
}

func (m *Model) abortHistoricalRebase() tea.Cmd {
	runner := git.NewRunner(m.Discovery.Root)
	generation := m.repositoryGeneration
	return func() tea.Msg {
		result, err := runner.AbortRebase(m.commandContext())
		return RebaseAbortFinishedMsg{Repository: generation, Result: result, Err: err}
	}
}

func removeLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}

func (m *Model) updateJournalFilterKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.JournalFilterMode, m.JournalFilterInput = false, ""
		m.JournalOffset = 0
		m.Status = "journal filter cancelled"
	case "backspace":
		m.JournalFilterInput = removeLastRune(m.JournalFilterInput)
	case "space":
		m.JournalFilterInput += " "
	case "enter":
		m.JournalFilterMode = false
		m.JournalOffset = 0
		m.Status = "journal filter applied"
	default:
		if len([]rune(key)) == 1 {
			r := []rune(key)[0]
			if r != '\r' && r != '\n' && r != 0 {
				m.JournalFilterInput += key
			}
		}
	}
	if m.JournalFilterMode {
		m.Status = "journal filter: " + m.JournalFilterInput
	}
	return nil
}

func (m *Model) selectedSubmodulePath() string {
	path := string(m.Files.SelectedPath())
	for _, module := range m.Submodules.Modules {
		if module.Path == path {
			return module.Path
		}
	}
	return ""
}

func (m *Model) openSelectedSubmodule() tea.Cmd {
	if len(m.repositoryParents) >= maxRepositoryParentDepth {
		m.Status = "submodule navigation depth limit reached"
		return nil
	}
	path := m.selectedSubmodulePath()
	if path == "" {
		return nil
	}
	for _, module := range m.Submodules.Modules {
		if module.Path == path && (module.State == submodules.StateUninitialized || module.State == submodules.StateMissing) {
			m.Status = "submodule is not initialized: " + platform.SafeText(path)
			return nil
		}
	}
	root, ctx, generation := m.Discovery.Root, m.commandContext(), m.repositoryGeneration
	child := filepath.Join(root, path)
	m.State, m.Status = StateOperationPending, "opening submodule "+platform.SafeText(path)
	return func() tea.Msg {
		discovery, err := git.Discover(ctx, child)
		return SubmoduleOpenedMsg{Generation: generation, Path: path, Discovery: discovery, Err: err}
	}
}

func (m *Model) returnToParentRepository() tea.Cmd {
	if len(m.repositoryParents) == 0 {
		return nil
	}
	parent := m.repositoryParents[len(m.repositoryParents)-1]
	m.repositoryParents = m.repositoryParents[:len(m.repositoryParents)-1]
	if err := m.setRepository(parent.Discovery); err != nil {
		m.State, m.Status = StateError, "return to parent repository: "+err.Error()
		return nil
	}
	m.State, m.Status = StateReady, "returned to "+platform.SafeText(parent.Label)
	if m.Workspace != nil {
		m.Workspace.Back()
	}
	return tea.Batch(m.refresh(), waitForRefresh(m.RefreshCoordinator), m.startWatcher())
}

func (m *Model) beginSubmoduleActions() {
	path := m.selectedSubmodulePath()
	if path == "" {
		m.Status = "select a configured submodule first"
		return
	}
	m.SubmoduleAction, m.SubmodulePath = "menu", path
	m.Status = "submodule " + platform.SafeText(path) + ": [i/u/s] single [I/U/Y] bulk all [1/2/3] bulk selected init/update/sync [d] deinit [x] remove [a] URL [r] retry failed [esc] cancel"
}

func (m *Model) updateSubmoduleKey(key string) tea.Cmd {
	if m.SubmoduleAction == "add-url" {
		switch key {
		case "esc":
			m.SubmoduleAction, m.SubmodulePath, m.SubmoduleInput = "", "", ""
			m.Status = "submodule add cancelled"
		case "backspace":
			m.SubmoduleInput = removeLastRune(m.SubmoduleInput)
		case "enter":
			if strings.TrimSpace(m.SubmoduleInput) == "" {
				m.Status = "submodule URL is required"
			} else {
				m.SubmoduleURL, m.SubmoduleAction = m.SubmoduleInput, "confirm-add"
				m.Status = "confirm add submodule " + platform.SafeText(m.SubmodulePath) + " from " + platform.SafeText(submodules.RedactURL(m.SubmoduleURL)) + "? (y/n)"
			}
		case "space", " ":
			m.SubmoduleInput += " "
		default:
			if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
				m.SubmoduleInput += key
			}
		}
		if m.SubmoduleAction == "add-url" {
			m.Status = "submodule URL: " + platform.SafeText(submodules.RedactURL(m.SubmoduleInput))
		}
		return nil
	}
	if strings.HasPrefix(m.SubmoduleAction, "confirm-") {
		switch key {
		case "y", "Y":
			m.SubmoduleAction = strings.TrimPrefix(m.SubmoduleAction, "confirm-")
			m.State, m.Status = StateOperationPending, "running submodule "+m.SubmoduleAction
			return m.runSubmoduleOperation()
		case "n", "N", "esc":
			m.SubmoduleAction, m.SubmodulePath, m.SubmoduleInput, m.SubmoduleURL = "", "", "", ""
			m.Status = "submodule operation cancelled"
		}
		return nil
	}
	if m.SubmoduleAction == "bulk-confirm" {
		switch key {
		case "y", "Y":
			m.SubmoduleAction = "bulk-running"
			m.State, m.Status = StateOperationPending, "running bulk submodule "+m.BulkSubmoduleAction
			return m.runBulkSubmodule()
		case "n", "N", "esc":
			m.SubmoduleAction, m.BulkSubmoduleAction, m.BulkSubmodulePaths = "", "", nil
			m.Status = "bulk submodule operation cancelled"
		}
		return nil
	}
	if m.SubmoduleAction == "bulk-running" {
		if key == "esc" && m.BulkSubmoduleCancel != nil {
			m.BulkSubmoduleCancel()
			m.Status = "bulk submodule cancellation requested"
		}
		return nil
	}
	if m.SubmoduleAction != "menu" {
		return nil
	}
	switch key {
	case "esc":
		m.SubmoduleAction, m.SubmodulePath = "", ""
		m.Status = "submodule actions cancelled"
	case "i":
		m.SubmoduleAction, m.State, m.Status = "initialize", StateOperationPending, "initializing submodule"
		return m.runSubmoduleOperation()
	case "u":
		m.SubmoduleAction, m.State, m.Status = "update", StateOperationPending, "updating submodule"
		return m.runSubmoduleOperation()
	case "s":
		m.SubmoduleAction, m.State, m.Status = "sync", StateOperationPending, "syncing submodule"
		return m.runSubmoduleOperation()
	case "I":
		m.beginBulkSubmodule(submodules.BulkInitialize, true)
	case "U":
		m.beginBulkSubmodule(submodules.BulkUpdate, true)
	case "Y":
		m.beginBulkSubmodule(submodules.BulkSync, true)
	case "1":
		m.beginBulkSubmodule(submodules.BulkInitialize, false)
	case "2":
		m.beginBulkSubmodule(submodules.BulkUpdate, false)
	case "3":
		m.beginBulkSubmodule(submodules.BulkSync, false)
	case "r":
		m.retryBulkSubmodule()
	case "d":
		m.SubmoduleAction = "confirm-deinit"
		m.Status = "confirm deinit of exact submodule path " + platform.SafeText(m.SubmodulePath) + "? (y/n)"
	case "x":
		m.SubmoduleAction = "confirm-remove"
		m.Status = "confirm remove exact submodule path " + platform.SafeText(m.SubmodulePath) + "? (y/n)"
	case "a":
		m.SubmoduleAction, m.SubmoduleInput = "add-url", ""
		m.Status = "submodule URL: "
	}
	return nil
}

func (m *Model) beginBulkSubmodule(action submodules.BulkAction, all bool) {
	paths := make([]string, 0)
	if all {
		for _, module := range m.Submodules.Modules {
			paths = append(paths, module.Path)
		}
	} else if path := m.selectedSubmodulePath(); path != "" {
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		m.Status = "select a configured submodule first"
		return
	}
	m.BulkSubmoduleAction = string(action)
	m.BulkSubmodulePaths = paths
	m.SubmoduleAction = "bulk-confirm"
	m.Status = fmt.Sprintf("preview bulk %s for %d submodule(s)? (y/n)", action, len(paths))
}

func (m *Model) retryBulkSubmodule() {
	if m.BulkSubmoduleOutcome == nil {
		m.Status = "no bulk submodule result to retry"
		return
	}
	paths := make([]string, 0)
	for _, item := range m.BulkSubmoduleOutcome.Items {
		if item.State == submodules.ItemFailed {
			paths = append(paths, item.Path)
		}
	}
	if len(paths) == 0 {
		m.Status = "no failed bulk submodules to retry"
		return
	}
	m.BulkSubmoduleAction = string(m.BulkSubmoduleOutcome.Action)
	m.BulkSubmodulePaths = paths
	m.SubmoduleAction = "bulk-confirm"
	m.Status = fmt.Sprintf("preview retry of %d failed submodule(s)? (y/n)", len(paths))
}

func (m *Model) runBulkSubmodule() tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.BulkSubmoduleCancel = cancel
	generation := m.repositoryGeneration
	action := submodules.BulkAction(m.BulkSubmoduleAction)
	paths := append([]string(nil), m.BulkSubmodulePaths...)
	runner := git.NewRunner(m.Discovery.Root)
	var outcome submodules.BulkOutcome
	command := m.OperationEngine.Command(ctx, fmt.Sprintf("submodule-bulk-%s-%d", action, generation), m.Discovery.Root, "bulk submodule "+string(action), 30*time.Minute, func(ctx context.Context) error {
		outcome = submodules.Bulk(ctx, runner, submodules.BulkRequest{Repository: m.Discovery.Root, Paths: paths, Action: action})
		return nil
	})
	return func() tea.Msg {
		_ = command()
		return BulkSubmoduleFinishedMsg{Generation: generation, Outcome: outcome}
	}
}

func (m Model) runSubmoduleOperation() tea.Cmd {
	if m.OperationEngine == nil {
		m.OperationEngine = operations.New(4)
	}
	ctx, generation := m.commandContext(), m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	action, path, rawURL := m.SubmoduleAction, m.SubmodulePath, m.SubmoduleURL
	request := submodules.Request{Repository: m.Discovery.Root, Path: path}
	var outcome submodules.Outcome
	command := m.OperationEngine.Command(ctx, fmt.Sprintf("submodule-%s-%d", action, generation), m.Discovery.Root, "submodule "+action, 10*time.Minute, func(ctx context.Context) error {
		switch action {
		case "initialize":
			outcome = submodules.Initialize(ctx, runner, request)
		case "update":
			outcome = submodules.Update(ctx, runner, request)
		case "sync":
			outcome = submodules.Sync(ctx, runner, request)
		case "deinit":
			outcome = submodules.Deinit(ctx, runner, submodules.RemoveRequest{Request: request, ConfirmedPath: path})
		case "remove":
			outcome = submodules.Remove(ctx, runner, submodules.RemoveRequest{Request: request, ConfirmedPath: path})
		case "add":
			outcome = submodules.Add(ctx, runner, submodules.AddRequest{Repository: m.Discovery.Root, Path: path, URL: rawURL})
		default:
			outcome.Err = fmt.Errorf("unknown submodule action %q", action)
		}
		return outcome.Err
	})
	return func() tea.Msg {
		result := command()
		if outcome.Err == nil {
			outcome.Err = result.Result.Err
		}
		return SubmoduleFinishedMsg{Generation: generation, Outcome: outcome}
	}
}

func shortSHA(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func (m *Model) updateComposerKey(key string) tea.Cmd {
	if key == "ctrl+x" && m.HistoricalRebaseAction != "" {
		m.State, m.Status = StateOperationPending, "aborting historical rebase"
		return m.abortHistoricalRebase()
	}
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
		if m.HistoricalPatchMode {
			m.HistoricalPatchMode, m.HistoricalPatch, m.HistoricalPatchTarget, m.HistoricalPatchPath = false, nil, "", ""
		}
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
	case "enter":
		if !m.HistoricalPatchMode {
			return nil
		}
		if m.Hunks.Selection.Count() == 0 {
			m.Status = "select at least one historical line or hunk"
			return nil
		}
		selected, err := m.Hunks.Selection.BuildPatch(m.Hunks.Files)
		if err != nil {
			m.Status = "historical patch: " + err.Error()
			return nil
		}
		m.HistoricalPatch = append([]byte(nil), selected...)
		m.HistoricalPatchMode = false
		m.Workspace.Back()
		return m.openHistoricalRebase(rebase.Edit)
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

func (m *Model) updateRebaseKey(key string) tea.Cmd {
	if m.RebaseAutosquashConfirm {
		switch key {
		case "y":
			m.RebaseAutosquashConfirm = false
			m.State, m.Status = StateOperationPending, "starting autosquash rebase"
			return m.startRebase(true)
		case "n", "esc":
			m.RebaseAutosquashConfirm = false
			m.Status = "autosquash cancelled"
		}
		return nil
	}
	if m.RebaseConfirmAction != "" {
		switch key {
		case "y":
			action := m.RebaseConfirmAction
			m.RebaseConfirmAction = ""
			if err := m.Rebase.ApplyAction(action, true); err != nil {
				m.Status = err.Error()
			} else {
				m.Status = "rebase plan updated"
			}
		case "n", "esc":
			m.RebaseConfirmAction = ""
			m.Status = "rebase edit cancelled"
		}
		return nil
	}
	switch key {
	case "esc":
		m.Workspace.Back()
		m.Status = "rebase cancelled"
	case "b":
		m.Rebase.BaseMode = !m.Rebase.BaseMode
	case "enter":
		if m.Rebase.BaseMode {
			if err := m.Rebase.SetBase(m.Rebase.BaseSelected); err != nil {
				m.Status = err.Error()
			} else {
				m.Rebase.BaseMode = false
				m.Status = "rebase base selected: " + m.Rebase.Base.Ref
			}
			break
		}
		m.State, m.Status = StateOperationPending, "starting interactive rebase"
		return m.startRebase(false)
	case "a":
		if (m.Rebase.Published || m.Rebase.ReachableRemote) && m.Rebase.Base.Ref != "" {
			m.RebaseAutosquashConfirm = true
			m.Status = "confirm autosquash rewrite of published history? (y/n)"
			return nil
		}
		m.State, m.Status = StateOperationPending, "starting autosquash rebase"
		return m.startRebase(true)
	case "j", "down":
		m.Rebase.Move(1)
	case "k", "up":
		m.Rebase.Move(-1)
	case "space":
		m.Rebase.ToggleMark()
	case "<", "[":
		if err := m.Rebase.MoveSelection(-1); err != nil {
			m.Status = err.Error()
		}
	case ">", "]":
		if err := m.Rebase.MoveSelection(1); err != nil {
			m.Status = err.Error()
		}
	case "p", "s", "f", "r", "e", "d":
		action := map[string]rebase.Action{"p": rebase.Pick, "s": rebase.Squash, "f": rebase.Fixup, "r": rebase.Reword, "e": rebase.Edit, "d": rebase.Drop}[key]
		if (m.Rebase.Published || m.Rebase.ReachableRemote) && action != rebase.Pick {
			m.RebaseConfirmAction = action
			m.Status = "confirm rewrite of published history? (y/n)"
			return nil
		}
		if err := m.Rebase.ApplyAction(action, false); err != nil {
			m.Status = err.Error()
		} else {
			m.Status = "rebase plan updated"
		}
	}
	return nil
}

func (m *Model) updateConflictKey(key string) tea.Cmd {
	action := conflictview.Key(key)
	switch action {
	case conflictview.ActionNextConflict:
		m.Conflict.Move(1)
		return m.loadConflictContent()
	case conflictview.ActionPreviousConflict:
		m.Conflict.Move(-1)
		return m.loadConflictContent()
	case conflictview.ActionNextHunk:
		m.Conflict.MoveHunk(1, max(1, m.Conflict.RegionCount()))
	case conflictview.ActionPreviousHunk:
		m.Conflict.MoveHunk(-1, max(1, m.Conflict.RegionCount()))
	case conflictview.ActionStatus:
		m.Workspace.Navigate(workspace.Status, "Status")
	case conflictview.ActionChooseOurs, conflictview.ActionChooseTheirs, conflictview.ActionChooseBoth, conflictview.ActionMarkResolved, conflictview.ActionRestoreUnresolved:
		selected, ok := m.Conflict.SelectedConflict()
		if !ok {
			m.Status = "no conflict selected"
			return nil
		}
		choice := map[conflictview.Action]git.ConflictChoice{
			conflictview.ActionChooseOurs:        git.ChooseOurs,
			conflictview.ActionChooseTheirs:      git.ChooseTheirs,
			conflictview.ActionChooseBoth:        git.ChooseBoth,
			conflictview.ActionMarkResolved:      git.MarkResolved,
			conflictview.ActionRestoreUnresolved: git.RestoreUnresolved,
		}[action]
		m.State, m.Status = StateOperationPending, "applying conflict action: "+string(choice)
		return m.resolveConflict(append([]byte(nil), selected.Path...), choice)
	case conflictview.ActionRegionOurs, conflictview.ActionRegionTheirs, conflictview.ActionRegionBoth:
		selected, selectedOK := m.Conflict.SelectedConflict()
		region, expectedHash, regionOK := m.Conflict.SelectedRegion()
		if !selectedOK || !regionOK {
			m.Status = "no parsed conflict region selected"
			return nil
		}
		choice := map[conflictview.Action]conflicts.Choice{
			conflictview.ActionRegionOurs:   conflicts.ChoiceOurs,
			conflictview.ActionRegionTheirs: conflicts.ChoiceTheirs,
			conflictview.ActionRegionBoth:   conflicts.ChoiceBoth,
		}[action]
		m.State, m.Status = StateOperationPending, "applying conflict region action"
		return m.resolveConflictRegion(append([]byte(nil), selected.Path...), expectedHash, region, choice)
	case conflictview.ActionEditExternal:
		return m.openConflictEditor()
	case conflictview.ActionRegionManual:
		return m.openConflictEditor()
	case conflictview.ActionContinue, conflictview.ActionAbort, conflictview.ActionSkip:
		if m.Conflict.Operation == sequencer.KindUnknown {
			m.Status = "operation lifecycle is unavailable"
			return nil
		}
		recovery := m.Conflict.RecoveryActions()
		if (action == conflictview.ActionContinue && !recovery.Continue) ||
			(action == conflictview.ActionAbort && !recovery.Abort) ||
			(action == conflictview.ActionSkip && !recovery.Skip) {
			m.Status = "operation lifecycle action is unavailable"
			return nil
		}
		actionName := "continue"
		switch action {
		case conflictview.ActionAbort:
			actionName = "abort"
		case conflictview.ActionSkip:
			actionName = "skip"
		}
		m.State, m.Status = StateOperationPending, actionName+" "+m.Conflict.Operation.String()
		return m.conflictLifecycle(actionName)
	case conflictview.ActionNone:
		return nil
	default:
		// Resolution and lifecycle actions remain coordinator intents until the
		// repository-scoped mutation boundary is available. Do not change the
		// snapshot optimistically.
		m.Status = "conflict action pending coordinator: " + key
	}
	return nil
}

func (m *Model) openConflictEditor() tea.Cmd {
	selected, ok := m.Conflict.SelectedConflict()
	if !ok {
		m.Status = "no conflict selected"
		return nil
	}
	command, err := git.NewRunner(m.Discovery.Root).ExternalMergeToolCommand(append([]byte(nil), selected.Path...))
	if err != nil {
		m.Status = err.Error()
		return nil
	}
	m.Status = "external merge tool active"
	generation := m.repositoryGeneration
	return tea.ExecProcess(command, func(processErr error) tea.Msg {
		return OperationFinishedMsg{Name: "external merge tool", Repository: generation, Err: processErr}
	})
}

func (m Model) conflictLifecycle(action string) tea.Cmd {
	runner, ctx, generation, kind := git.NewRunner(m.Discovery.Root), m.commandContext(), m.repositoryGeneration, m.Conflict.Operation
	operation := &history.OperationRecord{
		Repository: m.Discovery.Root,
		Kind:       kind.String(),
		Args:       history.RedactArgs([]string{kind.String(), "--" + action}),
		Target:     m.Conflict.Target,
		OldHead:    m.Snapshot.Branch.OID,
		Refs:       []string{m.Snapshot.Branch.Name},
	}
	return func() tea.Msg {
		started := time.Now()
		_, err := runner.OperationLifecycle(ctx, kind, action)
		completed := *operation
		completed.Duration = time.Since(started)
		attachLatestRecoveryPoint(ctx, runner, &completed)
		return OperationFinishedMsg{Name: action + " " + kind.String(), Repository: generation, Operation: &completed, Err: err}
	}
}

func (m *Model) loadConflictContent() tea.Cmd {
	selected, ok := m.Conflict.SelectedConflict()
	if !ok || m.Discovery.Root == "" {
		return nil
	}
	if m.ConflictContentCancel != nil {
		m.ConflictContentCancel()
	}
	ctx, cancel := context.WithCancel(m.commandContext())
	m.ConflictContentCancel = cancel
	m.ConflictContentLoading = true
	m.ConflictContentRequest++
	request, generation := m.ConflictContentRequest, m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	path := append([]byte(nil), selected.Path...)
	return func() tea.Msg {
		selected.Path = path
		content, err := git.LoadConflictContent(ctx, runner, selected, int(m.DiffMaxBytes))
		return ConflictContentReadyMsg{Content: content, Generation: generation, Request: request, Err: err}
	}
}

func conflictViewContent(content git.Content) conflictview.Content {
	return conflictview.Content{Text: string(content.Bytes), Binary: content.Binary, InvalidUTF8: content.InvalidUTF8, Truncated: content.Truncated, Missing: content.Missing, Hash: content.Hash, Regions: content.Regions}
}

func (m Model) resolveConflict(path []byte, choice git.ConflictChoice) tea.Cmd {
	runner, ctx, generation := git.NewRunner(m.Discovery.Root), m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := runner.ResolveConflict(ctx, path, choice)
		return OperationFinishedMsg{Name: "conflict " + string(choice), Repository: generation, Err: err}
	}
}

func (m Model) resolveConflictRegion(path []byte, expectedHash [32]byte, region int, choice conflicts.Choice) tea.Cmd {
	runner, ctx, generation := git.NewRunner(m.Discovery.Root), m.commandContext(), m.repositoryGeneration
	return func() tea.Msg {
		_, err := runner.ResolveConflictRegion(ctx, path, expectedHash, region, choice, nil)
		return OperationFinishedMsg{Name: "conflict region", Repository: generation, Err: err}
	}
}

func (m Model) currentView() workspace.View {
	if m.Workspace == nil {
		return workspace.Status
	}
	view, _, _, _ := m.Workspace.Snapshot()
	return view
}

func (m Model) recoveryWorkspace() bool {
	view := m.currentView()
	return view == workspace.Conflict || view == workspace.CherryPick
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
		if m.currentView() == workspace.Repositories && m.RepositoryBatchConfirm {
			switch v.String() {
			case "y", "Y":
				m.RepositoryBatchConfirm = false
				m.State = StateOperationPending
				if m.RepositoryBatchAction == multirepo.ActionPull {
					m.Status = "pulling discovered repositories with " + m.RepositoryBatchStrategy
				} else {
					m.Status = "fetching discovered repositories"
				}
				return m, m.runRepositoryBatchFetch()
			case "n", "N", "esc":
				m.RepositoryBatchConfirm, m.RepositoryBatchRetry = false, false
				m.Status = "batch fetch cancelled"
			}
			return m, nil
		}
		if m.CustomCommandForm != nil {
			return m, m.updateCustomCommandForm(v.String())
		}
		if m.currentView() == workspace.Journal && m.JournalFilterMode {
			return m, m.updateJournalFilterKey(v.String())
		}
		if m.currentView() == workspace.GitHub && m.GitHubCreateMode {
			return m, m.updateGitHubCreateKey(v.String())
		}
		if m.currentView() == workspace.GitHub && m.GitHubIssueMode {
			return m, m.updateGitHubIssueKey(v.String())
		}
		if m.currentView() == workspace.GitHub && m.GitHubCreateConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubCreateConfirm = false
				m.State, m.Status = StateOperationPending, "creating GitHub pull request"
				return m, m.createGitHubPullRequest()
			case "n", "N", "esc":
				m.GitHubCreateConfirm = false
				m.Status = "GitHub PR creation cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.GitHubIssueConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubIssueConfirm = false
				m.State, m.Status = StateOperationPending, "creating GitHub issue"
				return m, m.createGitHubIssue()
			case "n", "N", "esc":
				m.GitHubIssueConfirm = false
				m.Status = "GitHub issue creation cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.GitHubMergeMode {
			return m, m.updateGitHubMergeKey(v.String())
		}
		if m.currentView() == workspace.GitHub && m.GitHubMergeConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubMergeConfirm = false
				m.State, m.Status = StateOperationPending, "merging GitHub pull request"
				return m, m.mergeGitHubPullRequest()
			case "n", "N", "esc":
				m.GitHubMergeConfirm = false
				m.Status = "GitHub merge cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.GitHubBranchDeleteConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubBranchDeleteConfirm = false
				m.State, m.Status = StateOperationPending, "deleting remote GitHub branch"
				return m, m.deleteGitHubBranch()
			case "n", "N", "esc":
				m.GitHubBranchDeleteConfirm = false
				m.State, m.Status = StateReady, "remote branch deletion cancelled; local refs unchanged"
				m.GitHubBranchDeleteTarget = ""
				return m, m.loadGitHub()
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.GitHubReviewMode {
			return m, m.updateGitHubReviewKey(v.String())
		}
		if m.currentView() == workspace.GitHub && m.GitHubReviewConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubReviewConfirm = false
				m.State, m.Status = StateOperationPending, "submitting GitHub review"
				return m, m.submitGitHubReview()
			case "n", "N", "esc":
				m.GitHubReviewConfirm = false
				m.Status = "GitHub review cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.GitHubCheckActionConfirm {
			switch v.String() {
			case "y", "Y":
				m.GitHubCheckActionConfirm = false
				m.State, m.Status = StateOperationPending, "requesting GitHub check "+m.GitHubCheckAction
				return m, m.runGitHubCheckAction()
			case "n", "N", "esc":
				m.GitHubCheckActionConfirm = false
				m.Status = "GitHub check action cancelled"
			}
			return m, nil
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
		if m.currentView() == workspace.Rebase {
			return m, m.updateRebaseKey(v.String())
		}
		if m.currentView() == workspace.Bisect {
			return m, m.updateBisectKey(v.String())
		}
		if m.currentView() == workspace.Status && m.SubmoduleAction != "" {
			return m, m.updateSubmoduleKey(v.String())
		}
		if m.recoveryWorkspace() {
			return m, m.updateConflictKey(v.String())
		}
		if m.currentView() == workspace.Gitignore {
			if m.GitignoreCreateConfirm {
				switch v.String() {
				case "y":
					m.GitignoreCreateConfirm = false
					m.Gitignore.SetPreview("")
					if m.GitignoreMissing && m.GitignoreMutationAction == "add" {
						m.GitignoreReturnToStatus = true
						m.Workspace.Navigate(workspace.Status, "Status")
					}
					m.State, m.Status = StateOperationPending, "applying gitignore "+m.GitignoreMutationAction
					return m, m.executeGitignoreMutation(m.GitignoreCreatePlan, m.GitignoreMutationAction)
				case "n", "esc":
					m.GitignoreCreateConfirm = false
					m.Gitignore.SetPreview("")
					m.Status = "gitignore creation cancelled"
				}
				return m, nil
			}
			if v.String() == "r" {
				m.State, m.Status = StateOperationPending, "refreshing gitignore catalog"
				return m, m.refreshGitignoreCatalog()
			}
			if v.String() == "b" {
				source, err := catalog.UseBundled()
				if err != nil {
					m.Status = "bundled catalog: " + err.Error()
					return m, nil
				}
				m.GitignoreCatalog, m.GitignoreCatalogSource = source.Catalog, source.Kind
				m.Status = "using bundled gitignore catalog"
				return m, m.openGitignore()
			}
			if v.String() == "a" || v.String() == "p" || v.String() == "d" || v.String() == "u" || v.String() == "m" {
				action := map[string]string{"a": "add", "p": "add", "d": "remove", "u": "update", "m": "adopt"}[v.String()]
				if v.String() == "p" && m.GitignoreMissing {
					action = "add"
				}
				if m.GitignoreMissing && action != "add" {
					m.Status = "only add is available before .gitignore exists"
					return m, nil
				}
				if len(m.Gitignore.SelectedEntries()) == 0 {
					m.Status = "select at least one template"
					return m, nil
				}
				m.GitignoreMutationAction = action
				m.State, m.Status = StateOperationPending, "building gitignore creation preview"
				if m.GitignoreMissing {
					return m, m.previewGitignoreCreate()
				}
				return m, m.previewGitignoreMutation(action)
			}
			if v.String() == "esc" {
				m.Workspace.Back()
				m.Status = "gitignore browser closed"
				return m, nil
			}
			m.Gitignore.UpdateKey(v.String())
			return m, nil
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
		if m.currentView() == workspace.Journal && m.UndoConfirm {
			switch v.String() {
			case "y", "Y":
				m.UndoConfirm, m.State, m.Status = false, StateOperationPending, "undoing selected commit; preserving index and worktree"
				return m, m.undoJournalOperation()
			case "n", "N", "esc":
				m.UndoConfirm, m.UndoRecord, m.Status = false, nil, "undo cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Journal && m.RedoConfirm {
			switch v.String() {
			case "y", "Y":
				m.RedoConfirm, m.State, m.Status = false, StateOperationPending, "redoing selected commit; preserving index and worktree"
				return m, m.redoJournalOperation()
			case "n", "N", "esc":
				m.RedoConfirm, m.RedoRecord, m.Status = false, nil, "redo cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Journal && m.JournalCancelConfirm {
			switch v.String() {
			case "y", "Y":
				id := m.JournalCancelID
				m.JournalCancelID, m.JournalCancelConfirm = "", false
				if m.OperationEngine != nil && m.OperationEngine.Cancel(id) {
					m.Status = "cancellation requested for " + platform.SafeText(id)
				} else {
					m.Status = "operation is no longer running"
				}
			case "n", "N", "esc":
				m.JournalCancelID, m.JournalCancelConfirm = "", false
				m.Status = "operation cancellation cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Journal && m.JournalRetryConfirm {
			switch v.String() {
			case "y", "Y":
				id := m.JournalRetryID
				m.JournalRetryID, m.JournalRetryConfirm = "", false
				if m.OperationEngine == nil {
					m.Status = "operation retry is unavailable"
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "retrying journal operation"
				retry := m.OperationEngine.RetryCommand(m.commandContext(), id)
				return m, func() tea.Msg { return retry() }
			case "n", "N", "esc":
				m.JournalRetryID, m.JournalRetryConfirm = "", false
				m.Status = "operation retry cancelled"
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
		if m.currentView() == workspace.Tags && m.TagsFilterMode {
			m.updateTagsFilter(v.String())
			return m, nil
		}
		if m.currentView() == workspace.Tags && (m.TagCreateMode != "" || m.TagDeleteMode) {
			return m, m.updateTagMutationKey(v.String())
		}
		if m.currentView() == workspace.Tags && m.TagCheckoutConfirm {
			switch v.String() {
			case "y", "Y":
				m.TagCheckoutConfirm = false
				m.State, m.Status = StateOperationPending, "checking out tag "+platform.SafeText(m.TagCheckoutTarget)+" detached"
				return m, m.checkoutSelectedTag()
			case "n", "N", "esc":
				m.TagCheckoutConfirm, m.TagCheckoutTarget = false, ""
				m.Status = "tag checkout cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Tags && m.TagWorktreeMode {
			return m, m.updateTagWorktreeKey(v.String())
		}
		if m.currentView() == workspace.PathHistory {
			switch v.String() {
			case "esc":
				if m.PathHistoryCancel != nil {
					m.PathHistoryCancel()
					m.PathHistoryCancel = nil
				}
				m.PathHistoryLoading = false
				m.Workspace.Back()
				m.Status = "returned from path history"
				return m, nil
			case "f":
				m.PathHistory.Follow = !m.PathHistory.Follow
				m.State, m.Status = StateOperationPending, "reloading path history"
				return m, m.openPathHistory(m.PathHistory.Path, m.PathHistory.Follow)
			case "]":
				if m.PathHistory.HasMore && !m.PathHistoryLoading {
					m.State, m.Status = StateOperationPending, "loading more path history"
					return m, m.loadPathHistoryPage(len(m.PathHistory.Entries))
				}
				return m, nil
			case "enter":
				return m, m.inspectSelectedPathHistory()
			case "Y":
				return m, m.compareSelectedPathHistory()
			}
		}
		if m.currentView() == workspace.Blame {
			switch v.String() {
			case "esc":
				if m.BlameCancel != nil {
					m.BlameCancel()
					m.BlameCancel = nil
				}
				m.BlameLoading = false
				m.Workspace.Back()
				m.Status = "returned from blame"
				return m, nil
			case "]":
				return m, m.loadBlamePage()
			case "enter":
				return m, m.inspectSelectedBlame()
			}
		}
		if m.currentView() == workspace.Branches && m.RemoteBranchAction == "track" {
			return m, m.updateRemoteBranchKey(v.String())
		}
		if m.currentView() == workspace.Branches && m.BranchResetPrompt {
			return m, m.updateBranchResetKey(v.String())
		}
		if m.currentView() == workspace.Branches && m.RemoteBranchConfirm {
			switch v.String() {
			case "y", "Y":
				branch, action := m.RemoteBranchTarget, m.RemoteBranchAction
				m.RemoteBranchConfirm, m.RemoteBranchAction, m.RemoteBranchTarget = false, "", branches.Branch{}
				m.State, m.Status = StateOperationPending, "applying remote branch action"
				if action == "detached" {
					return m, m.remoteBranchMutation("checked out detached", branch, "", "")
				}
				return m, m.remoteBranchMutation("deleted remote branch", branch, "", branch.RemoteName+"/"+branch.RemoteBranch)

			case "n", "N", "esc":
				m.RemoteBranchConfirm, m.RemoteBranchAction, m.RemoteBranchTarget = false, "", branches.Branch{}
				m.Status = "remote branch action cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.GitHub && m.RemoteBranchConfirm {
			switch v.String() {
			case "y", "Y":
				branch := m.RemoteBranchTarget
				m.RemoteBranchConfirm, m.RemoteBranchAction, m.RemoteBranchTarget = false, "", branches.Branch{}
				m.State, m.Status = StateOperationPending, "checking out provider-declared branch"
				return m, m.remoteBranchMutation("checked out detached", branch, "", "")
			case "n", "N", "esc":
				m.RemoteBranchConfirm, m.RemoteBranchAction, m.RemoteBranchTarget = false, "", branches.Branch{}
				m.Status = "GitHub branch checkout cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Branches && m.BranchRecoveryConfirm {
			switch v.String() {
			case "y", "Y":
				target := m.BranchRecoveryTarget
				m.BranchRecoveryAction, m.BranchRecoveryTarget, m.BranchRecoveryConfirm = "", "", false
				m.State, m.Status = StateOperationPending, "fast-forwarding to "+platform.SafeText(target)
				return m, m.fastForwardBranch(target)
			case "n", "N", "esc":
				m.BranchRecoveryAction, m.BranchRecoveryTarget, m.BranchRecoveryConfirm = "", "", false
				m.Status = "branch recovery cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Branches && (m.BranchCreateMode || m.BranchRenameMode || m.BranchUpstreamMode || m.BranchDeleteMode || m.BranchMergeMode) {
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
		if m.currentView() == workspace.Remotes && (m.RemoteTagMode || m.RemoteTagDeleteMode) {
			switch v.String() {
			case "esc":
				m.RemoteTagMode, m.RemoteTagDeleteMode, m.RemoteTag = false, false, ""
				m.Status = "remote tag action cancelled"
			case "backspace":
				m.RemoteTag = removeLastRune(m.RemoteTag)
			case "enter":
				if strings.TrimSpace(m.RemoteTag) == "" {
					m.Status = "tag name is required"
				} else if m.RemoteTagDeleteMode {
					m.RemoteTagDeleteMode, m.RemoteTagDeleteConfirm = false, true
					m.Status = "confirm DELETE remote tag " + strings.TrimSpace(m.RemoteTag) + "? (y/n)"
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
			} else if m.RemoteTagDeleteMode {
				m.Status = "remote tag to DELETE: " + m.RemoteTag
			}
			return m, nil
		}
		if m.currentView() == workspace.Remotes && remoteMutationInputMode(m.RemoteMutationMode) {
			return m, m.updateRemoteMutationKey(v.String())
		}
		if m.currentView() == workspace.Remotes && (m.RemoteMutationConfirm || m.RemotePruneConfirm) {
			switch v.String() {
			case "y", "Y":
				m.RemoteMutationConfirm, m.RemotePruneConfirm = false, false
				m.State, m.Status = StateOperationPending, "applying remote change"
				return m, m.executeRemoteLifecycle()
			case "n", "N", "esc":
				m.resetRemoteMutation()
				m.Status = "remote lifecycle action cancelled"
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
		if (m.currentView() == workspace.Log || m.currentView() == workspace.Reflog) && m.HistoryActionConfirm {
			switch v.String() {
			case "y":
				m.HistoryActionConfirm, m.State, m.Status = false, StateOperationPending, "checking out "+m.HistoryActionTarget
				return m, m.checkoutSelectedHistory()
			case "n", "esc":
				m.HistoryActionConfirm, m.HistoryActionTarget, m.Status = false, "", "history action cancelled"
			}
			return m, nil
		}
		if m.currentView() == workspace.Log && m.CherryPickConfirm {
			switch v.String() {
			case "y", "Y":
				m.CherryPickConfirm = false
				m.State, m.Status = StateOperationPending, "cherry-picking selected commits"
				return m, m.cherryPickSelectedHistory()
			case "n", "N", "esc":
				m.CherryPickConfirm, m.CherryPickCommits = false, nil
				m.Status = "cherry-pick cancelled"
			}
			return m, nil
		}
		if (m.currentView() == workspace.Log || m.currentView() == workspace.Reflog) && m.HistoryBranchCreating {
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
		if m.currentView() == workspace.Log && m.HistoryRevertParentMode {
			switch v.String() {
			case "esc":
				m.HistoryRevertParentMode, m.HistoryRevertParentInput, m.HistoryRevertParent, m.HistoryRevertParentMax = false, "", 0, 0
				m.HistoryRevertCommits, m.HistoryRevertTarget, m.Status = nil, "", "revert cancelled"
			case "backspace":
				m.HistoryRevertParentInput = removeLastRune(m.HistoryRevertParentInput)
			case "enter":
				parent, err := strconv.Atoi(strings.TrimSpace(m.HistoryRevertParentInput))
				if err != nil || parent < 1 || parent > m.HistoryRevertParentMax {
					m.Status = fmt.Sprintf("mainline parent must be between 1 and %d", m.HistoryRevertParentMax)
				} else {
					m.HistoryRevertParentMode, m.HistoryRevertParentInput, m.HistoryRevertParent = false, "", parent
					m.HistoryRevertConfirm, m.HistoryRevertInvalid = true, false
					m.Status = "type SHA " + m.HistoryRevertTarget + ": "
				}
			default:
				if len([]rune(v.String())) == 1 && v.String() >= "0" && v.String() <= "9" {
					m.HistoryRevertParentInput += v.String()
				}
			}
			if m.HistoryRevertParentMode {
				m.Status = fmt.Sprintf("revert %s mainline parent (1-%d): %s", m.HistoryRevertTarget, m.HistoryRevertParentMax, m.HistoryRevertParentInput)
			}
			return m, nil
		}
		if m.currentView() == workspace.Log && m.HistoryRevertConfirm {
			switch v.String() {
			case "esc":
				m.HistoryRevertConfirm, m.HistoryRevertTarget, m.HistoryRevertInput, m.HistoryRevertInvalid, m.HistoryRevertCommits, m.HistoryRevertParent = false, "", "", false, nil, 0
				m.Status = "revert cancelled"
			case "backspace":
				m.HistoryRevertInput = removeLastRune(m.HistoryRevertInput)
				m.HistoryRevertInvalid = false
			case "enter":
				if !(history.RevertConfirmation{SHA: m.HistoryRevertTarget}).Accept(m.HistoryRevertInput) {
					m.HistoryRevertInvalid = true
					m.Status = "type the exact SHA to revert"
					if len(m.HistoryRevertCommits) > 0 {
						m.Status = "type the exact ordered SHA list to revert"
					}
				} else {
					m.HistoryRevertConfirm, m.HistoryRevertInvalid, m.HistoryRevertRunning, m.State, m.Status = false, false, true, StateOperationPending, "reverting"
					return m, m.revertSelectedHistory()
				}
			default:
				if len([]rune(v.String())) == 1 {
					m.HistoryRevertInput += v.String()
				}
				m.HistoryRevertInvalid = false
			}
			if m.HistoryRevertConfirm && !m.HistoryRevertInvalid {
				label := "type SHA "
				if len(m.HistoryRevertCommits) > 0 {
					label = "type ordered SHAs "
				}
				m.Status = label + m.HistoryRevertTarget + ": " + m.HistoryRevertInput
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
		if m.currentView() == workspace.Remotes && m.RemoteTagDeleteConfirm {
			switch v.String() {
			case "y", "Y":
				m.RemoteTagDeleteConfirm, m.State, m.Status = false, StateOperationPending, "deleting remote tag"
				return m, m.deleteSelectedRemoteTag()
			case "n", "N", "esc":
				m.RemoteTagDeleteConfirm, m.RemoteTag, m.Status = false, "", "remote tag deletion cancelled"
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
		if m.currentView() == workspace.Reflog && v.String() == "]" {
			if m.Reflog.HasMore && !m.ReflogLoading {
				m.State, m.Status = StateOperationPending, "loading more reflog entries"
				return m, m.loadReflog(true)
			}
			return m, nil
		}
		for _, definition := range m.CustomCommands {
			if definition.Binding == v.String() {
				return m, m.runCustomCommand(definition.Name)
			}
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
			if m.currentView() == workspace.Compare && m.CompareCancel != nil {
				m.CompareCancel()
				m.CompareCancel = nil
				m.CompareLoading = false
			}
			if m.currentView() == workspace.Compare && m.ComparePatchCancel != nil {
				m.ComparePatchCancel()
				m.ComparePatchCancel = nil
				m.ComparePatchLoading = false
			}
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
			} else if m.currentView() == workspace.Status && len(m.repositoryParents) > 0 {
				return m, m.returnToParentRepository()
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
			if m.currentView() == workspace.Tags {
				m.cycleTagsSort()
				m.Status = "tag sort: " + m.TagsSort
				return m, nil
			}
			return m, m.navigate(workspace.Stashes, "Stashes")
		case "l":
			return m, m.navigate(workspace.Log, "History")
		case "n":
			if m.currentView() == workspace.GitHub {
				return m, m.startGitHubCreate()
			}
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
			if m.currentView() == workspace.Log {
				return m, m.openHistoricalRebase(rebase.Edit)
			}
			if m.PluginsEnabled {
				return m, m.navigate(workspace.Plugins, "Plugins")
			}
		case "W":
			if m.currentView() == workspace.GitHub {
				if len(m.GitHub.Checks.Runs) == 0 || m.GitHub.SelectedRun < 0 || m.GitHub.SelectedRun >= len(m.GitHub.Checks.Runs) || m.GitHub.Checks.Runs[m.GitHub.SelectedRun].URL == "" {
					m.Status = "no GitHub check URL available"
					return m, nil
				}
				run := m.GitHub.Checks.Runs[m.GitHub.SelectedRun]
				command, err := platform.OpenURLCommand(run.URL)
				if err != nil {
					m.Status = err.Error()
					return m, nil
				}
				m.Status = "opening GitHub check " + platform.SafeText(run.Name)
				return m, tea.ExecProcess(command, nil)
			}
			if m.currentView() == workspace.Log {
				return m, m.openHistoricalRebase(rebase.Reword)
			}
		case "w":
			if m.currentView() == workspace.Tags {
				m.TagWorktreeMode, m.TagWorktreePath = true, ""
				m.Status = "tag worktree path: "
				return m, nil
			}
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.WorktreeAddMode, m.WorktreeAddPath, m.WorktreeAddCommit = true, "", branch.Name
					m.Status = "worktree path for " + platform.SafeText(branch.Name) + ": "
					return m, m.navigate(workspace.Worktrees, "Worktrees")
				}
			}
			return m, m.navigate(workspace.Worktrees, "Worktrees")
		case "v":
			if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				if !m.HistoryRangeAnchorSet {
					m.HistoryRangeAnchor, m.HistoryRangeAnchorSet = m.History.Selected, true
					m.Status = fmt.Sprintf("history range start: row %d; press v at the end", m.History.Selected+1)
					return m, nil
				}
				start, end := m.HistoryRangeAnchor, m.History.Selected
				m.HistoryRangeAnchorSet = false
				if err := m.History.SelectRange(start, end); err != nil {
					m.Status = "history range: " + err.Error()
				} else {
					m.Status = fmt.Sprintf("selected history range rows %d-%d (%d commits)", min(start, end)+1, max(start, end)+1, m.History.Basket.Count())
				}
				return m, nil
			}
			return m, m.navigate(workspace.Repositories, "Repositories")
		case "A":
			if m.currentView() == workspace.GitHub {
				return m, m.startGitHubReview(provider.ReviewEventApprove)
			} else if m.currentView() == workspace.Remotes {
				m.resetRemoteMutation()
				m.RemoteMutationMode = "add-name"
				m.Status = "new remote name: "
				return m, nil
			} else if m.currentView() == workspace.Tags {
				m.startTagCreation(tags.CreateAnnotated)
				return m, nil
			} else if m.currentView() == workspace.Worktrees {
				m.WorktreeAddMode, m.WorktreeAddPath, m.WorktreeAddCommit = true, "", ""
				m.Status = "worktree path: "
			} else if m.currentView() == workspace.Rebase {
				if (m.Rebase.Published || m.Rebase.ReachableRemote) && m.Rebase.Base.Ref != "" {
					m.RebaseAutosquashConfirm = true
					m.Status = "confirm autosquash rewrite of published history? (y/n)"
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "starting autosquash rebase"
				return m, m.startRebase(true)
			}
		case "F":
			if m.currentView() == workspace.Repositories {
				return m, m.startRepositoryBatchFetch()
			} else if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				if m.Snapshot.Counts.Staged == 0 {
					m.Status = "stage changes before creating a fixup commit"
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "creating fixup commit"
				return m, m.createFixup()
			}
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if !branch.Current {
					m.Status = "fast-forward requires the checked-out branch"
				} else if branch.Upstream == "" {
					m.Status = "checked-out branch has no upstream"
				} else if branch.Behind == 0 {
					m.Status = "branch is not behind its upstream"
				} else {
					m.BranchRecoveryAction, m.BranchRecoveryTarget, m.BranchRecoveryConfirm = "fast-forward", branch.Upstream, true
					m.Status = "confirm fast-forward to " + platform.SafeText(branch.Upstream) + "? (y/n)"
				}
			}
		case "z":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if !branch.Current {
					m.Status = "branch reset requires the checked-out branch"
				} else {
					m.BranchResetPrompt, m.BranchResetInput = true, ""
					m.Status = "reset: soft <ref> or mixed <ref>"
				}
			}
		case "U":
			if m.currentView() == workspace.Status {
				m.State, m.Status = StateOperationPending, "unstaging all changes; working-tree content will be preserved"
				return m, m.mutateAll(false)
			}
		case "S":
			if m.currentView() == workspace.Tags {
				m.startTagCreation(tags.CreateSigned)
				return m, nil
			}
			if m.currentView() == workspace.Status {
				m.Files.CycleSort()
				m.rebuildStatusFileTree()
				m.Status = "file sort mode changed"
			}
		case "V":
			if m.currentView() == workspace.Status && m.DiffPath != "" {
				return m, m.openDiffMode(!m.DiffStaged)
			}
			if m.currentView() == workspace.Tags {
				return m, m.verifySelectedTag()
			}
		case "!":
			if m.currentView() == workspace.GitHub {
				return m, m.startGitHubCheckAction("rerun")
			} else if m.currentView() == workspace.Status {
				m.FileConflictOnly = !m.FileConflictOnly
				if m.FileConflictOnly {
					m.Files.SetConflictFilter(true)
					m.rebuildStatusFileTree()
					m.Status = "showing conflicted files only"
				} else {
					m.Files.SetFilter(m.FileFilterInput)
					m.rebuildStatusFileTree()
					m.Status = "conflict filter cleared"
				}
			}
		case "D":
			if m.currentView() == workspace.Tags {
				if selected, ok := m.selectedTag(); ok {
					m.TagDeleteMode, m.TagDeleteTarget, m.TagDeleteInput = true, selected.Name, ""
					m.Status = "type " + platform.SafeText(selected.Name) + " to confirm deletion: "
				}
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.resetRemoteMutation()
				m.RemoteMutationRemote, m.RemoteMutationMode, m.State, m.Status = remote, "remove-loading", StateOperationPending, "loading tracking branches for "+platform.SafeText(remote)
				return m, m.loadRemoteTracking(remote)
			} else if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.RemoteBranchAction, m.RemoteBranchTarget, m.RemoteBranchConfirm = "delete", branch, true
					m.Status = "confirm delete remote branch " + platform.SafeText(branch.RemoteName+"/"+branch.RemoteBranch) + "? (y/n)"
				} else if branch.Current || branch.OccupiedPath != "" {
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
			if m.currentView() == workspace.Compare {
				if !m.selectCompareRemote() {
					m.Status = "comparison remote is not loaded"
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "fetching comparison remote"
				return m, m.fetchSelectedRemote()
			} else if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "fetching"
				return m, m.fetchSelectedRemote()
			} else if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				m.HistoryInspectorPathMode, m.HistoryInspectorPath = true, ""
				m.Status = "path filter: "
			}
		case "m", "e", "o":
			if m.currentView() == workspace.GitHub && v.String() == "m" {
				return m, m.startGitHubMerge()
			}
			if m.currentView() == workspace.GitHub && v.String() == "o" {
				if command, err := platform.OpenURLCommand(m.GitHub.Pull.URL); err == nil {
					m.Status = "opening pull request"
					return m, tea.ExecProcess(command, nil)
				} else {
					m.Status = err.Error()
				}
				break
			}
			if m.currentView() == workspace.Compare {
				if !m.selectCompareRemote() {
					m.Status = "comparison remote is not loaded"
					return m, nil
				}
				strategy := map[string]string{"m": "merge", "e": "rebase", "o": "ff-only"}[v.String()]
				m.State, m.Status = StateOperationPending, "pulling comparison remote ("+strategy+")"
				return m, m.pullSelectedRemote(strategy)
			} else if m.currentView() == workspace.Remotes {
				strategy := map[string]string{"m": "merge", "e": "rebase", "o": "ff-only"}[v.String()]
				m.State, m.Status = StateOperationPending, "pulling "+strategy
				return m, m.pullSelectedRemote(strategy)
			}
		case "p":
			if m.currentView() == workspace.Compare {
				if !m.selectCompareRemote() {
					m.Status = "comparison remote is not loaded"
					return m, nil
				}
				m.State, m.Status = StateOperationPending, "preparing comparison push preview"
				return m, m.previewSelectedRemotePush()
			} else if m.currentView() == workspace.Remotes {
				m.State, m.Status = StateOperationPending, "preparing push preview"
				return m, m.previewSelectedRemotePush()
			}
			if m.currentView() == workspace.Stashes && m.Stashes.Selected >= 0 && m.Stashes.Selected < len(m.Stashes.Entries) {
				ref := m.Stashes.Entries[m.Stashes.Selected].Ref
				m.StashConfirmAction, m.StashConfirmRef = "pop", ref
				m.Status = "confirm pop " + ref + "? (y/n)"
			}
		case "u":
			if m.currentView() == workspace.Journal {
				if record := m.selectedUndoRecord(); record != nil {
					m.UndoRecord, m.UndoConfirm = record, true
					m.Status = "undo commit " + record.NewHead + " -> " + record.OldHead + "? (y/n)"
				} else {
					m.Status = "selected journal entry has no safe commit undo"
				}
			} else if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
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
			if m.currentView() == workspace.Repositories {
				return m, m.startRepositoryBatchPull()
			} else if m.currentView() == workspace.Log {
				m.prepareCherryPick()
			} else if m.currentView() == workspace.Status {
				return m, m.selectLowerPane("unpushed")
			} else if m.currentView() == workspace.Worktrees {
				m.WorktreeConfirmAction, m.WorktreeConfirmTarget = "prune", "repository"
				m.Status = "confirm prune stale worktrees? (y/n)"
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.RemoteForceConfirm, m.Status = true, "confirm force-with-lease push to "+remote+" for "+m.Snapshot.Branch.Name+" (y/n)"
			}
		case "K":
			if m.currentView() == workspace.Repositories && m.RepositoryBatchCancel != nil {
				m.RepositoryBatchCancel()
				m.Status = "cancelling repository batch"
			} else if m.currentView() == workspace.Journal {
				active := m.activeJournalOperations()
				if len(active) == 0 {
					m.Status = "no running journal operation to cancel"
				} else {
					m.JournalCancelID, m.JournalCancelConfirm = active[0].ID, true
					m.Status = "cancel running operation " + platform.SafeText(active[0].Name) + "? (y/n)"
				}
			} else if m.currentView() == workspace.GitHub {
				return m, m.startGitHubCheckAction("cancel")
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.resetRemoteMutation()
				m.RemoteMutationRemote, m.RemoteMutationMode, m.State, m.Status = remote, "prune-loading", StateOperationPending, "previewing stale refs for "+platform.SafeText(remote)
				return m, m.previewRemotePrune(remote)
			}
		case "L":
			if m.currentView() == workspace.GitHub {
				index := m.GitHub.SelectedRelease
				if index < 0 || index >= len(m.GitHub.Releases) || m.GitHub.Releases[index].URL == "" {
					m.Status = "no GitHub release URL available"
					return m, nil
				}
				release := m.GitHub.Releases[index]
				command, err := platform.OpenURLCommand(release.URL)
				if err != nil {
					m.Status = err.Error()
					return m, nil
				}
				m.Status = "opening GitHub release " + platform.SafeText(release.TagName)
				return m, tea.ExecProcess(command, nil)
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected]
				m.resetRemoteMutation()
				m.RemoteMutationRemote, m.RemoteMutationMode, m.Status = remote.Name, "set-url", "new URL for "+platform.SafeText(remote.Name)+" (current: "+platform.SafeText(remote.FetchURL)+"): "
				return m, nil
			}
			path := ""
			if m.currentView() == workspace.Status {
				path = string(m.Files.SelectedPath())
			} else if m.currentView() == workspace.PathHistory {
				path = m.PathHistory.Path
				if entry, ok := m.PathHistory.SelectedEntry(); ok && entry.NewPath != "" {
					path = entry.NewPath
				}
			} else if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				path = m.HistoryInspectorPath
				if path == "" {
					path = string(m.Files.SelectedPath())
				}
			}
			if path != "" {
				return m, m.openBlame(path, 1)
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
			if m.currentView() == workspace.Remotes {
				if m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
					m.RemoteTagDeleteMode, m.RemoteTag = true, ""
					m.Status = "remote tag to DELETE: "
				}
			} else if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.RemoteBranchAction, m.RemoteBranchTarget, m.RemoteBranchConfirm = "delete", branch, true
					m.Status = "confirm delete remote branch " + platform.SafeText(branch.RemoteName+"/"+branch.RemoteBranch) + "? (y/n)"
				} else if branch.Current || branch.OccupiedPath != "" {
					m.Status = "cannot delete checked-out or worktree-bound branch"
				} else {
					m.BranchDeleteMode, m.BranchDeleteTarget, m.BranchDeleteForce, m.BranchMutationInput = true, branch, true, ""
					m.Status = "type " + branch.Name + " to confirm FORCE deletion: "
				}
			}
		case "]":
			if m.currentView() == workspace.GitHub && len(m.GitHub.Comments) > 0 {
				m.GitHub.SelectComment(1)
				m.GitHubReplyCommentID = m.GitHub.Comments[m.GitHub.SelectedComment].ID
				m.Status = "selected GitHub comment reply target #" + fmt.Sprint(m.GitHubReplyCommentID)
			} else if m.currentView() == workspace.Status && m.StatusTreeMode {
				m.FileTree.ExpandAll()
				m.Status = "all directories expanded"
			} else if m.currentView() == workspace.Log && m.HistoryHasMore {
				m.State, m.Status = StateOperationPending, "loading more history"
				return m, m.loadHistoryPage(m.HistorySkip)
			}
		case "/":
			if m.currentView() == workspace.Journal {
				m.JournalFilterMode = true
				m.Status = "journal filter: " + m.JournalFilterInput
				return m, nil
			}
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
			if m.currentView() == workspace.Tags {
				m.TagsFilterMode, m.Status = true, "tag filter: "
			}
			if m.currentView() == workspace.Status {
				m.FileFilterMode, m.FileConflictOnly = true, false
				m.Files.SetFilter(m.FileFilterInput)
				m.rebuildStatusFileTree()
				m.Status = "file filter: " + m.FileFilterInput
			}
		case "x":
			if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) && m.Branches.Entries[m.Branches.Selected].Remote {
				branch := m.Branches.Entries[m.Branches.Selected]
				m.RemoteBranchAction, m.RemoteBranchTarget, m.RemoteBranchConfirm = "detached", branch, true
				m.Status = "confirm detached checkout of remote branch " + platform.SafeText(branch.RemoteName+"/"+branch.RemoteBranch) + "? (y/n)"
			} else if m.currentView() == workspace.GitHub {
				branch, err := m.githubCheckoutBranch()
				if err != nil {
					m.Status = "GitHub checkout unavailable: " + platform.SafeText(err.Error())
				} else {
					m.RemoteBranchAction, m.RemoteBranchTarget, m.RemoteBranchConfirm = "detached", branch, true
					m.Status = "confirm checkout of provider-declared " + platform.SafeText(branch.RemoteName+"/"+branch.RemoteBranch) + " detached? (y/n)"
				}
			} else if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryActionTarget = m.History.Rows[m.History.Selected].Commit.SHA
				m.HistoryActionConfirm = true
				m.Status = "checkout commit " + m.HistoryActionTarget + "? (y/n)"
			} else if m.currentView() == workspace.Reflog {
				if entry, ok := m.Reflog.SelectedEntry(); ok {
					m.HistoryActionTarget, m.HistoryActionConfirm = entry.SHA, true
					m.Status = "checkout recovery point " + entry.SHA + " (detached HEAD)? (y/n)"
				}
			} else if m.currentView() == workspace.Tags {
				if selected, ok := m.selectedTag(); ok {
					m.TagCheckoutTarget, m.TagCheckoutConfirm = selected.Name, true
					m.Status = "confirm detached checkout of tag " + platform.SafeText(selected.Name) + "? (y/n)"
				}
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
		case "J":
			if m.currentView() == workspace.Journal {
				m.JournalOffset = 0
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
			if m.currentView() == workspace.Reflog {
				if entry, ok := m.Reflog.SelectedEntry(); ok {
					m.HistoryBranchTarget, m.HistoryBranchName, m.HistoryBranchCreating = entry.SHA, "", true
					m.Status = "branch at " + entry.SHA + ": enter name"
				}
			}
		case "R":
			if m.currentView() == workspace.Repositories {
				return m, m.startRepositoryBatchRetry()
			} else if m.currentView() == workspace.GitHub {
				return m, m.startGitHubReview(provider.ReviewEventRequestChanges)
			} else if m.currentView() == workspace.Remotes && m.Remotes.Selected >= 0 && m.Remotes.Selected < len(m.Remotes.Dashboard.Remotes) {
				remote := m.Remotes.Dashboard.Remotes[m.Remotes.Selected].Name
				m.resetRemoteMutation()
				m.RemoteMutationRemote, m.RemoteMutationMode, m.State, m.Status = remote, "rename-loading", StateOperationPending, "loading tracking branches for "+platform.SafeText(remote)
				return m, m.loadRemoteTracking(remote)
			} else if m.currentView() == workspace.Journal {
				if record := m.selectedRedoRecord(); record != nil {
					m.RedoRecord, m.RedoConfirm = record, true
					m.Status = "redo commit " + record.NewHead + " -> " + record.OldHead + "? (y/n)"
				} else if event := m.selectedJournalEvent(); event != nil && event.Operation != nil {
					m.Status = "redo unavailable for " + event.Operation.Kind + "; reflog opened for guided recovery (d compare, B branch)"
					return m, m.navigate(workspace.Reflog, "Guided reflog recovery")
				} else {
					m.Status = "selected journal entry has no safe redo"
				}
			} else if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Remote {
					m.Status = "remote branch cannot be renamed here"
				} else {
					m.BranchRenameMode, m.BranchRenameOld, m.BranchMutationInput = true, branch.Name, ""
					m.Status = "rename " + branch.Name + " to: "
				}
			} else if m.currentView() == workspace.Log && m.History.Selected >= 0 && m.History.Selected < len(m.History.Rows) {
				m.HistoryRevertParentMode, m.HistoryRevertParentInput, m.HistoryRevertParent, m.HistoryRevertParentMax = false, "", 0, 0
				if m.History.Basket.Count() > 0 {
					m.HistoryRevertCommits = m.History.Basket.SHAs()
					mergeCount := 0
					mergeSHA, mergeParents := "", 0
					for _, sha := range m.HistoryRevertCommits {
						for _, row := range m.History.Rows {
							if row.Commit.SHA == sha && len(row.Commit.Parents) > 1 {
								mergeCount++
								mergeSHA, mergeParents = sha, len(row.Commit.Parents)
							}
						}
					}
					if mergeCount > 0 {
						if mergeCount != 1 || len(m.HistoryRevertCommits) != 1 {
							m.HistoryRevertCommits, m.HistoryRevertTarget, m.HistoryRevertConfirm = nil, "", false
							m.Status = "merge commits must be reverted individually with a mainline parent"
							return m, nil
						}
						m.HistoryRevertTarget = mergeSHA
						m.HistoryRevertParentMode, m.HistoryRevertParentMax = true, mergeParents
						m.Status = fmt.Sprintf("revert %s mainline parent (1-%d): ", mergeSHA, mergeParents)
						return m, nil
					}
					m.HistoryRevertTarget = strings.Join(m.HistoryRevertCommits, " ")
					m.Status = "type ordered SHAs " + m.HistoryRevertTarget + ": "
				} else {
					m.HistoryRevertCommits = nil
					m.HistoryRevertTarget = m.History.Rows[m.History.Selected].Commit.SHA
					row := m.History.Rows[m.History.Selected]
					if len(row.Commit.Parents) > 1 {
						m.HistoryRevertCommits = []string{m.HistoryRevertTarget}
						m.HistoryRevertParentMode, m.HistoryRevertParentMax = true, len(row.Commit.Parents)
						m.Status = fmt.Sprintf("revert %s mainline parent (1-%d): ", m.HistoryRevertTarget, m.HistoryRevertParentMax)
						return m, nil
					}
					m.Status = "type SHA " + m.HistoryRevertTarget + ": "
				}
				m.HistoryRevertInput, m.HistoryRevertConfirm, m.HistoryRevertInvalid = "", true, false
			} else if m.currentView() == workspace.Status {
				m.beginRestore()
			}
		case "I":
			if m.currentView() == workspace.GitHub {
				return m, m.startGitHubIssue()
			} else if m.currentView() == workspace.Log {
				return m, m.openRebaseWorkspace()
			}
			if m.currentView() == workspace.Branches {
				return m, m.openSelectedBranchRebase()
			}
			if m.Discovery.Root != "" {
				m.Workspace.Navigate(workspace.Gitignore, "Gitignore catalog")
				return m, m.openGitignore()
			}
		case "M":
			if m.currentView() == workspace.Status {
				m.beginSubmoduleActions()
			} else if m.currentView() == workspace.Branches && m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) {
				branch := m.Branches.Entries[m.Branches.Selected]
				if branch.Current {
					m.Status = "cannot merge the current branch into itself"
				} else if branch.OccupiedPath != "" {
					m.Status = "cannot merge a branch checked out in another worktree: " + platform.SafeText(branch.OccupiedPath)
				} else if m.Snapshot.Counts.Staged > 0 || m.Snapshot.Counts.Unstaged > 0 || m.Snapshot.Counts.Untracked > 0 {
					m.Status = "merge requires a clean worktree; stash explicitly first"
				} else {
					m.BranchMergeMode, m.BranchMergeTarget, m.BranchMutationInput = true, branch.Name, ""
					m.Status = "merge " + branch.Name + " strategy [merge/ff-only/no-ff/squash]: "
				}
			} else if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" && len(m.HistoryInspector.Commit.Parents) > 0 {
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
		case "Y":
			if m.currentView() == workspace.Journal {
				retryable := m.retryableJournalOperation()
				if retryable == nil {
					m.Status = "no failed replayable journal operation"
				} else {
					m.JournalRetryID, m.JournalRetryConfirm = retryable.ID, true
					m.Status = "retry " + platform.SafeText(retryable.Name) + "? (y/n)"
				}
			} else if m.currentView() == workspace.Log || m.currentView() == workspace.Tags || m.currentView() == workspace.Branches || m.currentView() == workspace.Reflog || m.currentView() == workspace.Remotes {
				return m, m.assignCompareSelection()
			}
		case "t":
			if m.currentView() == workspace.Log {
				m.State, m.Status = StateOperationPending, "loading tags"
				return m, m.loadHistoryTags()
			}
			return m, m.navigate(workspace.Tags, "Tags")
		case "C":
			if m.currentView() == workspace.Log {
				m.History.ClearBasket()
				m.Status = "commit basket cleared"
			} else if m.currentView() == workspace.Stashes {
				m.StashCreateMode, m.StashCreateMessage, m.StashIncludeUntracked = true, "", true
				m.Status = "stash message: "
			} else if m.currentView() == workspace.Status && (len(m.Snapshot.Conflicts) > 0 || (m.Snapshot.Operation != nil && recoverableOperation(m.Snapshot.Operation.Kind()))) {
				view, label := workspace.Conflict, "Conflicts"
				if m.Snapshot.Operation != nil {
					if routed, ok := recoveryWorkspaceRoute(m.Snapshot.Operation.Kind()); ok {
						view, label = routed, recoveryWorkspaceLabel(m.Snapshot.Operation.Kind())
					}
				}
				if view == workspace.Bisect {
					return m, m.openBisectWorkspace()
				}
				m.Workspace.Navigate(view, label)
				if len(m.Snapshot.Conflicts) > 0 {
					return m, m.loadConflictContent()
				}
				return m, nil
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
			if m.currentView() == workspace.Tags {
				m.startTagCreation(tags.CreateLightweight)
				return m, nil
			}
			if m.currentView() == workspace.Branches {
				m.BranchCreateMode, m.BranchMutationInput = true, ""
				m.Status = "branch name: "
				return m, nil
			}
			if m.currentView() == workspace.GitHub {
				if m.GitHubReplyCommentID == 0 && len(m.GitHub.Comments) > 0 {
					m.GitHubReplyCommentID = m.GitHub.Comments[m.GitHub.SelectedComment].ID
				}
				return m, m.startGitHubReview(provider.ReviewEventComment)
			}
			return m, m.beginCommit()
		case "enter":
			if m.currentView() == workspace.Journal {
				if m.selectedJournalEvent() == nil {
					m.Status = "no journal entry selected"
				} else {
					m.JournalDetailMode = !m.JournalDetailMode
					m.Status = map[bool]string{true: "journal details opened", false: "journal details closed"}[m.JournalDetailMode]
				}
				return m, nil
			}
			if m.currentView() == workspace.Compare {
				return m, m.loadComparePatch()
			}
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
				return m, m.startRebase(false)
			}
			if m.currentView() == workspace.Repositories {
				m.State, m.Status = StateOperationPending, "opening repository"
				return m, m.openSelectedRepository()
			}
			if m.currentView() == workspace.Branches {
				if m.Branches.Selected >= 0 && m.Branches.Selected < len(m.Branches.Entries) && m.Branches.Entries[m.Branches.Selected].Remote {
					branch := m.Branches.Entries[m.Branches.Selected]
					m.RemoteBranchAction, m.RemoteBranchTarget, m.RemoteBranchInput = "track", branch, branch.RemoteBranch
					m.Status = "local tracking branch name: " + platform.SafeText(m.RemoteBranchInput)
					return m, nil
				}
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
			if m.currentView() == workspace.Tags {
				rows := m.filteredTags()
				if m.TagsSelected >= 0 && m.TagsSelected < len(rows) {
					m.State, m.Status = StateOperationPending, "loading tag target details"
					return m, m.inspectSelectedTag()
				}
				return m, nil
			}
			if m.currentView() == workspace.Reflog {
				m.State, m.Status = StateOperationPending, "loading recovery point details"
				return m, m.inspectSelectedReflog()
			}
			if m.currentView() == workspace.Status && m.showCommitTreePane() && m.CommitTreeFocused {
				return m, m.inspectStatusCommit(m.StatusCommitSelectedLine)
			}
			if m.currentView() == workspace.Status && m.selectedSubmodulePath() != "" {
				return m, m.openSelectedSubmodule()
			}
			if m.currentView() == workspace.Status && m.StatusTreeMode && m.FileTree.ToggleSelected() {
				m.rebuildStatusFileTree()
				m.Status = "directory toggled"
				return m, nil
			}
			return m, m.openDiff()
		case "j", "down":
			if m.currentView() == workspace.GitHub {
				m.GitHub.SelectRun(1)
				m.Status = "selected GitHub check run " + fmt.Sprint(m.GitHub.SelectedRun+1)
				return m, nil
			}
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
			case workspace.PathHistory:
				m.PathHistory.Move(1)
			case workspace.Blame:
				m.Blame.Move(1)
			case workspace.Branches:
				m.Branches.Move(1)
			case workspace.Stashes:
				m.Stashes.Move(1)
			case workspace.Log:
				m.History.Move(1)
			case workspace.Reflog:
				m.Reflog.Move(1)
			case workspace.Journal:
				m.JournalOffset = min(m.JournalOffset+1, m.journalMaxOffset())
			case workspace.Remotes:
				m.Remotes.Move(1)
			case workspace.Worktrees:
				m.Worktrees.Move(1)
			case workspace.Repositories:
				m.Repositories.Move(1)
			case workspace.Rebase:
				m.Rebase.Move(1)
			case workspace.Conflict:
				m.Conflict.Move(1)
			case workspace.CherryPick:
				m.Conflict.Move(1)
			case workspace.Plugins:
				m.Plugins.Move(1)
			case workspace.Tags:
				m.moveTags(1)
			case workspace.Compare:
				m.Compare.Move(1)
			default:
				m.moveStatusFiles(1)
				command := m.openDiff()
				return m, command
			}
		case "k", "up":
			if m.currentView() == workspace.GitHub {
				m.GitHub.SelectRun(-1)
				m.Status = "selected GitHub check run " + fmt.Sprint(m.GitHub.SelectedRun+1)
				return m, nil
			}
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
			case workspace.PathHistory:
				m.PathHistory.Move(-1)
			case workspace.Blame:
				m.Blame.Move(-1)
			case workspace.Branches:
				m.Branches.Move(-1)
			case workspace.Stashes:
				m.Stashes.Move(-1)
			case workspace.Log:
				m.History.Move(-1)
			case workspace.Reflog:
				m.Reflog.Move(-1)
			case workspace.Journal:
				m.JournalOffset = max(0, m.JournalOffset-1)
			case workspace.Remotes:
				m.Remotes.Move(-1)
			case workspace.Worktrees:
				m.Worktrees.Move(-1)
			case workspace.Repositories:
				m.Repositories.Move(-1)
			case workspace.Rebase:
				m.Rebase.Move(-1)
			case workspace.Conflict:
				m.Conflict.Move(-1)
			case workspace.CherryPick:
				m.Conflict.Move(-1)
			case workspace.Plugins:
				m.Plugins.Move(-1)
			case workspace.Tags:
				m.moveTags(-1)
			case workspace.Compare:
				m.Compare.Move(-1)
			default:
				m.moveStatusFiles(-1)
				command := m.openDiff()
				return m, command
			}
		case "pgup":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.scrollContextPane(-m.statusLayout().CommitTree.Height)
				return m, nil
			}
			if m.currentView() == workspace.Status && m.DiffPath == "" {
				m.moveStatusFiles(-m.statusRowCount())
			} else {
				m.scrollDiff(-m.statusRowCount())
			}
		case "pgdown":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.scrollContextPane(m.statusLayout().CommitTree.Height)
				return m, nil
			}
			if m.currentView() == workspace.Status && m.DiffPath == "" {
				m.moveStatusFiles(m.statusRowCount())
			} else {
				m.scrollDiff(m.statusRowCount())
			}
		case "home":
			if m.currentView() == workspace.Status && m.contextPaneFocused() {
				m.CommitTreeOffset = 0
				m.UnpushedOffset = 0
				return m, nil
			}
			if m.currentView() == workspace.Status && m.DiffPath == "" {
				if m.StatusTreeMode {
					m.FileTree.Home(m.statusRowCount())
					m.moveStatusFiles(0)
				} else {
					m.Files.Selected, m.Files.Offset = 0, 0
				}
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
			if m.currentView() == workspace.Status && m.DiffPath == "" {
				if m.StatusTreeMode {
					m.FileTree.End(m.statusRowCount())
					m.moveStatusFiles(0)
				} else if len(m.Files.Visible) > 0 {
					m.Files.Selected, m.Files.Offset = len(m.Files.Visible)-1, max(0, len(m.Files.Visible)-m.statusRowCount())
				}
				return m, nil
			}
		case "space":
			if m.currentView() == workspace.Log {
				if err := m.History.ToggleBasket(); err != nil {
					m.Status = "commit basket: " + err.Error()
				} else {
					m.Status = fmt.Sprintf("commit basket: %d selected", m.History.Basket.Count())
				}
				return m, nil
			}
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
			if m.currentView() == workspace.Reflog {
				m.State, m.Status = StateOperationPending, "comparing recovery point to HEAD"
				return m, m.compareSelectedReflog()
			}
			if m.currentView() == workspace.Tags {
				m.TagCompareLoading = true
				m.State, m.Status = StateOperationPending, "comparing tag to HEAD"
				return m, m.compareSelectedTag()
			}
			return m, m.openDiff()
		case "ctrl+e":
			if m.currentView() == workspace.Status {
				return m, m.openSelectedExternalTool("editor", m.EditorTool)
			}
			if m.currentView() == workspace.Compare {
				return m, m.prepareCompareExternalTool("editor", m.EditorTool)
			}
		case "ctrl+o":
			if m.currentView() == workspace.Status {
				return m, m.openSelectedExternalTool("opener", m.OpenerTool)
			}
			if m.currentView() == workspace.Compare {
				return m, m.prepareCompareExternalTool("opener", m.OpenerTool)
			}
		case "ctrl+t":
			if m.currentView() == workspace.Status {
				if m.StatusTreeMode {
					if _, ok := m.selectedStatusTreeEntry(); !ok {
						m.Status = "difftool: select a file row"
						return m, nil
					}
				}
				path := string(m.Files.SelectedPath())
				if m.StatusTreeMode {
					entry, _ := m.selectedStatusTreeEntry()
					path = string(entry.Path)
				}
				if m.Difftool.Executable == "" {
					command, err := git.NewRunner(m.Discovery.Root).ExternalDiffToolCommand([]byte("HEAD"), nil, []byte(path))
					if err != nil {
						m.Status = "difftool: " + err.Error()
						return m, nil
					}
					return m, m.openExternalProcess("difftool", command)
				}
				return m, m.openExternalTool("difftool", m.Difftool, map[string]string{"left": "HEAD", "right": "WORKTREE", "path": path, "repo": m.Discovery.Root})
			}
			if m.currentView() == workspace.Compare {
				if m.Compare.Selected < 0 || m.Compare.Selected >= len(m.Compare.Result.Changes) {
					m.Status = "difftool: select a changed path first"
					return m, nil
				}
				change := m.Compare.Result.Changes[m.Compare.Selected]
				path := change.NewPath
				if path == "" {
					path = change.OldPath
				}
				command, err := git.NewRunner(m.Discovery.Root).ExternalDiffToolCommand([]byte(m.Compare.Result.Left.SHA), []byte(m.Compare.Result.Right.SHA), []byte(path))
				if err != nil {
					m.Status = "difftool: " + err.Error()
					return m, nil
				}
				return m, m.openExternalProcess("difftool", command)
			}
		case "H":
			if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				m.beginHistoricalHunks()
			} else if m.DiffText != "" {
				m.beginHunks()
			}
		case "O":
			if m.currentView() == workspace.GitHub {
				index := m.GitHub.SelectedIssue
				if index < 0 || index >= len(m.GitHub.Issues) || m.GitHub.Issues[index].URL == "" {
					m.Status = "no GitHub issue URL available"
					return m, nil
				}
				issue := m.GitHub.Issues[index]
				command, err := platform.OpenURLCommand(issue.URL)
				if err != nil {
					m.Status = err.Error()
					return m, nil
				}
				m.Status = "opening GitHub issue #" + fmt.Sprint(issue.Number)
				return m, tea.ExecProcess(command, nil)
			} else if m.currentView() == workspace.Status {
				m.StatusTreeMode = !m.StatusTreeMode
				m.rebuildStatusFileTree()
				if m.StatusTreeMode {
					m.Status = "tree status mode"
				} else {
					m.Status = "flat status mode"
				}
			}
		case "[":
			if m.currentView() == workspace.GitHub && len(m.GitHub.Comments) > 0 {
				m.GitHub.SelectComment(-1)
				m.GitHubReplyCommentID = m.GitHub.Comments[m.GitHub.SelectedComment].ID
				m.Status = "selected GitHub comment reply target #" + fmt.Sprint(m.GitHubReplyCommentID)
			} else if m.currentView() == workspace.Status && m.StatusTreeMode {
				m.FileTree.CollapseAll()
				m.Status = "all directories collapsed"
			}
		case "h":
			path := ""
			if m.currentView() == workspace.Status {
				path = string(m.Files.SelectedPath())
			} else if m.currentView() == workspace.Log && m.HistoryInspector.Commit.SHA != "" {
				path = m.HistoryInspectorPath
				if path == "" {
					path = string(m.Files.SelectedPath())
				}
			}
			if path != "" {
				return m, m.openPathHistory(path, false)
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
		if m.currentView() == workspace.Journal {
			switch v.Button {
			case tea.MouseWheelUp:
				m.JournalOffset = max(0, m.JournalOffset-3)
			case tea.MouseWheelDown:
				m.JournalOffset = min(m.journalMaxOffset(), m.JournalOffset+3)
			}
			return m, nil
		}
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
			if m.currentView() == workspace.Bisect {
				row := v.Y - 4
				switch {
				case row == 4:
					return m, m.updateBisectKey("i")
				case row == 5 && v.X < 20:
					return m, m.updateBisectKey("g")
				case row == 5 && v.X < 40:
					return m, m.updateBisectKey("b")
				case row == 5 && v.X < 55:
					return m, m.updateBisectKey("s")
				case row == 5:
					return m, m.updateBisectKey("x")
				}
				return m, nil
			}
			if m.currentView() == workspace.Journal {
				row := v.Y - 4
				if row >= 0 {
					m.JournalOffset = min(row, m.journalMaxOffset())
				}
				return m, nil
			}
			if m.currentView() == workspace.PathHistory {
				// The view has a title, mode row, then two terminal rows per entry.
				row := (v.Y - 4) / 2
				if v.Y >= 4 && row >= 0 && row < len(m.PathHistory.Entries) {
					m.PathHistory.Selected = row
				}
				return m, nil
			}
			if m.currentView() == workspace.Blame {
				row := v.Y - 3 // feature title, blank line, and blame header
				if v.Y >= 3 && row >= 0 && row < len(m.Blame.Lines) {
					m.Blame.Selected = row
					return m, m.inspectSelectedBlame()
				}
				return m, nil
			}
			if m.recoveryWorkspace() {
				action, index := m.Conflict.Click(v.X, v.Y-2, m.Width, m.Height-2)
				if action == conflictview.MouseSelectConflict {
					m.Conflict.Selected = index
					return m, nil
				}
				mouseKeys := map[conflictview.MouseAction]string{
					conflictview.MouseChooseOurs:        "o",
					conflictview.MouseChooseTheirs:      "t",
					conflictview.MouseChooseBoth:        "b",
					conflictview.MouseMarkResolved:      "m",
					conflictview.MouseRestoreUnresolved: "u",
					conflictview.MouseStatus:            "1",
				}
				if key, ok := mouseKeys[action]; ok {
					return m, m.updateConflictKey(key)
				}
				return m, nil
			}
			if m.currentView() == workspace.Rebase {
				action, index := m.Rebase.Click(v.X, v.Y-2, m.Width, m.Height-2)
				switch action {
				case rebaseview.MouseChooseBase:
					if index >= 0 {
						if err := m.Rebase.SetBase(index); err != nil {
							m.Status = err.Error()
						} else {
							m.Rebase.BaseMode = false
						}
					} else {
						m.Rebase.BaseMode = true
					}
					return m, nil
				case rebaseview.MouseSelectPlan:
					m.Rebase.Selected = index
					return m, nil
				case rebaseview.MouseStart:
					m.State, m.Status = StateOperationPending, "starting interactive rebase"
					return m, m.startRebase(false)
				case rebaseview.MouseCancel:
					m.Workspace.Back()
					m.Status = "rebase cancelled"
					return m, nil
				}
			}
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
			visibleHeight := max(1, files.Height-1-m.statusFileHeaderRows(files.Width))
			rowOffset, rowHeights, rowCount := m.Files.Offset, m.statusFileRowHeights(files.Width, visibleHeight), len(m.Files.Visible)
			if m.StatusTreeMode {
				rowOffset, rowHeights, rowCount = m.FileTree.Offset, m.statusTreeRowHeights(files.Width, visibleHeight), len(m.FileTree.Rows)
			}
			hit := uimouse.HitMap{Files: files, RowTop: files.Y + 1 + m.statusFileHeaderRows(files.Width), RowHeight: 1, Offset: rowOffset, RowHeights: rowHeights, StageX: files.X + 1, StageWidth: 3, RowCount: rowCount}
			action, row, ok := hit.Hit(v.X, v.Y, 0)
			if ok {
				if m.StatusTreeMode {
					m.FileTree.Selected = row
					if row >= 0 && row < len(m.FileTree.Rows) && m.FileTree.Rows[row].Directory {
						m.FileTree.ToggleSelected()
						m.Status = "directory toggled"
						return m, nil
					}
					if index, selected := m.FileTree.SelectedEntryIndex(); selected {
						for visible, entryIndex := range m.Files.Visible {
							if entryIndex == index {
								m.Files.Selected = visible
								break
							}
						}
					}
				} else {
					m.Files.Selected = row
				}
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
		m.Gitignore.SetSize(v.Width, v.Height)
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
		preview := m.previewSelectedStatusDiff()
		return m, tea.Batch(waitForRefresh(v.Coordinator), m.loadSubmodules(v.Result.Snapshot.Generation), m.refreshStatusContextIfNeeded(), preview)
	case refreshRequestedMsg:
		if v.Coordinator != m.RefreshCoordinator {
			return m, nil
		}
		m.State = StateRefreshing
		v.Coordinator.Request(v.Context)
	case SnapshotMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.applySnapshot(v.Snapshot)
		m.State = StateReady
		preview := m.previewSelectedStatusDiff()
		return m, tea.Batch(m.loadSubmodules(v.Snapshot.Generation), m.refreshStatusContextIfNeeded(), preview)
	case SubmodulesReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.SubmodulesLoading, m.SubmodulesErr = false, v.Err
		if v.Err == nil {
			m.Submodules = v.Snapshot
		}
		return m, nil
	case SubmoduleFinishedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.SubmoduleAction, m.SubmodulePath, m.SubmoduleInput, m.SubmoduleURL = "", "", "", ""
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, "submodule "+v.Outcome.Action+": "+v.Outcome.Err.Error()
			return m, nil
		}
		m.State, m.Status = StateReady, "submodule "+v.Outcome.Action+" complete"
		return m, m.refresh()
	case BulkSubmoduleFinishedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.BulkSubmoduleCancel = nil
		m.SubmoduleAction = ""
		outcome := v.Outcome
		m.BulkSubmoduleOutcome = &outcome
		succeeded, failed, skipped := 0, 0, 0
		for _, item := range outcome.Items {
			switch item.State {
			case submodules.ItemSucceeded:
				succeeded++
			case submodules.ItemFailed:
				failed++
			case submodules.ItemSkipped:
				skipped++
			}
		}
		m.State = StateReady
		if outcome.Cancelled {
			m.Status = fmt.Sprintf("bulk submodule %s cancelled: %d succeeded, %d failed, %d skipped", outcome.Action, succeeded, failed, skipped)
		} else {
			m.Status = fmt.Sprintf("bulk submodule %s complete: %d succeeded, %d failed, %d skipped", outcome.Action, succeeded, failed, skipped)
		}
		return m, m.refresh()
	case SubmoduleOpenedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		if len(m.repositoryParents) >= maxRepositoryParentDepth {
			m.State, m.Status = StateError, "submodule navigation depth limit reached"
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, "open submodule: "+v.Err.Error()
			return m, nil
		}
		m.repositoryParents = append(m.repositoryParents, repositoryParent{Discovery: m.Discovery, Label: m.Discovery.Root})
		if err := m.setRepository(v.Discovery); err != nil {
			m.State, m.Status = StateError, "open submodule: "+err.Error()
			return m, nil
		}
		m.Workspace.Navigate(workspace.Status, "Submodule: "+v.Path)
		m.State, m.Status = StateReady, "opened submodule "+platform.SafeText(v.Path)+"; Esc returns to parent"
		return m, tea.Batch(m.refresh(), waitForRefresh(m.RefreshCoordinator), m.startWatcher())
	case GitignoreReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.GitignoreReadOnly = v.ReadOnly
		if v.ReadOnly {
			m.GitignoreCreateConfirm = false
			m.GitignoreCreatePlan = domain.MutationPlan{}
			m.Gitignore.SetPreview("")
			m.State = StateReady
			m.Status = "gitignore is read-only: " + v.Err.Error()
			return m, nil
		}
		if v.Model.RepositoryID != "" {
			m.Gitignore = v.Model
			m.GitignoreMissing = v.Missing
		}
		if v.Err != nil {
			m.Status = "gitignore catalog: " + v.Err.Error()
			return m, nil
		}
		m.State, m.Status = StateReady, "gitignore catalog ready"
		return m, nil
	case GitignoreCatalogReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateReady, "gitignore catalog refresh failed; retaining current catalog: "+v.Err.Error()
			return m, nil
		}
		m.GitignoreCatalog, m.GitignoreCatalogSource = v.Source.Catalog, v.Source.Kind
		m.State, m.Status = StateReady, "gitignore catalog refreshed from "+string(v.Source.Kind)
		return m, m.openGitignore()
	case GitignoreCreatePreviewMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.Status = "gitignore create preview: " + v.Err.Error()
			return m, nil
		}
		m.GitignoreCreatePlan, m.GitignoreCreateConfirm = v.Plan, true
		m.Gitignore.SetPreview(v.Text)
		m.Status = "review gitignore mutation preview; confirm with y or cancel with n"
		return m, nil
	case GitignoreMutationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			if errors.Is(v.Err, domain.ErrConcurrentModification) {
				m.State, m.Status = StateReady, "gitignore changed while preview was open; reloaded existing-file flow"
				return m, m.openGitignore()
			}
			m.State, m.Status = StateError, "gitignore "+v.Action+": "+v.Err.Error()
			return m, nil
		}
		m.State, m.Status = StateReady, "gitignore "+v.Action+" complete"
		if m.GitignoreReturnToStatus {
			m.GitignoreReturnToStatus = false
			return m, m.refresh()
		}
		return m, tea.Batch(m.openGitignore(), m.refresh())
	case ConflictContentReadyMsg:
		if v.Generation != m.repositoryGeneration || v.Request != m.ConflictContentRequest {
			return m, nil
		}
		m.ConflictContentLoading = false
		if v.Err != nil {
			m.Status = "conflict detail: " + v.Err.Error()
			return m, nil
		}
		selected, ok := m.Conflict.SelectedConflict()
		if !ok || string(selected.Path) != string(v.Content.Path) {
			return m, nil
		}
		m.Conflict.SetDetail(conflictview.Detail{Path: append([]byte(nil), v.Content.Path...), Ours: conflictViewContent(v.Content.Ours), Theirs: conflictViewContent(v.Content.Theirs), Result: conflictViewContent(v.Content.Result)})
		return m, nil
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
	case AutoFetchTickMsg:
		if !m.AutoFetchEnabled {
			return m, nil
		}
		return m, tea.Batch(m.runAutoFetch(), m.autoFetchTick())
	case AutoFetchFinishedMsg:
		m.AutoFetchRunning = false
		if m.AutoFetchResults == nil {
			m.AutoFetchResults = make(map[string]remoteintel.Result)
		}
		for _, result := range v.Results {
			m.AutoFetchResults[result.Repository] = result
			matched := false
			for index := range m.RepositoryRegistry {
				if m.RepositoryRegistry[index].Path != result.Repository {
					continue
				}
				matched = true
				m.RepositoryRegistry[index].LastAutoFetch = result.Finished
				m.RepositoryRegistry[index].LastAutoFetchStatus = result.Status
				m.RepositoryRegistry[index].LastAutoFetchError = result.FailureClass
				if !result.Started.IsZero() && !result.Finished.IsZero() {
					m.RepositoryRegistry[index].LastAutoFetchMillis = result.Finished.Sub(result.Started).Milliseconds()
				}
			}
			if !matched && result.Repository != "" {
				m.RepositoryRegistry = append(m.RepositoryRegistry, registry.Repository{Path: result.Repository, Name: filepath.Base(result.Repository), LastAutoFetch: result.Finished, LastAutoFetchStatus: result.Status, LastAutoFetchError: result.FailureClass})
			}
		}
		if len(m.Repositories.AllRows) > 0 {
			m.Repositories.SetRows(m.applyAutoFetchResults(m.Repositories.AllRows))
		}
		fetched, failed, skipped := 0, 0, 0
		for _, result := range v.Results {
			switch result.Status {
			case "fetched":
				fetched++
			case "failed":
				failed++
			case "skipped-active":
				skipped++
			}
		}
		if fetched > 0 {
			m.Status = fmt.Sprintf("auto-fetch complete: %d fetched, %d failed, %d skipped", fetched, failed, skipped)
		} else if failed > 0 {
			m.Status = fmt.Sprintf("auto-fetch: %d failed, %d skipped", failed, skipped)
		}
		return m, m.loadRepositories()
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
		if m.currentView() == workspace.Gitignore && m.GitignoreCreateConfirm && gitignoreWatchHint(v.Event.Path, m.Discovery.Root) {
			m.GitignoreCreateConfirm = false
			m.GitignoreCreatePlan = domain.MutationPlan{}
			m.Gitignore.SetPreview("")
			m.State, m.Status = StateReady, "file changed externally; refresh preview"
			return m, tea.Batch(m.refresh(), m.openGitignore(), waitForWatcher(v.Manager))
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
			m.recordActivityWithOperation(history.OperationFailure, "", v.Name+": "+v.Err.Error(), v.Operation)
		} else {
			m.State = StateReady
			m.Status = v.Name + " complete"
			m.notify(notifications.JobComplete, notifications.Success, m.Status, "", false)
			m.recordActivityWithOperation(history.OperationSuccess, "", m.Status, v.Operation)
		}
		return m, m.refresh()
	case ExternalToolFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.State = StateReady
		if v.Err != nil {
			m.Status = v.Name + ": " + v.Err.Error()
		} else {
			m.Status = v.Name + " exited; refreshing authoritative status"
		}
		return m, m.refresh()
	case CustomCommandFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.State = StateReady
		if v.Err != nil {
			m.Status = "custom command " + platform.SafeText(v.Name) + ": " + platform.SafeText(platform.RedactSecrets(v.Err.Error()))
		} else {
			output := strings.TrimSpace(string(append(append([]byte(nil), v.Output.Stdout...), v.Output.Stderr...)))
			if output != "" {
				m.Status = "custom command " + platform.SafeText(v.Name) + ": " + platform.SafeText(platform.RedactSecrets(output))
			} else {
				m.Status = "custom command " + platform.SafeText(v.Name) + " complete"
			}
		}
		if v.Refresh {
			return m, m.refresh()
		}
		return m, nil
	case HistoricalToolReadyMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Name+": "+v.Err.Error()
			return m, nil
		}
		materialized, err := platform.MaterializeFile(filepath.Base(v.Path), v.Content, int(m.DiffMaxBytes))
		if err != nil {
			m.State, m.Status = StateError, v.Name+": "+err.Error()
			return m, nil
		}
		command, err := v.Tool.CommandValues(map[string]string{"path": materialized.Path, "repo": m.Discovery.Root, "left": m.Compare.Result.Left.SHA, "right": m.Compare.Result.Right.SHA}, m.Discovery.Root, nil)
		if err != nil {
			materialized.Cleanup()
			m.State, m.Status = StateError, v.Name+": "+err.Error()
			return m, nil
		}
		return m, m.openExternalProcessWithCleanup(v.Name, command, materialized.Cleanup)
	case HistoricalPatchAppliedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, "historical patch: "+v.Err.Error()
			return m, nil
		}
		return m, m.beginHistoricalAmend()
	case RebaseFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Outcome.Paused {
			if m.HistoricalRebaseAction == rebase.Reword || m.HistoricalRebaseAction == rebase.Edit {
				if len(m.HistoricalPatch) > 0 {
					return m, m.applyHistoricalPatch()
				}
				cmd := m.beginHistoricalAmend()
				m.Status = "rebase paused; amend the selected historical commit (ctrl+x aborts)"
				return m, cmd
			}
			m.State = StateReady
			m.Status = "interactive rebase paused: " + v.Outcome.State.Phase().String()
			if v.Err != nil {
				m.Status += " (Git reported: " + v.Err.Error() + ")"
			}
			return m, m.refresh()
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "", "interactive rebase: "+v.Err.Error(), v.Operation)
		} else {
			m.State, m.Status = StateReady, "interactive rebase complete"
			m.recordActivityWithOperation(history.OperationSuccess, "", m.Status, v.Operation)
		}
		return m, m.refresh()
	case CherryPickFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.CherryPickCommits = nil
		if v.Outcome.Snapshot != nil {
			m.applySnapshot(*v.Outcome.Snapshot)
		}
		if v.Outcome.Paused {
			m.State = StateReady
			m.Status = "cherry-pick paused for conflict recovery"
			m.Workspace.Navigate(workspace.CherryPick, "Cherry-pick recovery")
			m.recordActivityWithOperation(history.OperationFailure, "cherry-pick", m.Status, v.Operation)
			if len(m.Snapshot.Conflicts) > 0 {
				return m, m.loadConflictContent()
			}
			return m, m.refresh()
		}
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, v.Outcome.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "cherry-pick", v.Outcome.Err.Error(), v.Operation)
		} else {
			m.State, m.Status = StateReady, "cherry-pick completed"
			m.recordActivityWithOperation(history.OperationSuccess, "cherry-pick", m.Status, v.Operation)
		}
		return m, tea.Batch(m.refresh(), m.loadHistory())
	case UndoFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.UndoRecord = nil
		if v.Outcome.Snapshot.Root != "" {
			m.applySnapshot(v.Outcome.Snapshot)
		}
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, "undo commit: "+v.Outcome.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "undo commit", m.Status, v.Operation)
		} else {
			m.State, m.Status = StateReady, "undo commit complete"
			m.recordActivityWithOperation(history.OperationSuccess, "undo commit", m.Status, v.Operation)
		}
		return m, m.refresh()
	case RedoFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.RedoRecord = nil
		if v.Outcome.Snapshot.Root != "" {
			m.applySnapshot(v.Outcome.Snapshot)
		}
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, "redo commit: "+v.Outcome.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "redo commit", m.Status, v.Operation)
		} else {
			m.State, m.Status = StateReady, "redo commit complete"
			m.recordActivityWithOperation(history.OperationSuccess, "redo commit", m.Status, v.Operation)
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
	case CompareReadyMsg:
		if v.Generation != m.CompareGeneration || v.Request != m.CompareRequest {
			return m, nil
		}
		m.CompareCancel, m.CompareLoading = nil, false
		m.CompareErr = v.Err
		if v.Err != nil {
			m.State, m.Status = StateError, "comparison: "+v.Err.Error()
			return m, nil
		}
		m.Compare.SetResult(v.Result)
		m.State, m.Status = StateReady, "comparison loaded"
	case PathHistoryReadyMsg:
		if v.Generation != m.PathHistoryGeneration || v.Request != m.PathHistoryRequest {
			return m, nil
		}
		m.PathHistoryCancel, m.PathHistoryLoading = nil, false
		m.PathHistoryErr = v.Err
		if v.Err != nil {
			m.State, m.Status = StateError, "path history: "+v.Err.Error()
			return m, nil
		}
		m.PathHistory.Follow = v.Follow
		if v.Skip == 0 {
			m.PathHistory.SetPage(v.Path, v.Entries, v.HasMore)
		} else {
			m.PathHistory.AppendPage(v.Entries, v.HasMore)
		}
		m.State, m.Status = StateReady, "path history loaded"
	case BlameReadyMsg:
		if v.Generation != m.BlameGeneration || v.Request != m.BlameRequest {
			return m, nil
		}
		m.BlameCancel, m.BlameLoading = nil, false
		m.BlameErr = v.Err
		if v.Err != nil {
			m.State, m.Status = StateError, "blame: "+v.Err.Error()
			return m, nil
		}
		if v.Start == m.Blame.Start || len(m.Blame.Lines) == 0 {
			m.Blame.SetPage(v.Path, v.Start, v.Lines, v.HasMore)
		} else {
			m.Blame.AppendPage(v.Start, v.Lines, v.HasMore)
		}
		m.State, m.Status = StateReady, "blame loaded"
	case ComparePatchReadyMsg:
		if v.Generation != m.CompareGeneration || v.Request != m.ComparePatchRequest {
			return m, nil
		}
		m.ComparePatchCancel, m.ComparePatchLoading = nil, false
		if v.Err != nil {
			m.CompareErr, m.State, m.Status = v.Err, StateError, "comparison patch: "+v.Err.Error()
			return m, nil
		}
		m.Compare.SetPatch(v.Path, v.Text, v.Truncated)
		m.State, m.Status = StateReady, "comparison patch loaded"
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
		m.rebuildStatusFileTree()
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
	case ReflogReadyMsg:
		if v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.ReflogLoading = false
		if v.Err != nil {
			m.State, m.Status = StateError, "reflog: "+v.Err.Error()
		} else if v.Skip == 0 {
			m.Reflog.SetPage(v.Entries, v.HasMore)
			m.ReflogSkip = len(v.Entries)
			m.State, m.Status = StateReady, ""
		} else {
			m.Reflog.AppendPage(v.Entries, v.HasMore)
			m.ReflogSkip = v.Skip + len(v.Entries)
			m.State = StateReady
		}
	case BisectReadyMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.BisectLoading = false
		if v.Err != nil {
			m.State, m.Status = StateError, "bisect: "+v.Err.Error()
		} else {
			m.Bisect, m.State, m.Status = v.State, StateReady, ""
		}
	case BisectOutputMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Open {
			m.BisectRunOutput = appendBisectDisplayOutput(m.BisectRunOutput, v.Chunk)
			return m, waitBisectOutput(v.Repository, v.Output)
		}
	case BisectFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Outcome.Snapshot.Root != "" {
			m.applySnapshot(v.Outcome.Snapshot)
		}
		m.Bisect = v.Outcome.State
		if v.Action == "run" {
			m.BisectRunMode, m.BisectRunInput, m.BisectRunConfirm = "", "", false
			m.BisectRunArgs = nil
			m.BisectRunOutput = platform.SafeText(string(append(append([]byte(nil), v.Outcome.Result.Stdout...), v.Outcome.Result.Stderr...)))
		}
		if v.Action == "start" {
			m.BisectStartMode, m.BisectStartInput, m.BisectStartConfirm = "", "", false
		}
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, "bisect "+v.Action+": "+v.Outcome.Err.Error()
		} else {
			m.State, m.Status = StateReady, "bisect "+v.Action+" complete"
			if v.Action == "reset" {
				m.Workspace.Back()
			}
		}
		return m, m.refresh()
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
			m.recordActivityWithOperation(history.OperationFailure, "", "commit: "+v.Err.Error(), v.Operation)
		} else if m.HistoricalRebaseAction != "" {
			m.CommitAmendConfirm, m.CommitAuthorMode = false, false
			m.State, m.Status = StateOperationPending, "continuing historical rebase"
			return m, m.continueHistoricalRebase()
		} else {
			m.CommitAmendConfirm, m.CommitAuthorMode = false, false
			m.State, m.Status = StateReady, "commit "+v.SHA
			m.Workspace.Back()
			m.recordActivityWithOperation(history.OperationSuccess, "", m.Status, v.Operation)
		}
		return m, m.refresh()
	case RebaseContinueFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			return m, m.refresh()
		}
		m.HistoricalRebaseAction, m.HistoricalRebaseTarget = "", ""
		m.HistoricalPatch, m.HistoricalPatchPath = nil, ""
		m.State, m.Status = StateReady, "historical rebase continued"
		m.Workspace.Back()
		return m, tea.Batch(m.refresh(), m.loadHistory())
	case RebaseAbortFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.HistoricalRebaseAction, m.HistoricalRebaseTarget = "", ""
			m.HistoricalPatch, m.HistoricalPatchPath = nil, ""
			m.State, m.Status = StateReady, "historical rebase aborted"
			m.Workspace.Back()
		}
		return m, tea.Batch(m.refresh(), m.loadHistory())
	case FixupFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Target, "fixup: "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "fixup commit "+v.SHA+" for "+v.Target
			m.recordActivity(history.OperationSuccess, v.Target, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadHistory())
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
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		if v.Err != nil {
			m.GitHubMergeRefresh = false
			m.GitHub.SetError(v.Repository, v.Branch, v.Err)
			if m.ProviderCI == nil {
				m.ProviderCI = make(map[string]providerCIAttention)
			}
			m.ProviderCI[m.Discovery.Root] = providerCIAttention{State: string(m.GitHub.State), Attention: "provider"}
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			v.Pull.Checks = provider.Checks{Total: v.Checks.Passing + v.Checks.Failing + v.Checks.Pending, Passing: v.Checks.Passing, Failing: v.Checks.Failing, Pending: v.Checks.Pending}
			v.Pull.ReviewState = v.Review.State()
			m.GitHub.SetData(v.Repository, v.Branch, v.Pull, v.Checks)
			m.GitHub.SetPullRequests(v.Pulls)
			m.GitHub.SetIssues(v.Issues)
			m.GitHub.SetReleases(v.Releases)
			m.GitHub.SetProviderFreshness(v.ProviderStale)
			if m.ProviderCI == nil {
				m.ProviderCI = make(map[string]providerCIAttention)
			}
			ciState, attention := "passing", ""
			if v.Checks.Failing > 0 {
				ciState, attention = "failing", "checks"
			} else if v.Checks.Pending > 0 {
				ciState = "pending"
			}
			m.ProviderCI[m.Discovery.Root] = providerCIAttention{State: ciState, Stale: v.ProviderStale, Attention: attention}
			if len(m.Repositories.AllRows) > 0 {
				m.Repositories.SetRows(m.applyProviderCIAttention(m.Repositories.AllRows))
			}
			if v.Detail != nil {
				m.GitHub.SetDetail(*v.Detail)
			}
			m.GitHub.SetComments(v.Comments)
			if m.GitHubMergeRefresh {
				m.GitHubMergeRefresh, m.GitHubMergeConfirm = false, true
				m.State = StateReady
				m.Status = "confirm GitHub " + string(m.GitHubMergeMethod) + " merge of PR #" + fmt.Sprint(v.Pull.Number) + "? (y/n)"
				return m, nil
			}
			m.State, m.Status = StateReady, "GitHub data loaded"
			m.reindexPalette()
		}
	case GitHubPullRequestCreatedMsg:
		m.GitHubCreateConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub PR creation: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, fmt.Sprintf("GitHub PR #%d created", v.Pull.Number)
			return m, m.loadGitHub()
		}
	case GitHubMergeFinishedMsg:
		m.GitHubMergeConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub merge: "+platform.SafeText(v.Err.Error())
		} else if !v.Result.Merged {
			m.State, m.Status = StateError, "GitHub merge was not completed: "+platform.SafeText(v.Result.Message)
		} else {
			m.State = StateReady
			m.GitHubBranchDeleteTarget = m.GitHub.Pull.Head
			if err := provider.ValidateCheckoutRef(m.GitHubBranchDeleteTarget); err == nil {
				m.GitHubBranchDeleteConfirm = true
				m.Status = "GitHub merge completed. delete remote branch " + platform.SafeText(m.GitHubBranchDeleteTarget) + "? (y/n)"
				return m, nil
			}
			m.Status = "GitHub merge completed; local refs unchanged until fetch"
			return m, m.loadGitHub()
		}
	case GitHubBranchDeleteFinishedMsg:
		m.GitHubBranchDeleteConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub branch deletion: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "deleted remote branch "+platform.SafeText(v.Branch)+"; local refs unchanged"
			m.GitHubBranchDeleteTarget = ""
		}
		return m, m.loadGitHub()
	case GitHubReviewFinishedMsg:
		m.GitHubReviewConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub review: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "GitHub review submitted; refreshing review state"
			return m, m.loadGitHub()
		}
	case GitHubReviewCommentFinishedMsg:
		m.GitHubReviewConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub review comment: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "GitHub reply submitted; refreshing review comments"
			m.GitHubReplyCommentID = 0
			return m, m.loadGitHub()
		}
	case GitHubCheckActionFinishedMsg:
		m.GitHubCheckActionConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub check action: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, "GitHub check "+v.Action+" requested; refreshing checks"
			return m, m.loadGitHub()
		}
	case GitHubIssueCreatedMsg:
		m.GitHubIssueConfirm = false
		if v.Err != nil {
			m.State, m.Status = StateError, "GitHub issue: "+platform.SafeText(v.Err.Error())
		} else {
			m.State, m.Status = StateReady, fmt.Sprintf("GitHub issue #%d created", v.Issue.Number)
			return m, m.loadGitHub()
		}
	case PluginsReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Plugins.SetEntries(v.Entries)
			m.publishPluginNotifications(v.Entries)
			m.State, m.Status = StateReady, "plugins loaded"
			m.reindexPalette()
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
	case MergeFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.BranchMergeMode, m.BranchMergeTarget, m.BranchMutationInput = false, "", ""
		if v.Outcome.Snapshot != nil {
			m.applySnapshot(*v.Outcome.Snapshot)
		}
		if v.Outcome.Paused {
			m.State, m.Status = StateReady, "merge paused for conflict recovery"
			m.Workspace.Navigate(workspace.Conflict, "Merge recovery")
			if len(m.Snapshot.Conflicts) > 0 {
				return m, m.loadConflictContent()
			}
			return m, nil
		}
		if v.Outcome.Err != nil {
			m.State, m.Status = StateError, v.Outcome.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "merge", v.Outcome.Err.Error(), v.Operation)
			return m, m.refresh()
		}
		m.State, m.Status = StateReady, "merge completed"
		m.recordActivityWithOperation(history.OperationSuccess, "merge", m.Status, v.Operation)
		return m, tea.Batch(m.refresh(), m.loadBranches(), m.loadHistory())
	case HistoryReadyMsg:
		m.HistoryCancel = nil
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			if v.Skip == 0 {
				m.HistoryCommits = append([]history.Commit(nil), v.Commits...)
				m.HistoryRangeAnchorSet = false
			} else {
				m.HistoryCommits = append(m.HistoryCommits, v.Commits...)
			}
			if v.Skip == 0 {
				basket := m.History.Basket
				m.History = historyview.New(m.HistoryCommits)
				m.History.Basket = basket
			} else {
				m.History.AppendCommits(v.Commits)
			}
			m.HistorySkip, m.HistoryHasMore, m.State = v.Skip+len(v.Commits), v.HasMore, StateReady
			m.Repositories.SetRows(m.applyCommitActivity(m.Repositories.AllRows))
		}
	case HistoryInspectorReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.HistoryInspector, m.State, m.Status = v.Inspector, StateReady, ""
		}
	case ReflogCompareReadyMsg:
		if v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.ReflogCompareLoading = false
		if v.Err != nil {
			m.State, m.Status = StateError, "reflog compare: "+v.Err.Error()
		} else {
			m.ReflogCompare, m.State, m.Status = v.Text, StateReady, "recovery point compared to HEAD"
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
	case TagsReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.TagsLoading = false
		if v.Err != nil {
			m.TagsErr, m.State, m.Status = v.Err, StateError, "load tags: "+v.Err.Error()
		} else {
			m.TagSnapshot, m.TagsErr, m.TagsSelected, m.State, m.Status = v.Snapshot, nil, 0, StateReady, ""
		}
	case TagSignatureReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.TagSignatureChecking = ""
		m.setTagSignature(v.Name, v.State)
		if v.Err != nil {
			m.State, m.Status = StateReady, "tag "+platform.SafeText(v.Name)+" signature: "+string(v.State)
		} else {
			m.State, m.Status = StateReady, "tag "+platform.SafeText(v.Name)+" signature verified"
		}
	case TagCheckoutFinishedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.TagCheckoutConfirm, m.TagCheckoutTarget = false, ""
		if v.Err != nil {
			m.State, m.Status = StateError, "checkout tag "+platform.SafeText(v.Name)+": "+v.Err.Error()
			return m, nil
		}
		m.Workspace.Navigate(workspace.Status, "Status")
		m.State, m.Status = StateReady, "checked out tag "+platform.SafeText(v.Name)+" detached"
		return m, m.refresh()
	case TagCompareReadyMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.TagCompareLoading, m.TagCompare = false, v.Text
		if v.Err != nil {
			m.State, m.Status = StateError, "compare tag "+platform.SafeText(v.Name)+": "+v.Err.Error()
		} else {
			m.State, m.Status = StateReady, "tag "+platform.SafeText(v.Name)+" compared to HEAD"
		}
	case TagWorktreeFinishedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.TagWorktreeMode, m.TagWorktreePath = false, ""
		if v.Err != nil {
			m.State, m.Status = StateError, "create tag worktree "+platform.SafeText(v.Name)+": "+v.Err.Error()
			return m, nil
		}
		m.State, m.Status = StateReady, "created worktree "+platform.SafeText(v.Path)+" from tag "+platform.SafeText(v.Name)
		return m, m.refresh()
	case TagMutationFinishedMsg:
		if v.Generation != 0 && v.Generation != m.repositoryGeneration {
			return m, nil
		}
		m.resetTagMutation()
		if v.Err != nil {
			m.State, m.Status = StateError, v.Operation+" tag "+platform.SafeText(v.Name)+": "+v.Err.Error()
			return m, nil
		}
		m.State, m.Status = StateReady, v.Operation+" tag "+platform.SafeText(v.Name)
		m.recordActivity(history.OperationSuccess, v.Name, m.Status)
		return m, tea.Batch(m.refresh(), m.loadTags(), m.loadHistoryTags(), m.loadRemotes())
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
	case RevertFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		if v.Snapshot != nil {
			m.applySnapshot(*v.Snapshot)
		}
		if v.Paused {
			m.State, m.Status = StateReady, "revert paused for conflict recovery"
			if view, ok := recoveryWorkspaceRoute(sequencer.KindRevert); ok {
				m.Workspace.Navigate(view, recoveryWorkspaceLabel(sequencer.KindRevert))
			}
			m.recordActivityWithOperation(history.OperationFailure, "revert", m.Status, v.Operation)
			if len(m.Snapshot.Conflicts) > 0 {
				return m, m.loadConflictContent()
			}
			return m, m.refresh()
		}
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivityWithOperation(history.OperationFailure, "revert", v.Err.Error(), v.Operation)
		} else {
			m.State, m.Status = StateReady, "revert complete"
			m.Workspace.Back()
			m.recordActivityWithOperation(history.OperationSuccess, "revert", m.Status, v.Operation)
		}
		return m, tea.Batch(m.refresh(), m.loadHistory())
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
	case RemoteTrackingReadyMsg:
		if v.Repository != 0 && v.Repository != m.repositoryGeneration {
			return m, nil
		}
		if v.Err != nil {
			m.resetRemoteMutation()
			m.State, m.Status = StateError, "load remote tracking branches: "+v.Err.Error()
			return m, nil
		}
		m.RemoteMutationImpact = append([]remotes.TrackingBranch(nil), v.Branches...)
		switch m.RemoteMutationMode {
		case "rename-loading":
			m.RemoteMutationMode, m.State = "rename", StateReady
			m.RemoteMutationInput = ""
			m.Status = "new name for " + platform.SafeText(v.Remote) + ": "
		case "remove-loading":
			m.RemoteMutationMode, m.RemoteMutationConfirm, m.State = "remove", true, StateReady
			m.Status = remoteImpactStatus("remove "+v.Remote, m.RemoteMutationImpact)
		}
	case RemotePrunePreviewMsg:
		if v.Repository != 0 && v.Repository != m.repositoryGeneration {
			return m, nil
		}
		m.RemoteMutationMode = "prune"
		if v.Err != nil {
			m.resetRemoteMutation()
			m.State, m.Status = StateError, "preview remote prune: "+v.Err.Error()
			return m, nil
		}
		m.RemotePrunePreview, m.RemotePruneConfirm, m.State = v.Text, true, StateReady
		m.Status = "confirm prune " + platform.SafeText(v.Remote) + "; stale refs: " + platform.SafeText(strings.TrimSpace(v.Text)) + " (y/n)"
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
			rows := m.applyProviderCIAttention(m.applyCommitActivity(m.applyAutoFetchResults(v.Rows)))
			if len(m.Repositories.Rows) == 0 {
				m.Repositories = repoview.New(rows)
			} else {
				m.Repositories.SetRows(rows)
			}
			m.State = StateReady
			m.reindexPalette()
		}
	case RepositoryBatchProgressMsg:
		action := "fetch"
		if m.RepositoryBatchAction == multirepo.ActionPull {
			action = "pull " + m.RepositoryBatchStrategy
		}
		m.State = StateOperationPending
		m.Status = fmt.Sprintf("batch %s: %d/%d complete · %s %s", action, v.Completed, v.Total, v.Status, platform.SafeText(v.Path))
		return m, batchProgressCommand(v.Events)
	case RepositoryBatchFinishedMsg:
		if m.RepositoryBatchCancel != nil {
			m.RepositoryBatchCancel()
			m.RepositoryBatchCancel = nil
		}
		m.RepositoryBatchConfirm, m.RepositoryBatchRetry = false, false
		m.RepositoryBatchResults = append([]multirepo.Result(nil), v.Results...)
		succeeded, failed, cancelled, skipped := 0, 0, 0, 0
		for _, result := range v.Results {
			switch result.Status {
			case "succeeded":
				succeeded++
			case "failed":
				failed++
			case "cancelled":
				cancelled++
			case "skipped":
				skipped++
			}
		}
		m.State = StateReady
		action := "fetch"
		if m.RepositoryBatchAction == multirepo.ActionPull {
			action = "pull " + m.RepositoryBatchStrategy
		}
		m.Status = fmt.Sprintf("batch %s complete: %d succeeded, %d failed, %d cancelled, %d skipped", action, succeeded, failed, cancelled, skipped)
		return m, m.loadRepositories()
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
			m.repositoryParents = nil
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
		m.WorktreeAddCommit = ""
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
			m.recordActivity(history.OperationFailure, v.Target, v.Operation+": "+v.Err.Error())
		} else {
			m.State, m.Status = StateReady, v.Operation+" complete"
			m.recordActivity(history.OperationSuccess, v.Target, m.Status)
		}
		return m, tea.Batch(m.refresh(), m.loadWorktrees(), m.loadBranches())
	case operations.ResultMsg:
		if v.Result.Repo != "" && v.Result.Repo != m.Discovery.Root {
			return m, nil
		}
		m.State = StateReady
		if v.Result.State == operations.Succeeded {
			m.Status = v.Result.Name + " retry complete"
			m.recordActivity(history.OperationSuccess, "", m.Status)
		} else {
			m.Status = v.Result.Name + " retry " + v.Result.State.String()
			if v.Result.Err != nil {
				m.Status += ": " + v.Result.Err.Error()
			}
			m.recordActivity(history.OperationFailure, "", m.Status)
		}
		return m, m.refresh()
	case RemoteOperationFinishedMsg:
		if !m.acceptsRepository(v.Repository) {
			return m, nil
		}
		m.resetRemoteMutation()
		m.RemoteSetUpstream, m.RemoteTag, m.RemoteTagDeleteConfirm, m.RemoteTagDeleteMode = false, "", false, false
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
			m.recordActivityWithOperation(history.OperationFailure, v.Remote, v.Operation+": "+v.Err.Error(), v.Journal)
		} else {
			m.notify(notifications.JobComplete, notifications.Success, v.Operation, v.Remote, false)
			m.State, m.Status = StateReady, v.Operation+" complete: "+v.Remote
			m.recordActivityWithOperation(history.OperationSuccess, v.Remote, m.Status, v.Journal)
		}
		return m, tea.Batch(m.refresh(), m.loadRemotes(), m.loadBranches(), m.loadGitHub())
	}
	return m, nil
}

func (m *Model) publishPluginNotifications(entries []plugins.Entry) {
	for _, entry := range entries {
		if !entry.Enabled || !entry.Healthy {
			continue
		}
		for _, contribution := range entry.Contributions {
			if contribution.Kind != "notification" {
				continue
			}
			title := platform.SafeText(contribution.Title)
			message := platform.SafeText(contribution.Description)
			if title == "" {
				continue
			}
			m.notify(notifications.PluginContribution, notifications.Info, title, message, false)
		}
	}
}

func gitignoreWatchHint(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	gitignorePath := filepath.Join(filepath.Clean(root), ".gitignore")
	if cleanPath == gitignorePath {
		return true
	}
	// Polling and reconciliation events identify the repository root rather
	// than the changed file. While a preview is open, treat that broad hint as
	// a possible external edit; the subsequent reload re-establishes Gitignore
	// state from disk.
	return cleanPath == filepath.Clean(root)
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

func (m Model) customCommandFormView() tea.View {
	lines := []string{"gitwatch custom command", ""}
	if m.CustomCommandForm == nil {
		return tea.NewView(strings.Join(lines, "\n"))
	}
	prompt, ok := m.CustomCommandForm.Current()
	if !ok {
		lines = append(lines, "Form complete")
	} else {
		position, total := m.CustomCommandForm.Progress()
		lines = append(lines, fmt.Sprintf("Prompt %d/%d: %s", position, total, platform.SafeText(prompt.Label)))
		switch prompt.Kind {
		case customcmd.PromptText, customcmd.PromptSecret:
			lines = append(lines, "", "> "+platform.SafeText(m.CustomCommandForm.Input()))
		case customcmd.PromptConfirm:
			lines = append(lines, "", "[y] yes  [n] no")
		case customcmd.PromptSelect, customcmd.PromptMultiSelect:
			selected := make(map[string]bool)
			for _, option := range m.CustomCommandForm.SelectedOptions() {
				selected[option] = true
			}
			for index, option := range m.CustomCommandForm.Options() {
				prefix := "  "
				if index == m.CustomCommandForm.Cursor() {
					prefix = "> "
				}
				if selected[option] {
					prefix = "✓ "
				}
				lines = append(lines, prefix+platform.SafeText(option))
			}
		}
	}
	lines = append(lines, "", "[enter] accept  [esc] cancel")
	v := tea.NewView(strings.Join(safeRenderLines(lines), "\n"))
	v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
	return v
}

func (m Model) View() tea.View {
	if m.PaletteMode {
		return m.paletteView()
	}
	if m.CustomCommandForm != nil {
		return m.customCommandFormView()
	}
	if view := m.currentView(); view == workspace.Branches || view == workspace.Stashes || view == workspace.Log || view == workspace.Reflog || view == workspace.Journal || view == workspace.Commit || view == workspace.Remotes || view == workspace.GitHub || view == workspace.Plugins || view == workspace.Hunks || view == workspace.Worktrees || view == workspace.Repositories || view == workspace.Rebase || view == workspace.Conflict || view == workspace.CherryPick || view == workspace.Gitignore || view == workspace.Tags || view == workspace.Compare || view == workspace.PathHistory || view == workspace.Blame {
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
		if m.HistoryRevertParentMode {
			content += "\n\n" + m.Status
		}
		if len(m.HistoryTags) > 0 {
			content += "\n\nTags:\n"
			for _, tag := range m.HistoryTags {
				content += "  " + tag.Name + " (" + tag.OID + ")\n"
			}
		}
	case workspace.Tags:
		title, content = "gitwatch · tags", m.tagsView()
		if m.TagsFilterMode {
			content += "\n\nFilter: " + platform.SafeText(m.TagsFilter)
		}
		if m.HistoryInspector.Commit.SHA != "" {
			content += "\n\n" + inspectorText(m.HistoryInspector)
		}
		if m.TagCompareLoading {
			content += "\n\nComparing tag to HEAD…"
		} else if m.TagCompare != "" {
			content += "\n\nComparison to HEAD:\n" + platform.SafeText(m.TagCompare)
		}
		if m.TagWorktreeMode {
			content += "\n\n" + platform.SafeText(m.Status)
		}
	case workspace.Reflog:
		title, content = "gitwatch · reflog", m.Reflog.View()
		if entry, ok := m.Reflog.SelectedEntry(); ok {
			content += "\n\nSelected recovery point: " + platform.SafeText(entry.SHA)
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
		if m.ReflogCompareLoading {
			content += "\n\n" + platform.SafeText(m.Status)
		} else if m.ReflogCompare != "" {
			content += "\n\nCompare to HEAD:\n" + platform.SafeText(m.ReflogCompare)
		}
		if m.ReflogLoading {
			content += "\n\n" + platform.SafeText(m.Status)
		}
	case workspace.Journal:
		title, content = "gitwatch · operation journal", m.operationJournalView()
	case workspace.Bisect:
		title, content = "gitwatch · bisect", m.bisectWorkspaceView()
		if m.HistoryInspector.Commit.SHA != "" {
			content += "\n\n" + inspectorText(m.HistoryInspector)
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
		if m.RemoteTagDeleteConfirm || m.RemoteTagDeleteMode {
			content += "\n\n" + m.Status
		}
		if m.RemoteMutationConfirm || m.RemotePruneConfirm || m.RemoteMutationMode != "" {
			content += "\n\n" + platform.SafeText(m.Status)
			if m.RemotePrunePreview != "" {
				content += "\n" + platform.SafeText(m.RemotePrunePreview)
			}
		}
	case workspace.GitHub:
		title, content = "gitwatch · GitHub", m.GitHub.View()
		if m.GitHubCreateMode || m.GitHubCreateConfirm {
			content += "\n\n" + platform.SafeText(m.Status)
		}
		if m.GitHubMergeMode || m.GitHubMergeConfirm || m.GitHubMergeRefresh {
			content += "\n\n" + platform.SafeText(m.Status)
		}
		if m.GitHubReviewMode || m.GitHubReviewConfirm {
			content += "\n\n" + platform.SafeText(m.Status)
		}
		if m.GitHubCheckActionConfirm {
			content += "\n\n" + platform.SafeText(m.Status)
		}
		if m.GitHubIssueMode || m.GitHubIssueConfirm {
			content += "\n\n" + platform.SafeText(m.Status)
		}
	case workspace.Plugins:
		title, content = "gitwatch · plugins", m.Plugins.View()
	case workspace.Hunks:
		title, content = "gitwatch · hunk selection", m.Hunks.View()
		if m.HistoricalPatchMode {
			content += "\n\nHistorical edit: select lines to remove, then press Enter to preview the controlled rebase."
		}
		if m.HunkDiscardConfirm {
			content += "\n\n" + m.Status + ": " + m.HunkDiscardInput
		}
	case workspace.Worktrees:
		title, content = "gitwatch · worktrees", m.Worktrees.View()
	case workspace.Repositories:
		title, content = "gitwatch · repositories", m.Repositories.View()
	case workspace.Rebase:
		title, content = "gitwatch · interactive rebase", m.Rebase.View()
		if m.Status != "" {
			content += "\n\nNOTICE: " + platform.SafeText(m.Status)
		}
	case workspace.Conflict:
		title, content = "gitwatch · conflict resolver", m.Conflict.View(m.Width, m.Height-6)
		if m.Status != "" {
			content += "\n\nNOTICE: " + platform.SafeText(m.Status)
		}
	case workspace.CherryPick:
		title, content = "gitwatch · cherry-pick progress", m.Conflict.View(m.Width, m.Height-6)
		if m.Status != "" {
			content += "\n\nNOTICE: " + platform.SafeText(m.Status)
		}
	case workspace.Gitignore:
		title, content = "gitwatch · gitignore catalog", "catalog source: "+string(m.GitignoreCatalogSource)+"\n"+m.Gitignore.View()
	case workspace.Compare:
		title, content = "gitwatch · comparison", m.Compare.View()
		if m.CompareLoading {
			content += "\n\nComparing revisions…"
		}
		if m.CompareErr != nil {
			content += "\n\nComparison error: " + platform.SafeText(m.CompareErr.Error())
		}
	case workspace.PathHistory:
		title, content = "gitwatch · path history", m.PathHistory.View()
		if m.PathHistoryLoading {
			content += "\n\nLoading path history…"
		}
		if m.PathHistoryErr != nil {
			content += "\n\nPath-history error: " + platform.SafeText(m.PathHistoryErr.Error())
		}
	case workspace.Blame:
		title, content = "gitwatch · blame", m.Blame.View()
		if m.BlameLoading {
			content += "\n\nLoading blame…"
		}
		if m.BlameErr != nil {
			content += "\n\nBlame error: " + platform.SafeText(m.BlameErr.Error())
		}
	}
	title += " · watch:" + watchModeName(m.WatchMode)
	lines := []string{title, "", content, "", "──────────────────────────────────────────────────────────────", "[j/k] move  [1] status  [b] branches  [s] stashes  [l] history  [n] remotes  [esc] back  [q] quit"}
	if m.Notifications != nil && m.Notifications.Attention() > 0 {
		lines[len(lines)-1] += fmt.Sprintf("  [!] %d attention  [ctrl+n] dismiss", m.Notifications.Attention())
	}
	if view == workspace.Log {
		lines[len(lines)-1] = "[j/k] move  [space] basket  [v] range  [C] clear basket  [enter] inspect  [/] search  [] more  [t] tags  [g] ref  [M] parent  [f] path  [y] copy SHA  [x] checkout  [B] branch  [R] revert  [P] cherry-pick  [1] status  [esc] back  [q] quit"
	}
	if view == workspace.Tags {
		lines[len(lines)-1] = "[j/k] move  [c] light tag  [A] annotated  [S] signed  [D] delete  [/] filter  [s] sort  [enter] inspect  [d] compare  [V] verify  [x] checkout  [w] worktree  [t] reload  [esc] back  [q] quit"
		if m.TagsFilterMode {
			lines[len(lines)-1] = "tag filter: type text  [enter] apply  [esc] cancel"
		} else if m.TagCreateMode != "" {
			lines[len(lines)-1] = "tag " + m.TagCreateMode + ": type value  [enter] next  [esc] cancel"
		} else if m.TagDeleteMode {
			lines[len(lines)-1] = "type exact tag name  [enter] delete  [esc] cancel"
		} else if m.TagWorktreeMode {
			lines[len(lines)-1] = "tag worktree path: type path  [enter] create  [esc] cancel"
		}
	}
	if view == workspace.Reflog {
		lines[len(lines)-1] = "[j/k] move  [enter] inspect  [B] branch  [x] checkout  [] load more  [1] status  [esc] back  [q] quit"
	}
	if view == workspace.Journal {
		lines[len(lines)-1] = "[j/k] move  [/] filter  [enter] details  [K] cancel running  [Y] retry failed fetch  [u] undo  [R] redo  [J] newest  [1] status  [esc] back  [q] quit"
		if m.JournalFilterMode {
			lines[len(lines)-1] = "journal filter (repo:/path type:merge outcome:success): " + platform.SafeText(m.JournalFilterInput) + "  [enter] apply  [esc] cancel"
		} else if m.JournalDetailMode {
			lines[len(lines)-1] = "journal details open  [enter] close  [j/k] move  [/] filter  [esc] back  [q] quit"
		} else if m.JournalCancelConfirm {
			lines[len(lines)-1] = "cancel operation: [y] yes  [n] no  [esc] cancel"
		} else if m.JournalRetryConfirm {
			lines[len(lines)-1] = "retry operation: [y] yes  [n] no  [esc] cancel"
		} else if m.UndoConfirm || m.RedoConfirm {
			lines[len(lines)-1] = "confirm: [y] yes  [n] no  [esc] cancel"
		}
	}
	if view == workspace.Bisect {
		lines[len(lines)-1] = "[S] start  [A] run  [g] good  [b] bad  [s] skip  [x] reset  [i] inspect  [r] refresh  [1] status  [esc] back  [q] quit"
		if m.BisectResetConfirm || m.BisectStartConfirm || m.BisectStartMode != "" || m.BisectRunMode != "" || m.BisectRunConfirm {
			lines[len(lines)-1] = "bisect prompt: type token  [enter] next  [y/n] confirm  [esc] cancel"
		}
	}
	if view == workspace.Branches {
		lines[len(lines)-1] = "[j/k] move  [/] filter  [s] sort  [enter] checkout/track  [x] detached  [w] worktree  [F] fast-forward  [z] reset  [I] rebase  [M] merge  [c] create  [R] rename  [u/N] upstream  [D/X] delete  [esc] back  [q] quit"
		if m.BranchSearching {
			lines[len(lines)-1] = "filter: " + platform.SafeText(m.Branches.Query) + "  [enter] apply  [esc] cancel"
		} else if m.RemoteBranchAction == "track" {
			lines[len(lines)-1] = "remote branch: edit local name  [enter] track  [esc] cancel"
		} else if m.RemoteBranchConfirm {
			lines[len(lines)-1] = "remote branch confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.BranchRecoveryConfirm {
			lines[len(lines)-1] = "fast-forward confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.BranchResetPrompt {
			lines[len(lines)-1] = "reset: soft <ref> or mixed <ref>  [enter] apply  [esc] cancel"
		}
	}
	if view == workspace.Gitignore {
		lines[len(lines)-1] = "[j/k] move  [space] select  [tab] filter  [type] search  [a/d] preview/apply  [p] preview  [r] refresh  [b] bundled  [esc] back  [q] quit"
	}
	if view == workspace.Remotes {
		lines[len(lines)-1] = "[j/k] move  [Y] compare  [A] add  [R] rename  [L] set-url  [D] remove  [K] prune  [f] fetch  [p] push  [T/X] tag  [esc] back  [q] quit"
	}
	if view == workspace.GitHub {
		lines[len(lines)-1] = "[j/k] select check  [[/]] reply target  [!] rerun failed  [K] cancel run  [O] issue  [L] release  [r] refresh  [A] approve  [R] request changes  [c] comment/reply  [I] create issue  [n] create PR  [m] merge  [x] checkout head  [o] open  [esc] back  [q] quit"
		if m.GitHubCreateMode {
			lines[len(lines)-1] = "PR form: type  [tab/enter] next  [esc] cancel"
		} else if m.GitHubCreateConfirm {
			lines[len(lines)-1] = "PR creation confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.GitHubMergeMode {
			lines[len(lines)-1] = "merge form: [m] merge  [s] squash  [r] rebase  [enter] refresh  [esc] cancel"
		} else if m.GitHubMergeConfirm {
			lines[len(lines)-1] = "merge confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.GitHubBranchDeleteConfirm {
			lines[len(lines)-1] = "delete remote branch confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.GitHubReviewMode {
			lines[len(lines)-1] = "review form: type  [enter] submit  [esc] cancel"
		} else if m.GitHubReviewConfirm {
			lines[len(lines)-1] = "review confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.GitHubCheckActionConfirm {
			lines[len(lines)-1] = "check action confirmation: [y] yes  [n] no  [esc] cancel"
		} else if m.GitHubIssueMode {
			lines[len(lines)-1] = "issue form: type  [tab/enter] next  [esc] cancel"
		} else if m.GitHubIssueConfirm {
			lines[len(lines)-1] = "issue confirmation: [y] yes  [n] no  [esc] cancel"
		}
		if m.RemoteBranchConfirm {
			lines[len(lines)-1] = "GitHub checkout confirmation: [y] yes  [n] no  [esc] cancel"
		}
	}
	if view == workspace.Plugins {
		lines[len(lines)-1] = "[j/k] move  [r] reload  [esc] back  [q] quit"
	}
	if view == workspace.Hunks {
		lines[len(lines)-1] = "[j/k] move  [n/p] hunk  [N/P] file  [c] context  [space] select  [a/A/i] hunk/all/invert  [s] stage  [d] discard  [esc] back  [q] quit"
		if m.HistoricalPatchMode {
			lines[len(lines)-1] = "[j/k] move  [n/p] hunk  [N/P] file  [space] select  [a/A/i] hunk/all/invert  [enter] preview edit  [esc] cancel  [q] quit"
		}
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
		lines[len(lines)-1] = "[j/k] move  [/] filter  [s] sort  [F] fetch all  [P] pull ff-only  [R] retry failed  [K] cancel  [v] refresh  [enter] open  [esc] back  [q] quit"
		if m.RepositoryBatchConfirm {
			content += "\n\n" + platform.SafeText(m.Status)
		}
		if len(m.RepositoryBatchResults) > 0 {
			failed := make([]string, 0)
			for _, result := range m.RepositoryBatchResults {
				if result.Status == "failed" {
					failed = append(failed, result.Request.Repository.Root)
				}
			}
			if len(failed) > 0 {
				content += "\n\nFailed batch repositories:\n  " + platform.SafeText(strings.Join(failed, "\n  "))
			}
		}
		if m.RepositorySearching {
			lines[len(lines)-1] = "filter: " + platform.SafeText(m.Repositories.Query) + "  [enter] apply  [esc] cancel"
		}
	}
	if view == workspace.Rebase {
		lines[len(lines)-1] = "[j/k] move  [b] choose base  [enter] start  [esc] cancel  [q] quit"
	}
	if view == workspace.Compare {
		lines[len(lines)-1] = "[j/k] move  [enter] file patch  [f] fetch  [o] ff-only pull  [m] merge pull  [e] rebase pull  [p] push preview  [esc] back  [q] quit"
	}
	if view == workspace.PathHistory {
		lines[len(lines)-1] = "[j/k] move  [enter] inspect commit  [Y] compare to HEAD  [f] toggle follow  [] load more  [esc] back  [q] quit"
	}
	if view == workspace.Blame {
		lines[len(lines)-1] = "[j/k] move  [enter] inspect origin commit  [] load more  [esc] back  [q] quit"
	}
	if view == workspace.Conflict {
		lines[len(lines)-1] = "[j/k] conflict  [n/p] hunk  [o/t/b] choose  [m] mark  [u] restore  [c] continue  [x] abort  [1] status  [esc] back  [q] quit"
	}
	if view == workspace.CherryPick {
		lines[len(lines)-1] = "[j/k] commit/conflict  [n/p] hunk  [o/t/b] choose  [m] mark  [u] restore  [c] continue  [x] abort  [1] status  [esc] back  [q] quit"
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
