# Task 174: Add multi-repository batch fetch and safe pull

**Phase:** Multi-repository differentiation
**Depends on:** 125, 157, 159

## Goal

Make the repositories dashboard an operations console that goes beyond LZ’s single-repo focus.

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

1. Allow selected/group/all repositories batch fetch with a preview of exact repository set/remotes.
2. Add batch pull only with explicit strategies. Default recommendation is ff-only; merge/rebase batch modes require explicit opt-in and per-repo preflight.
3. Never batch-push by default; leave it out unless separately designed later.
4. Execute using bounded worker pool and existing per-repo write serialization.
5. Show per-repo queued/running/success/failure/skipped state live and retry failed subset.
6. One repository conflict/failure must not stop unrelated repositories; it becomes an attention row.
7. Local status refresh remains higher priority than batch network operations.

## Verification

- 50 disposable repos with clean/dirty/diverged/failure mix, cancel batch, process-count bound.

## Acceptance criteria

- [x] Batch operations are observable, bounded and failure-isolated.
- [ ] Watcher responsiveness remains acceptable during batch work.

## Completion record

- [x] Implementation commits recorded (`50a36b2`, `df1472b`, with follow-ups).
- [x] Exact tested revisions recorded (`172d00a` focused/parity; `91424f1` full gate).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded (`make check`, `91424f1`).
- [x] Race/vet/lint/format evidence recorded (`make check`, `91424f1`; focused race, `172d00a`).
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added a reusable typed batch executor for repository-scoped fetch and pull
  requests with explicit remote, branch, and strategy validation.
- The executor uses bounded worker concurrency, records queued/running,
  succeeded/failed/cancelled/skipped states, and isolates one repository
  failure from unrelated requests.
- Added focused unit tests for strategy validation, failure isolation,
  cancellation, and bounded execution; normal and race tests pass.
- Remaining: connect the executor to repositories-dashboard controls and typed
- Added an explicit repositories-dashboard `F` fetch-all confirmation path;
  each repository discovers its own configured remote and uses the typed
  remotes fetch operation. Results are summarized and failed repository paths
  remain visible as attention output before the dashboard reloads.
- Remaining: add live queued/running progress and failed-subset retry UI,
- Added `R` retry-failed confirmation that rebuilds the batch from failed
  repository results only and preserves unrelated successful rows.
- Remaining: add live queued/running progress, explicit batch-pull controls,
  and run the disposable-repository/process-count and watcher-responsiveness
  acceptance matrix.

## Progress evidence (2026-09-22)

- Added `TestRunFiftyRequestsKeepsWorkerBound`, which exercises 50 repository
  requests with a four-worker cap, measures peak concurrent work, verifies all
  requests succeed in input order, and guards against unbounded batch worker
  growth. Real disposable-repository and watcher-responsiveness acceptance
  remains open.

- Added an explicit repositories-dashboard `P` batch-pull action. It requires
  confirmation, uses `ff-only` as the visible strategy, carries each row's
  checked-out branch through the typed batch request, and never silently
  selects merge or rebase. Focused app coverage verifies the confirmation and
  cancellation path; live progress and real-repository acceptance remain open.

- Added bounded live progress messages for queued, running, succeeded, and
  failed repositories. The dashboard now reports action, completed/total
  counts, and the repository path while the existing bounded worker pool runs;
  the terminal result still drives authoritative result storage and refresh.
  `TestRepositoryBatchProgressCommandPreservesEventStream` covers delivery
  from progress through the terminal result, while
  `TestRepositoryBatchOperationEmitsBoundedProgressBeforeResults` exercises a
  real command stream through a failed repository operation; successful
  disposable-repository, cancellation, and native acceptance remain open.

- Added `[K] cancel` to the repositories dashboard. It cancels the active
  batch context used by the bounded worker pool, allowing queued work to
  resolve as cancelled and preserving the authoritative terminal result path.
  `TestRepositoryBatchCancelUsesActiveContext` verifies the UI control invokes
  the active cancellation function; successful remote cancellation and native
  acceptance remain open.

- At revision `1e1edc9`, focused app and multirepo tests passed under race
  detection, and `scripts/parity-check.sh` passed the multi-repository lane.
  The implementation now includes explicit fetch/pull controls, ff-only pull,
  bounded queued/running progress, failed-subset retry, and cancellation.
  Real 50-repository remote cancellation and native/manual acceptance remain
  open.

- Added `TestBatchFetchFiftyDisposableRepositoriesIsBoundedAndFailureIsolated`,
  which creates 50 disposable repositories with local bare remotes, mixes clean,
  dirty, and local-only histories, injects missing-remote failures, runs real
  typed fetches through the four-worker batch executor, and verifies successful,
  failed, and cancelled outcomes without exceeding the worker bound. The test
  is included in `scripts/parity-check.sh` and passed at the current revision.
  Native cancellation and watcher-responsiveness evidence remain open.

- Revision `50a36b2` passed the focused normal and race tests, `go vet`,
  `git diff --check`, and CI run `35777279686` on Ubuntu 24.04, macOS 15, and
  Windows 2025. The CI run also passed the repository security and performance
  gates. Native terminal cancellation and watcher-responsiveness evidence are
  still required before closing the task.

- At revision `df1472b`, `GOCACHE=/tmp/gitwatch-parity-current-cache
  GOMODCACHE=/tmp/gitwatch-modcache ./scripts/parity-check.sh` passed the real
  50-repository batch-fetch lane, watcher and polling lanes, and the complete
  integration/package coverage. Native cancellation and hosted cross-platform
  acceptance remain open.

## Progress evidence (2026-09-25)

- At revision `172d00a`, race-enabled focused tests passed:
  `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache go test
  -race ./internal/app ./internal/integration -run
  'TestRepositoryBatch|TestBatchFetchFiftyDisposableRepositoriesIsBoundedAndFailureIsolated'
  -count=1`. This includes dashboard progress/cancel coverage and the real
  50-repository disposable-local-remote test for worker bounds and failure
  isolation.
- At the same revision,
  `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  ./scripts/parity-check.sh` passed. Its batch-fetch, watcher/manager,
  provider, multirepo, and integration lanes all succeeded. The full
  `make check` at `91424f1` passed formatting, pinned lint, tests, race, vet,
  security fuzz, and performance checks; that revision contains the same Task
  174 source as `172d00a` (the later commit only changed task documentation).
- Hosted Actions run `36085163506` passed for `91424f1`, including Ubuntu,
  macOS, and Windows test/build/runtime jobs and scripted PTY/large-status
  acceptance. This does not measure concurrent watcher responsiveness during
  an active batch or substitute for native/manual cancellation. Those gates
  remain open, so Task 174 remains active.
