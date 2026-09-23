# Task 132: Complete rebase lifecycle recovery and continuation

**Phase:** Interactive rebase
**Depends on:** 127, 131, 143

## Goal

Make rebase durable: continue, skip, abort, restart recovery and conflict integration must work whether gitwatch or another terminal started it.

## Non-negotiable constraints

- Live filesystem-driven status is the product core. Do not replace, subordinate, or pause it except for the minimum repository-lock window required by Git itself.
- Filesystem events are refresh hints, never authoritative state. The authoritative worktree snapshot remains `git status --porcelain=v2 -z --branch --untracked-files=all` parsed into immutable repository state.
- Every successful mutation MUST request an authoritative refresh for the affected repository. Long-running sequencer operations must refresh after every observable state transition.
- Multi-repository support is first-class. New domain models and operations MUST carry repository identity/scope and remain correct while other repositories refresh or run unrelated work.
- Do not create unbounded watchers, goroutines, workers, or Git/provider/plugin processes. Reuse bounded registry/operation infrastructure.
- All Git commands use typed argv execution through the Git boundary. Never interpolate repository data into shell command strings. Use `--` where supported and machine-readable/NUL-delimited output where available.
- Bubble Tea owns UI state. Git/network/filesystem/process work never runs in the render path.
- Repository-controlled text is untrusted terminal input and MUST be sanitized before rendering.
- Destructive/history-rewriting actions require scope-specific confirmation. Keep the prohibition on generic `reset --hard`, raw `--force`, and `clean -fd` shortcuts.
- Keyboard and mouse must reach equivalent functionality. New views must work at 80x24, honor `NO_COLOR`, and support full/reduced/off motion.
- Do not reimplement Git. Use Git as source of truth and build safe typed control/presentation layers around it.
- Breaking config/plugin changes require versioning, migration, and compatibility fixtures.

## Implementation steps

1. Expose Continue, Skip and Abort whenever Task 123 reports an active rebase and the action is valid.
2. Show current commit, completed/remaining count where recoverable, conflict count, and edit-stop state.
3. Route unresolved conflicts into the unified conflict resolver from Tasks 138-143.
4. After every lifecycle command request status + operation-state refresh and remain in the workflow until Git reports completion.
5. Derive truth from Git after restart; persist only UI preference/selection, not rebase truth.
6. On completion show old/new HEAD and rewritten count; add semantic operation-history record.

## Git/process boundary

- `git rebase --continue`
- `git rebase --skip`
- `git rebase --abort`

## Verification

- Conflict rebase, edit-stop, skip, abort, restart mid-rebase.
- External conflict resolution and continuation.

## Acceptance criteria

- [ ] Any active rebase can be resumed or aborted after restart.
- [ ] Conflict handling uses unified resolver.
- [ ] Completion refreshes status/history/branch divergence.

## Completion record

- [x] Recovery entry point and progress presentation are implemented; an active rebase discovered by the live snapshot can be opened from Status with `C`, including when no conflict file is present. Continue and abort remain typed lifecycle intents in the existing conflict/recovery workspace.
- [x] Implementation commits recorded: `224362f` (active recovery route), `2374459` (continuation coverage and stale-marker fix), and the follow-up skip-support slice.
- [x] Exact tested revision recorded.
- [x] Focused tests recorded: `TestActiveRebaseWithoutConflictsHasRecoveryRoute`, `TestOperationLifecycleAbortsRebaseAndRefreshesOperationState`, `TestOperationLifecycleContinuesResolvedRebase`, `TestOperationLifecycleSkipsRebase`, and `TestDetectOperationStateSurvivesRunnerReconstructionDuringRebase`; the real disposable-repository tests create a rebase conflict, exercise typed abort/continue/skip, verify fresh-runner rediscovery, and verify authoritative post-operation snapshots and worktree content.
- [x] `go test ./...` recorded through `make check`.
- [x] Race/vet/lint/format evidence recorded through `GOCACHE=/tmp/gitwatch-go-cache make check` (lint reported 0 issues).
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented: edit-stop continue/amend/abort variants and native Linux/macOS/Windows operator evidence still require dedicated acceptance coverage.

## Local progress evidence (task remains active)

- `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 with formatting, lint, full tests, race tests, vet, security, and performance checks.
- Focused tests passed with host Git signing isolated using `GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=commit.gpgsign GIT_CONFIG_VALUE_0=false`.
- The operation detector now ignores a transition-only `REBASE_HEAD` after Git has removed the durable rebase state directory; the continuation test covers the regression.
- The continuation scenario starts the rebase outside the lifecycle call, resolves it through a newly configured runner, and verifies that a fresh authoritative snapshot reports no active operation.
- Status explicitly labels a stopped rebase with no conflicted paths as an `edit-stop`, alongside the Git-derived current commit and completed/remaining counts.
- Rebase recovery now accepts `skip` in the typed lifecycle boundary and exposes it as `s` in the recovery workspace; the real-repository test verifies Git completes the skipped rebase and clears the operation projection.
- Explicit restart coverage reconstructs a new runner/discovery against the paused repository, confirms the repository-scoped active rebase projection, and aborts it successfully.
- At revision `cd954ea`, the focused rebase recovery suite passed with
  `go test ./internal/app ./internal/git ./internal/integration`, including
  active-operation routing, continue/skip/abort lifecycle behavior, fresh-runner
  rediscovery, and authoritative post-operation snapshots. Native/manual
  terminal evidence remains open, so the task is not yet eligible for archival.
- The task is intentionally not moved to `tasks/completed` until restart recovery, edit-stop/skip behavior where applicable, and required native/manual evidence are proven.
- At revision `0d2ee23`, `TestInteractiveRebaseEditStopSurvivesRestartAndCanSkip`
  drove a real interactive rebase to an `edit` stop, reconstructed its
  repository-scoped progress through a fresh runner, then skipped the stopped
  commit and verified Git's completed operation state and resulting worktree.
  `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 for this
  source revision: pinned lint (0 issues), formatting, full tests, race tests,
  vet, security fuzz, performance budgets, and diff checks. Native terminal
  evidence and edit-stop continue/amend/abort variants remain open.
- Hosted Actions run `35825994043` on `c2941d6` passed Quality and policy,
  full-history secret scan, Ubuntu full tests/race/build/runtime/PTY checks,
  Windows full tests/build/runtime/path-and-CRLF parity, and macOS full
  tests/race/build/runtime smoke. The overall run remains failed only at the
  macOS PTY step because `tmux` is absent in the workflow environment; this is
  not a Task 132 test failure. Cross-platform native/manual acceptance remains
  open.
- At revision `a4d6d83`, real interactive-rebase edit-stop tests cover restart
  followed by skip, continue without changes, amend followed by continue, and
  restart followed by abort. They verify Git-derived stopped state, operation
  cleanup, branch/HEAD outcome, and final worktree content. The full local
  `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 (lint 0
  issues, full tests, race, vet, formatting, security fuzz, performance, and
  diff checks). Hosted Actions run `35863324235` passed quality/policy,
  full-history secret scan, and Ubuntu/macOS/Windows test matrices, including
  the PTY smoke on macOS. Native operator acceptance remains open, so the task
  remains active.
- Revision `56ccbd3` shows the Git-derived phase/edit-stop label, current
  commit, and completed/remaining counts directly in the recovery pane. A
  focused 80x24 view test passed. A new native tmux fixture on Darwin arm64
  launched gitwatch after an externally started conflict rebase, opened
  recovery with `C`, verified the progress pane under `NO_COLOR=1` and motion
  off, skipped the stopped commit with `s`, and verified Git restored the
  expected branch/HEAD and removed rebase metadata. The fixture is now part
  of Unix CI. Full `GOCACHE=/tmp/gitwatch-go-cache
  GOMODCACHE=/tmp/gitwatch-go-mod-cache make check` passed at the source
  revision (lint 0 issues, full/race tests, vet, format, diff, security, and
  performance). This is automated PTY evidence for one skip path, not full
  human/operator, abort/continue, edit-stop, or native Windows acceptance;
  the task remains active.
- Hosted Actions run
  [35912038087](https://github.com/sphireinc/gitwatch/actions/runs/35912038087)
  on `46443d0` passed quality/policy, full-history secret scan, Ubuntu and
  macOS tests/race/build/runtime plus the new native PTY rebase recovery
  fixture, and the Windows test/build/runtime/path-parity matrix. The Unix PTY
  result does not constitute native Windows terminal interaction or a complete
  operator-run rebase lifecycle matrix.
- Revision `e3c0afd` records an authoritative post-lifecycle snapshot before
  journaling rebase recovery actions. When Git reports completion, the status
  message and semantic operation journal show the original and result HEADs;
  the journal records the number of newly reachable commit objects when Git's
  original/onto/result OIDs permit an exact count. Completion also schedules
  history reload alongside status/operation refresh. Real-repository tests
  cover skipping a conflicting commit and replaying one later commit (count
  one), plus skipping into another conflict without claiming completion or a
  final count. Focused tests and full `GOCACHE=/tmp/gitwatch-go-cache
  GOMODCACHE=/tmp/gitwatch-go-mod-cache make check` passed on Darwin arm64,
  including lint (0 issues), full/race tests, vet, formatting, diff, security,
  and performance checks. Native/operator acceptance remains open.
