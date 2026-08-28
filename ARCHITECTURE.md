# Architecture

gitwatch keeps Git and filesystem work outside the Bubble Tea render path. Git command output is authoritative; watcher and timer events only request a refresh.

## Package layout

```text
cmd/gitwatch/            CLI parsing, configuration loading, and program startup
internal/app/            root Bubble Tea model, messages, commands, and view routing
internal/git/            typed Git runner, discovery, porcelain parsing, diffs, and mutations
internal/repo/           immutable repository snapshot/domain models
internal/watch/          fsnotify watcher, debounce, reconciliation, and polling fallback
internal/operations/     bounded asynchronous operation engine
internal/workspace/      workspace navigation and background-job lifecycle

internal/commitmodel/    commit draft and validation domain
internal/branches/       branch parsing, tracking, and guarded mutations
internal/stash/          stash parsing, previews, and guarded mutations
internal/worktrees/      worktree parsing and lifecycle operations
internal/history/        bounded log, graph, inspector, and history actions
internal/hunks/          hunk/line selection and patch construction
internal/patch/          unified-diff parser and workload budgets
internal/remotes/        remote dashboard and fetch/pull/push operations
internal/provider/       optional provider-neutral GitHub client and caches
internal/plugins/        plugin discovery, state, protocol negotiation, and runtime
internal/registry/       bounded multi-repository discovery and refresh engine
internal/commands/       command-palette matching
internal/notifications/  bounded user notification model

internal/ui/             UI package documentation
internal/ui/layout/      responsive layout calculations
internal/ui/table/       virtualized status table
internal/ui/details/     selected-file metadata
internal/ui/diff/        diff presentation model
internal/ui/theme/       semantic terminal capability and color handling
internal/ui/mouse/       mouse hit testing
internal/ui/*view/       workspace-specific pure presentation models

internal/config/         versioned configuration, migration, env, and CLI merging
internal/platform/       sanitized logs, URL/clipboard helpers, and OS boundaries
internal/integration/    real-repository workflow scenarios
internal/architecture/   static process-boundary tests
internal/testutil/       disposable repository helpers
internal/version/        build-time version, commit, and date identity
internal/releasearchive/ deterministic release archive writer
internal/releasepackcmd/ private release-packaging command
pkg/plugin/              public dependency-free plugin wire SDK
examples/                buildable plugin examples
```

## Runtime flow

1. Load and validate configuration.
2. Resolve the repository root and Git metadata with typed Git arguments.
3. Probe repository capabilities and obtain an initial authoritative snapshot.
4. Start filesystem watching or polling according to configuration and platform availability.
5. Coalesce refresh hints and permit one status operation per active repository.
6. Parse machine-readable Git output into immutable domain snapshots.
7. Diff old and new snapshots into typed activity and notification events.
8. Deliver messages to Bubble Tea; render only in-memory state.
9. Execute mutations asynchronously and force an authoritative refresh when they complete.
10. Cancel watchers, network/history jobs, plugins, and Git children during shutdown.

When enabled, the status workspace separately loads a bounded presentation
commit graph. Its request is cancellable and generation-scoped; HEAD/ref
identity prevents unnecessary reloads, and a graph failure never blocks the
authoritative status snapshot.

The lower-left status context region can switch between the optional commit
graph, bounded commits ahead of the configured upstream, and a read-only branch
summary. Built-in context shortcuts work without configuration and are merged
with safe keymap overrides. Unpushed and branch-summary data are loaded through
the same cancellable, generation-scoped model flow and never perform mutations.

Commit-tree capture uses an explicit colorized Git format, then passes the
bounded output through `internal/ui/committree` before rendering. Semantic
segments are styled with theme roles; raw Git SGR, OSC, CSI, and other control
sequences never become terminal output. `NO_COLOR` uses the same parser and
returns safe colorless text.

## Process boundaries

The primary status command is:

```text
git status --porcelain=v2 -z --branch --untracked-files=all
```

Git is invoked through `exec.CommandContext` with an executable and argument slice. Application packages do not construct shell command strings. Paths and refs are validated and passed as distinct arguments, with `--` where the Git command supports it.

The architecture test permits direct process execution only at explicit boundaries:

- `internal/git` for Git;
- `internal/provider` for optional provider authentication helpers;
- `internal/plugins` for supervised out-of-process plugins; and
- `internal/platform` for validated operating-system integration.

## Concurrency and refresh

- Bubble Tea owns UI state.
- The refresh coordinator deduplicates status work and records a dirty bit when another hint arrives during a refresh.
- The repository engine uses bounded workers; it never starts an unbounded watcher or process per repository.
- History, remote, provider, and plugin operations are context-cancellable.
- Watcher goroutines terminate through context cancellation and close their output channels.
- Channels, activity history, plugin output, provider responses, and history pages are bounded.

Filesystem events are debounced at approximately 75 milliseconds. Polling defaults to two seconds. Low-frequency reconciliation may run in filesystem mode to recover from dropped events.

## Safety and rendering

The root view composes responsive workspace models. Wide terminals show side-by-side status and details/diff content; narrow terminals expose equivalent overlays or switchable views. Width calculations use terminal display width rather than byte length.

Repository-controlled text, plugin output, provider text, refs, paths, and diagnostics are sanitized before display. Color never carries status meaning by itself, `NO_COLOR` is honored, and motion can be reduced or disabled.

Destructive or history-changing actions require a confirmation naming the affected paths or refs. Every successful mutation refreshes the affected authoritative state.
