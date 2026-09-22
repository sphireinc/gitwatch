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
| Interactive rebase | Rebase lifecycle and plan boundaries exist; complete workspace is not shipped | 126–132 | `TestRebaseConflictResumeParityScenario` real conflict/continue fixture plus `go test ./internal/rebase ./internal/sequencer`; restart, skip, abort, interactive-plan, and external-editor checks remain required |
| Squash, fixup, reword, edit, drop, reorder | Not shipped | 129–131 | Rebase-plan parser/model tests plus real-repository result and stale-state tests |
| Fixup commits and autosquash | Not shipped | 130 | Real-repository fixup/autosquash integration test and refresh assertion |
| Cherry-pick single, multiple, range, and merge-mainline | Conflict lifecycle adapter exists; complete workspace is not shipped | 133–135 | `TestCherryPickConflictResumeParityScenario` real conflict/resolve/continue fixture plus `go test ./internal/cherrypick`; ordered, empty, merge-mainline, skip, abort, and native workspace evidence remain required |
| Revert as a durable sequencer workflow | Typed resumable adapter exists; complete workspace is not shipped | 136 | `TestRevertConflictResumeParityScenario` real conflict/continue fixture plus multi-commit/conflict/abort and post-step status-refresh evidence |
| Merge strategies | Pull strategy selection exists; merge workspace is not shipped | 137–143 | `TestMergeConflictResumeParityScenario` covers a real typed conflict/resolve/continue flow; fast-forward, no-ff, abort, and external-resolution fixtures remain required |
| Conflict resolution | Conflicts are represented in status; unified resolver is not shipped | 138–143 | Unmerged-index fixtures, guarded file-write tests, external-change refusal, and shared merge/rebase/cherry-pick/revert scenarios |
| Reflog recovery | Reflog timestamps support remote/status views; recovery browser is not shipped | 144–147 | Bounded reflog parser tests, `TestAdvancedHistoryAndComparisonParityScenario` real recovery-point fixture, divergence checks, and safe undo/redo integration tests |
| Bisect | Bounded engine and state loader exist; workspace is not shipped | 148–150 | `TestBisectParityScenarioSurvivesFreshLoaderAndReset` real good/bad/skip/reset and fresh-loader fixture; bounded cancellation and optional argv command tests remain required |
| Submodules | Submodule status is surfaced; lifecycle/navigation is not shipped | 151–154 | Real parent/child repositories covering initialized, missing, dirty, detached, nested, URL-redaction, and bounded bulk operations |
| Tags and signing | Tag inspection exists in history; full lifecycle/signature operations are not shipped | 155–156 | `TestAdvancedHistoryAndComparisonParityScenario` real lightweight/annotated tag fixture; create/sign/verify/push/delete fixtures with explicit confirmation and remote error handling |
| Remote CRUD and remote branches | Remote dashboard and fetch/pull/push flows exist; CRUD/branch management is not shipped | 157–160 | Real bare-remote fixtures for add/remove/prune, branch tracking, divergence, reset/rebase safeguards, and comparison |
| Revision comparison | Parent-relative commit diffs exist; arbitrary revision comparison is not shipped | 161 | `TestAdvancedHistoryAndComparisonParityScenario` real revision/path comparison with bounded output |
| File history and blame | Bounded history and commit inspection exist; path history/blame are not shipped | 162–163 | `TestAdvancedHistoryAndComparisonParityScenario` real path-history/blame fixture; renames, binary files, unicode, and bounded scrolling/search remain required |
| File-tree status mode | Flat status presentation exists | 165 | Pure layout/parser tests plus 80x24 keyboard/mouse and `NO_COLOR` acceptance fixture |
| External editor, difftool, and mergetool | OS/browser boundaries exist; workflow handoff is not shipped | 142, 166 | Controlled argv/process-boundary integration fixtures, cancellation, timeout, terminal restoration, and sanitized diagnostics |
| Lightweight custom commands | Safe argv execution and host-rendered prompt forms exist; richer command authoring remains limited | 167–169 | `TestCustomCommandPromptFormBlocksExecutionUntilSubmit`, `TestCustomCommandConfirmationCanBeCancelled`, and `scripts/parity-check.sh` cover prompt validation, cancellation, bounded output, redaction, and plugin protocol compatibility; native keyboard/mouse matrix remains required |
| GitHub PR/review/actions workflows | Read-only PR/check visibility exists; lifecycle operations are not shipped | 170–173 | Provider integration fixtures for auth failure, mutation confirmation, pagination, redaction, cancellation, and refresh |
| Multi-repository workflows | Repository discovery, bounded refresh, groups, and dashboard exist; advanced operations are not repository-scoped across all lanes | 125, 174–178 | `scripts/parity-check.sh` runs `TestMultiRepositoryRefreshTransitionScenario`; 50-repository bounded-worker tests, mixed healthy/broken fixtures, interleaved operations, cancellation, and generation-switch checks remain required |

## Differentiation matrix

These are the capabilities required for gitwatch to supersede rather than
merely imitate LZ.

| Differentiator | Current baseline | Closing tasks | Executable acceptance evidence |
|---|---|---:|---|
| Live watcher-first authoritative status | Shipped through `internal/watch` and porcelain-v2 snapshots | 124, 183–186 | `scripts/parity-check.sh` runs external-edit integration tests plus fsnotify/polling manager lanes; recreated metadata coverage passes 100 repeated local runs, while event-storm benchmarks and native cross-platform evidence remain required |
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
