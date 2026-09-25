# Task 143: Create unified sequencer conflict coordinator

**Phase:** Merge and conflicts
**Depends on:** 135, 136, 137, 140, 141

## Goal

Make conflict handling identical across rebase, cherry-pick, revert and merge.

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

1. Create coordinator logic mapping sequencer state + conflict snapshot to valid actions.
2. Derive Continue/Skip/Abort availability from operation kind and current Git state; invalid actions are not rendered.
3. Zero unresolved index entries does not automatically mean Continue is valid; check edit/amend/commit requirements.
4. Route all operation-specific conflict entry points to the same conflict packages.
5. Detect external continue/abort on watcher refresh and update/close views generation-safely.
6. Emit one operation-attention notification path rather than duplicate feature toasts.

## Verification

- Same conflict fixture under merge/rebase/cherry-pick/revert.
- External continue/abort while UI is open.
- Repository switch during conflict.

## Acceptance criteria

- [x] One resolver/coordinator serves every sequencer workflow.
- [x] No duplicate conflict parser or lifecycle state machine remains.

## Completion record

- [x] Implementation commits recorded, including `1e1edc9`, `05dda0f`,
  `28021b7`, and `f254971`.
- [x] Exact tested revision recorded (`34562b5`; full local and hosted
  verification is recorded below).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual acceptance for Task 143 remains open. The user's all-cell
  sign-off is specific to candidate `5b2a8e9`; the carry-forward to `34562b5`
  was explicitly recorded for Task 137. Since `34562b5` changes merge-engine
  error propagation used by merge recovery, Task 143 awaits a separate
  carry-forward decision or fresh operator evidence.
- [x] Known limitations/deferred work documented: Linux cells use the explicit
  owner-approved macOS-equivalence decision while physical multi-distribution
  testing continues; no fresh per-platform transcript was captured at
  `34562b5`.

## Progress evidence

- Centralized recovery action availability in `conflictview.RecoveryActions`.
  Rendering and app lifecycle input now use the same operation-aware decision
  for Continue, Skip, and Abort across rebase, cherry-pick, revert, and merge.
- Continue is withheld when unresolved conflicts remain; for non-rebase
  operations it is also withheld when the authoritative staged count is known
  to be zero. Skip is limited to Git workflows that support `--skip`.
- Focused tests cover unresolved and clean-index gating plus the shared
  lifecycle matrix for all four sequencer workflows.
- Status recovery routing and the progress workspace now include revert and
  merge alongside rebase and cherry-pick; revert uses the same ordered
  progress presentation as cherry-pick.
- A refresh that observes a previously active sequencer disappear with no
  conflicts now closes the recovery workspace to Status and reports the
  externally completed/aborted transition. App coverage exercises this path.
- A revert started from History now follows the same authoritative refresh
  route as merge: when Git leaves a revert sequencer active after a conflict,
  the app opens the shared Revert recovery workspace with the authoritative
  conflict list. App coverage verifies the route and status message.
- Conflict attention notifications are now emitted on entry into a conflicted
  state or operation transition, rather than duplicated on every watcher
  refresh. App coverage verifies the single-notification path.
- Fallback refresh snapshots now carry repository generation metadata and stale
  snapshots are discarded after a repository switch. App coverage verifies
  that an old repository cannot overwrite current recovery state.
- The implementation slices now cover merge/revert entry points, external
  state transitions, repository-switch generation handling, and a single
  attention-notification path. Task remains active for the final coordinator
  acceptance audit and native/manual evidence.

- Added a single `operation_recovery` palette route for every recoverable
  repository-scoped operation (rebase, cherry-pick, revert, merge, and bisect),
  with kind-specific workspace presentation selected from the authoritative
  sequencer snapshot. App coverage exercises rebase, cherry-pick, revert, and
  merge routing. The task remains open for final coordinator audit and native
  acceptance evidence.

- At revision `1e1edc9`, the focused coordinator tests passed under race
  detection, and the parity harness passed the real cherry-pick, merge,
  rebase, revert, bisect, watcher, and multi-repository lanes. Native/manual
  terminal acceptance and the final requirement-by-requirement coordinator
  audit remain open.

- Revision `05dda0f` adds the UI-neutral `sequencer.RouteFor` coordinator and
  routes palette reopening, Status recovery navigation, snapshot-triggered
  revert recovery, and recoverability checks through the same operation-kind
  map. Coverage verifies every durable operation kind; normal and focused race
  tests pass. Native terminal acceptance and the deeper shared lifecycle
  abstraction remain open.

- Revision `28021b7` moves lifecycle action availability into the UI-neutral
  `sequencer.ActionsFor` matrix. The conflict view now consumes that shared
  decision rather than maintaining its own operation switch. Matrix coverage
  verifies unknown, rebase edit-stop/conflict, cherry-pick, revert, and merge
  states; focused normal, race, vet, and formatting checks pass. Native
  acceptance and the broader end-to-end coordinator audit remain open.
- At revision `201f441`, the full `go test ./...` gate passed after this
  coordinator slice, including the real integration conflict/resume scenarios.
- At revision `bd2603f`, the coordinator package set passed with
  `go test ./internal/sequencer ./internal/ui/conflictview ./internal/git
  ./internal/app ./internal/integration`. This verifies the shared action and
  route matrices, conflict-view gating, Git lifecycle boundary, app recovery
  routing, and real conflict/resume integrations. Native/manual acceptance and
  the final requirement-by-requirement audit remain open.
- At revision `f254971`, rebase continue/abort, cherry-pick continue/skip/abort,
  and merge abort now delegate to the shared `Runner.OperationLifecycle`
  boundary; production adapters no longer construct separate lifecycle Git
  invocations. The authoritative rebase projection identifies an edit-stop
  only when Git's completed todo records the stopped SHA as `edit`. The shared
  action matrix now requires staged results for ordinary rebase/cherry-pick/
  revert/merge continuation, permits a clean-index Continue only for a
  Git-derived rebase edit-stop, and always blocks Continue while conflicts are
  unresolved. Regression tests cover real edit-stop restart/skip, conflicted
  rebase identification, and the shared action matrix.
  `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 with lint
  (0 issues), full tests, race, vet, formatting, security fuzz, performance,
  and diff checks. The task remains open for broader cross-platform/native
  acceptance and its full end-to-end coordinator audit.
- At revision `122c8de`, the production call-site audit confirmed all
  rebase/cherry-pick/revert/merge lifecycle verbs use
  `Runner.OperationLifecycle`; conflict UI rendering and input both consume
  `sequencer.ActionsFor`, and operation recovery routing uses `RouteFor`.
  Focused coverage includes `TestRecoveryCoordinatorSharesLifecycleRulesAcrossSequencers`,
  all four real conflict/resume parity scenarios, fresh-runner revert abort,
  external sequencer completion, and the edit-stop lifecycle variants. Full
  local `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64, and
  hosted Actions run `35864391354` passed quality/policy, secret scanning, and
  Ubuntu/macOS/Windows matrices with macOS PTY acceptance. Native/manual
  acceptance remains open.
- Final production call-site audit found `sequencer.ActionsFor` supplies the
  conflict pane's rendered and accepted recovery actions, `sequencer.RouteFor`
  supplies app recovery routing, and rebase/cherry-pick/revert/merge verbs
  converge on `Runner.OperationLifecycle`. App tests cover external
  sequencer completion, single notification per operation transition, and
  stale snapshots after repository switches; integration fixtures cover
  continue/resume for each of the four sequencers. At `06771d6`, local
  `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  make check` passed (lint 0 issues, normal/race tests, vet, format, security,
  performance). Hosted run `36065168227` passed all five jobs on retry after
  one transient reflog fuzz timeout; Windows path/CRLF and Unix PTY checks
  passed. Native/manual acceptance remains outstanding.

## Merged-main coordinator acceptance (2026-09-25)

- Dependency metadata now follows the selected graph: Task 135 precedes Task
  143, and Task 143 precedes Task 132. Task 143 depends on 135, 136, 137, 140,
  and 141; Task 132 depends on 143.
- Serena's current-source audit confirmed that `sequencer.ActionsFor`
  centralizes Continue/Skip/Abort availability, `sequencer.RouteFor` supplies
  operation recovery destinations, and the app delegates workspace routing to
  that shared map. Continue is blocked while conflicts remain; ordinary
  cherry-pick, revert, and merge continuation requires staged results, while
  rebase edit-stop is Git-derived. Skip is restricted to supporting operations.
- Regression coverage includes shared lifecycle-action rules, route coverage
  for durable recovery kinds, four real conflict/resume parity scenarios,
  external sequencer completion, fresh-snapshot recovery, repository-generation
  isolation, and a single attention notification per operation transition.
- At application revision `34562b5`, local
  `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache
  make check` passed on Darwin arm64, and hosted Actions run
  [36143404629](https://github.com/sphireinc/gitwatch/actions/runs/36143404629)
  passed quality/policy, full-history secret scanning, and the Ubuntu, macOS,
  and Windows jobs. The Windows race job is skipped by workflow configuration.
  No Go source has changed since `34562b5`.
- The owner-provided all-cell green disposition for candidate
  `5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725` records macOS and Windows green,
  with Linux accepted by macOS equivalence rather than a physical Linux run.
- The user explicitly carried this disposition to `34562b5` for Task 137. The
  Task 143 operator gate remains open pending the separate decision requested,
  because that revision changes merge-engine error propagation used by merge
  recovery.

Task 143's implementation and automated verification audit is complete. Keep
the task active until its native/operator acceptance gate is resolved.
