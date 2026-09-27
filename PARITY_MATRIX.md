# LZ workflow parity matrix

This matrix is the public planning baseline for the Supersede milestone. It
separates the workflow parity needed to replace LZ from the capabilities that
differentiate gitwatch. A row is not green because a key or placeholder exists:
acceptance must prove an end-to-end, restart-safe workflow against real Git
repositories while live status, repository scope, safety, and narrow-terminal
behavior remain intact. “Implemented” describes source capability only; it does
not waive the linked automated, cross-platform, or native acceptance evidence.

## Workflow matrix

| LZ workflow | Current support in gitwatch | Task closing the gap | Executable acceptance evidence |
|---|---|---:|---|
| Interactive rebase | Git-derived recovery workspace and controlled rebase-plan editing are implemented; native lifecycle acceptance remains a separate gate | 126–132 | `scripts/parity-check.sh` runs `TestRebaseConflictResumeParityScenario`; restart/edit-stop/skip/abort UI and lifecycle tests (`TestActiveRebaseWithoutConflictsHasRecoveryRoute`, `TestRebaseRecoveryRecordsResultHeadAndRewrittenCount`, and `TestRebaseRecoveryShowsGitDerivedProgressAtNarrowWidth`) run in the full CI OS matrix |
| Squash, fixup, reword, edit, drop, reorder | Not shipped | 129–131 | Rebase-plan parser/model tests plus real-repository result and stale-state tests |
| Fixup commits and autosquash | Not shipped | 130 | Real-repository fixup/autosquash integration test and refresh assertion |
| Cherry-pick single, multiple, range, and merge-mainline | Repository-scoped progress workspace, durable continuation, skip, and conflict recovery are implemented; native acceptance remains separate | 133–135 | `scripts/parity-check.sh` runs `TestCherryPickConflictResumeParityScenario`; `TestCherryPickViewShowsRepositoryScopedProgress` and `TestExternalCherryPickResolutionEnablesContinueFromFreshSnapshot` run in the full CI OS matrix |
| Revert as a durable sequencer workflow | Typed multi-commit lifecycle, common conflict presentation, and restart recovery are implemented | 136 | `scripts/parity-check.sh` runs `TestRevertConflictResumeParityScenario`; `TestRevertConflictAbortAfterFreshRunnerParityScenario` and `TestRevertViewUsesCommonProgressAndRecoveryPresentation` run in the full CI OS matrix; post-step snapshot refresh is asserted by the real-repository scenarios |
| Merge strategies | Typed branch-merge strategy selection and the shared sequencer conflict lifecycle are implemented; broader parity remains evidence-gated | 137–143 | `scripts/parity-check.sh` runs `TestMergeConflictResumeParityScenario`; `TestInvalidMergeSourceAndStrategyAreRejected`, `TestBranchMergePromptRequiresCleanWorktreeAndExplicitStrategy`, and `TestConflictingMergeReturnsPausedStateAndAbortRestoresWorktree` run in the full CI OS matrix |
| Conflict resolution | Shared Git-derived sequencer coordinator and guarded resolution workspace cover rebase, cherry-pick, revert, and merge | 138–143 | `scripts/parity-check.sh` runs the four real conflict/resume scenarios; `TestConflictWorkspaceRouteAndResolutionIntent` and `TestResolveMultipleConflictFilesAndRefreshAuthoritativeState` run in the full CI OS matrix |
| Reflog recovery | Bounded reflog browser, semantic operation journal, safe undo, and guarded redo are implemented; native acceptance remains separate | 144–147 | `TestLoadParsesBoundedReflogRecords`, `TestAdvancedHistoryAndComparisonParityScenario`, `TestReflogRecoveryPointActionsUseExistingSafeFlows`, `TestExecuteUndoImmediatelyPreservesWorktreeContent`, and `TestExecuteRedoReplaysSuccessfulSoftUndo` |
| Bisect | Bounded state loader, repository-scoped workspace, and optional argv-only automated run are implemented | 148–150 | `TestBisectParityScenarioSurvivesFreshLoaderAndReset`, `TestBisectWorkspaceCompletesManualLoopInRealRepository`, bounded-cancellation and argv tests; these lanes run in `scripts/parity-check.sh` and the CI OS matrix |
| Submodules | Submodule status is surfaced; lifecycle/navigation is not shipped | 151–154 | Real parent/child repositories covering initialized, missing, dirty, detached, nested, URL-redaction, and bounded bulk operations |
| Tags and signing | Tag inspection exists in history; full lifecycle/signature operations are not shipped | 155–156 | `TestAdvancedHistoryAndComparisonParityScenario` real lightweight/annotated tag fixture; create/sign/verify/push/delete fixtures with explicit confirmation and remote error handling |
| Remote CRUD and remote branches | Remote dashboard and fetch/pull/push flows exist; CRUD/branch management is not shipped | 157–160 | Real bare-remote fixtures for add/remove/prune, branch tracking, divergence, reset/rebase safeguards, and comparison |
| Revision comparison | Bounded arbitrary revision/path comparison is implemented | 161 | `TestAdvancedHistoryAndComparisonParityScenario` real revision/path comparison and `internal/compare` tests; native navigation evidence remains tracked separately |
| File history and blame | Bounded path-history and blame workspaces are implemented | 162–163 | `TestAdvancedHistoryAndComparisonParityScenario`, `TestPathHistoryFromStatusPreservesStatusContext`, and `TestPathHistoryReadyAndActions`; native keyboard/mouse acceptance remains separate |
| File-tree status mode | Flat and collapsible tree presentations over the full logical status set are implemented | 165 | `TestStatusTreeMouseRowHeightsAreViewportBounded`, tree-selection/filtering tests, and 80x24/`NO_COLOR` acceptance coverage |
| External editor, difftool, and mergetool | OS/browser boundaries exist; workflow handoff is not shipped | 142, 166 | Controlled argv/process-boundary integration fixtures, cancellation, timeout, terminal restoration, and sanitized diagnostics |
| Lightweight custom commands | Safe argv execution and host-rendered prompt forms exist; richer command authoring remains limited | 167–169 | `TestCustomCommandPromptFormBlocksExecutionUntilSubmit`, `TestCustomCommandConfirmationCanBeCancelled`, and `scripts/parity-check.sh` cover prompt validation, cancellation, bounded output, redaction, and plugin protocol compatibility; native keyboard/mouse matrix remains required |
| GitHub PR/review/actions workflows | Read-only PR/check visibility exists; lifecycle operations are not shipped | 170–173 | Provider integration fixtures for auth failure, mutation confirmation, pagination, redaction, cancellation, and refresh |
| Multi-repository workflows | Repository discovery, bounded refresh, groups, and dashboard exist; advanced operations are not repository-scoped across all lanes | 125, 174–178 | `scripts/parity-check.sh` runs `TestMultiRepositoryRefreshTransitionScenario` and `TestBatchFetchFiftyDisposableRepositoriesIsBoundedAndFailureIsolated`; the real 50-repository lane covers mixed healthy/broken fixtures, bounded workers, and cancellation, while interleaved operations, generation-switch checks, and native evidence remain required |

## Differentiation matrix

These are the capabilities required for gitwatch to supersede rather than
merely imitate LZ.

| Differentiator | Current baseline | Closing tasks | Executable acceptance evidence |
|---|---|---:|---|
| Live watcher-first authoritative status | Shipped through `internal/watch`, porcelain-v2 snapshots, and bounded refresh coalescing | 124, 183–186 | `TestRefreshCoordinatorBoundsChildrenDuringWatcherEventStorm`, external-edit integration, fsnotify/polling manager lanes, and the 14,953/50,000-row performance gates; Task 183 native responsiveness remains open |
| Repository-scoped durable operation state | Shared operation admission and Git-derived sequencer projections are implemented | 122–125, 132–147 | `scripts/parity-check.sh` runs real conflict/restart scenarios; `TestActiveSequencerRecoveryPaletteRoutesAllOperationKinds` and repository-generation isolation tests run in the full CI OS matrix |
| Observable operation history and recovery | Semantic operation journal, guarded undo/redo, and reflog-guided recovery are implemented | 145–147, 176 | `TestOperationJournalCanFilterByRepository`, `TestExecuteUndoImmediatelyPreservesWorktreeContent`, `TestExecuteRedoReplaysSuccessfulSoftUndo`, `TestJournalRedoRefusalOpensGuidedReflogRecovery`, and stale-result tests |
| Repository health and attention | Repository-scoped health rows include local state, cached CI attention, staleness, and measured fetch latency | 175 | `TestComputeOfflineCleanRepositoryRemainsHealthy`, `TestViewShowsMeasuredAutoFetchLatency`, `TestViewShowsCachedCIAttentionAndStaleness`, and dashboard filtering/sorting tests; native acceptance remains open |
| Background remote intelligence | Bounded, cancellable auto-fetch and provider scheduling are implemented with local-status priority | 177 | `TestAutoFetchFinishedRecordsMeasuredLatencyForRegisteredAndNewRepos`, `TestRunOncePrunesRepositorySchedulerState`, cancellation/cache tests, and the CI OS matrix; native acceptance remains open |
| Cross-repository search and actionable graph | Repository-aware palette search and scalable actionable commit-graph lanes are implemented | 178–180 | `TestCommandPalette50RepositoriesAllocationBudget`, `TestViewPreservesMultipleGraphLanes`, graph-scale benchmarks, and the full CI OS matrix; native acceptance remains tracked in Tasks 178–180 |

## Claim rule

The project must not claim that gitwatch replaces or supersedes LZ until the
required parity rows, the differentiation rows, and Task 184’s executable
cross-platform evidence are accepted. Planned rows are not shipped features;
README and release copy must continue to describe only the current product.
