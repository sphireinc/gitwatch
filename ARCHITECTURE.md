# Architecture

## Package layout
```text
cmd/gitwatch/              executable entrypoint
internal/app/              root Bubble Tea model, messages, command wiring
internal/git/              Git runner, porcelain parser, operations, capabilities
internal/repo/             immutable repository snapshot/domain models
internal/watch/            fsnotify watcher, debounce, polling fallback
internal/ui/               screen composition and panels
internal/ui/theme/         semantic styles and terminal capability handling
internal/ui/components/    table/list, header, footer, timeline, detail pane, modals
internal/config/           defaults, config file, env and CLI merging
internal/history/          in-memory snapshot/event history
internal/platform/         OS-specific helpers
internal/testutil/         disposable repos and fixtures
```

## Runtime flow
1. Resolve repository root using Git.
2. Probe Git version/capabilities.
3. Obtain initial repository snapshot.
4. Start watcher on worktree and relevant `.git` paths.
5. Feed watcher events into a debounce/coalescing layer.
6. Run one refresh at a time. If more events arrive while refreshing, set a dirty bit and refresh once again afterward.
7. Parse Git output into an immutable `repo.Snapshot`.
8. Diff old/new snapshot into UI events.
9. Send snapshot/event messages to Bubble Tea.
10. Render without blocking.

## Git status command
Primary status acquisition:
```text
git status --porcelain=v2 -z --branch --untracked-files=all
```
Add capability-specific commands only when required for richer metadata. Never depend on localized human output.

## Concurrency
- Bubble Tea owns UI state.
- A refresh coordinator owns Git-status execution and deduplication.
- Mutating Git operations execute asynchronously and report typed completion messages.
- Watcher goroutines terminate via context cancellation.
- No unbounded channels.

## Refresh policy
Event driven by default. Debounce bursts at approximately 75 ms. Guarantee eventual refresh after the burst. Polling fallback defaults to 2 seconds and is user-configurable. Periodic low-frequency reconciliation (for example every 30 seconds) is allowed even in watcher mode to recover from missed events.

## Data model
`Snapshot` must contain at minimum repository root, git dir, branch/HEAD state, upstream, ahead/behind counts, staged/unstaged/untracked/conflicted entries, rename/copy metadata, summary counts, timestamp, refresh duration, and operation state.

## Rendering
The root view composes independent panels. Layout is responsive: wide terminals get multi-pane content; narrow terminals collapse details and timeline into switchable views. UI calculations use display width, not byte length.



