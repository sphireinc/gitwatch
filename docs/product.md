# gitwatch Product Contract

## Product promise

gitwatch is an interactive, htop-style dashboard for one or more Git worktrees. It keeps repository state visible, makes the staged/unstaged split understandable, and provides safe, reversible day-to-day status actions without asking users to leave the terminal.

“htop-style” is an operational requirement, not a visual metaphor: the application continuously updates from authoritative Git snapshots, presents dense summary metrics, lets selection drive actions, provides drill-down details, reports activity and operation state, and supports equivalent keyboard and mouse controls.

## Personas

- **Daily contributor:** needs to inspect a busy worktree, stage or unstage exact paths, and quickly understand branch and conflict state.
- **Reviewer/debugger:** needs a readable selected-file summary and on-demand staged or unstaged diffs without losing the dashboard context.
- **Release/operator user:** needs explicit operation feedback, safe handling of unusual filenames and repositories, and useful diagnostics when Git or filesystem facilities fail.
- **Maintainer:** needs predictable, testable packages, documented configuration, cross-platform behavior, and no telemetry or hidden destructive actions.

## Primary workflows

1. Launch `gitwatch` from a repository or nested directory.
2. Confirm branch/HEAD, upstream divergence, repository health, watcher mode, and last refresh in the header and metric bar.
3. Navigate the status table with arrows or `j`/`k`, optionally filter and sort it, and inspect the selected path.
4. Press Space to stage or unstage the selected path when the status has an unambiguous safe operation. Use `a` or `U` only after the scope is visible in the UI.
5. Open an on-demand staged or unstaged diff with `d`; return with Escape.
6. Observe operation success/failure and the authoritative post-operation refresh in the activity strip.
7. Quit with `q` or Ctrl-C and leave the terminal restored.

Mouse clicks select rows and explicit stage controls; wheel events scroll the focused pane. Mouse never gains destructive powers that are unavailable to the keyboard.

## v1 feature matrix

| Area | v1 contract |
|---|---|
| Repository | Discover normal, nested, detached, unborn, linked-worktree, submodule, and `.git`-file contexts; reject bare repositories clearly. |
| Status | Parse `git status --porcelain=v2 -z --branch --untracked-files=all` without human-format parsing. |
| Dashboard | Live branch/upstream/divergence metrics, status counts, file table, details, activity, footer/help, responsive layouts. |
| File actions | Safe single-file stage/unstage, explicit bulk stage/unstage, and optional confirmed restore/discard. |
| Inspection | On-demand staged/unstaged diff, binary-file indication, conflict details, filtering and sorting. |
| Refresh | Filesystem watcher with debounce and polling fallback, authoritative Git refresh, bounded/coalesced concurrency. |
| Accessibility | Keyboard parity, mouse support, `NO_COLOR`, semantic themes, high-contrast-safe text/symbols, reduced/off motion. |
| Reliability | Typed errors, cancellation, terminal restoration, sanitized untrusted text, bounded activity history, debug logging opt-in. |
| Distribution | macOS, Linux, and Windows binaries, reproducible archives/checksums, MIT license, and contributor/security documentation. |

## Terminology

- **Path:** a repository-relative byte sequence displayed with terminal-safe escaping.
- **Staged:** the index differs from `HEAD` for the path.
- **Unstaged:** the worktree differs from the index for the path.
- **Untracked:** a path exists in the worktree but is not in the index.
- **Conflict:** Git reports an unmerged index state; generic destructive shortcuts are unavailable.
- **Snapshot:** an immutable, authoritative observation produced by Git and tagged with a generation and timing metadata.
- **Refresh hint:** a watcher or timer signal that requests a new snapshot; it is never treated as repository truth.

## Safety boundaries

- Every Git invocation uses an executable plus an argument vector. No shell, interpolation, or command string is permitted.
- Paths are passed as exact arguments after `--` wherever Git supports it; display escaping never changes the operation argument.
- Staging and unstaging are immediate but reversible and always followed by an authoritative refresh.
- Restore, discard, reset, clean, delete, and equivalent data-loss actions require a modal confirmation naming the exact paths and affected content. Generic `reset --hard` and `clean -fd` shortcuts are not part of v1.
- Unmerged paths are never silently included in an unrelated destructive action.
- Untrusted paths and Git output are sanitized before terminal rendering and diagnostics.
- Git remains the source of truth; gitwatch does not implement Git object or index semantics and does not collect telemetry.

## Explicit non-goals

v1 does not include commit authoring, interactive rebase, remote hosting APIs, GitHub/GitLab integration, an embedded Git implementation, or a plugin system. Stash management, branch mutation, history graphs, network dashboards, worktree management, multi-repository dashboards, and interactive hunk staging are post-v1 work unless separately promoted by a recorded decision.

## Release boundary

v1 is releasable only when all P0 tasks and the repository’s release criteria pass on the supported platforms, including real temporary-repository flows, 10,000 changed paths remaining usable, safe unusual filenames, conflict representation, clean process shutdown, installation/version/help behavior, and complete user/contributor/security documentation.
