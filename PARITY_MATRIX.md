# LZ workflow parity matrix

This matrix is the public planning baseline for the Supersede milestone. It
separates the workflow parity needed to replace LZ from the capabilities that
differentiate gitwatch. A row is not green because a key or placeholder exists:
acceptance must prove an end-to-end, restart-safe workflow against real Git
repositories while live status, repository scope, safety, and narrow-terminal
behavior remain intact.

## Workflow matrix

| LZ workflow | Current support in gitwatch | Task closing the gap | Executable acceptance evidence |
|---|---|---:|---|
| Interactive rebase | Not shipped; history inspection and guarded revert exist | 126–132 | `go test ./internal/rebase ./internal/sequencer`; disposable-repository rebase fixtures; restart, continue, skip, abort, and external-editor checks |
| Squash, fixup, reword, edit, drop, reorder | Not shipped | 129–131 | Rebase-plan parser/model tests plus real-repository result and stale-state tests |
| Fixup commits and autosquash | Not shipped | 130 | Real-repository fixup/autosquash integration test and refresh assertion |
| Cherry-pick single, multiple, range, and merge-mainline | Not shipped | 133–135 | `go test ./internal/cherrypick`; real-repository ordered, empty, merge, conflict, continue, skip, and abort fixtures |
| Revert as a durable sequencer workflow | Single-commit guarded revert exists; resumable lifecycle is not shipped | 136 | Real-repository multi-commit/conflict/continue/abort tests and post-step status refresh evidence |
| Merge strategies | Pull strategy selection exists; merge workspace is not shipped | 137–143 | Real-repository merge, fast-forward, no-ff, conflict, continue, abort, and external-resolution tests |
| Conflict resolution | Conflicts are represented in status; unified resolver is not shipped | 138–143 | Unmerged-index fixtures, guarded file-write tests, external-change refusal, and shared merge/rebase/cherry-pick/revert scenarios |
| Reflog recovery | Reflog timestamps support remote/status views; recovery browser is not shipped | 144–147 | Bounded reflog parser tests, real recovery-point fixtures, divergence checks, and safe undo/redo integration tests |
| Bisect | Not shipped | 148–150 | Real-repository good/bad/skip/reset fixtures, restart detection, bounded cancellation, and optional argv command tests |
| Submodules | Submodule status is surfaced; lifecycle/navigation is not shipped | 151–154 | Real parent/child repositories covering initialized, missing, dirty, detached, nested, URL-redaction, and bounded bulk operations |
| Tags and signing | Tag inspection exists in history; full lifecycle/signature operations are not shipped | 155–156 | Real tag create/sign/verify/push/delete fixtures with explicit confirmation and remote error handling |
| Remote CRUD and remote branches | Remote dashboard and fetch/pull/push flows exist; CRUD/branch management is not shipped | 157–160 | Real bare-remote fixtures for add/remove/prune, branch tracking, divergence, reset/rebase safeguards, and comparison |
| Revision comparison | Parent-relative commit diffs exist; arbitrary revision comparison is not shipped | 161 | Real-repository revision/range/path comparison tests with bounded output |
| File history and blame | Bounded history and commit inspection exist; path history/blame are not shipped | 162–163 | Real history/blame fixtures including renames, binary files, unicode, and bounded scrolling/search |
| File-tree status mode | Flat status presentation exists | 165 | Pure layout/parser tests plus 80x24 keyboard/mouse and `NO_COLOR` acceptance fixture |
| External editor, difftool, and mergetool | OS/browser boundaries exist; workflow handoff is not shipped | 142, 166 | Controlled argv/process-boundary integration fixtures, cancellation, timeout, terminal restoration, and sanitized diagnostics |
| Lightweight custom commands | Plugins provide a separate capability-bounded contract; custom command forms are not shipped | 167–169 | Safe argv template validation, bounded output/redaction tests, real command cancellation, and protocol compatibility fixtures |
| GitHub PR/review/actions workflows | Read-only PR/check visibility exists; lifecycle operations are not shipped | 170–173 | Provider integration fixtures for auth failure, mutation confirmation, pagination, redaction, cancellation, and refresh |
| Multi-repository workflows | Repository discovery, bounded refresh, groups, and dashboard exist; advanced operations are not repository-scoped across all lanes | 125, 174–178 | 50-repository bounded-worker tests, mixed healthy/broken fixtures, interleaved operations, cancellation, and generation-switch checks |

## Differentiation matrix

These are the capabilities required for gitwatch to supersede rather than
merely imitate LZ.

| Differentiator | Current baseline | Closing tasks | Executable acceptance evidence |
|---|---|---:|---|
| Live watcher-first authoritative status | Shipped through `internal/watch` and porcelain-v2 snapshots | 124, 183–186 | External-edit integration tests, event-storm benchmarks, polling fallback, and native cross-platform evidence |
| Repository-scoped durable operation state | Shared operation engine exists; sequencer domain is not shipped | 122–125, 143, 145–147 | Two-repository interleaving and restart/recovery integration fixtures |
| Observable operation history and recovery | Bounded activity/operation lifecycle exists; semantic journal is not shipped | 145–147, 176 | Timeline records, reflog correlation, undo safety, and late-result rejection tests |
| Repository health and attention | Independent repository errors and status summaries exist; health model is not shipped | 175 | Mixed-repository health fixtures, freshness/error badges, and bounded dashboard checks |
| Background remote intelligence | Explicit fetch/pull exists; optional background intelligence is not shipped | 177 | Cancellable bounded auto-fetch/provider fixtures with local-status priority checks |
| Cross-repository search and actionable graph | Repository dashboard and bounded commit tree exist | 178–180 | Search/palette and graph scalability tests at documented repository/history budgets |

## Claim rule

The project must not claim that gitwatch replaces or supersedes LZ until the
required parity rows, the differentiation rows, and Task 184’s executable
cross-platform evidence are accepted. Planned rows are not shipped features;
README and release copy must continue to describe only the current product.
