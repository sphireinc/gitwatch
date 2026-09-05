# Task 154: Add bounded bulk submodule operations

**Phase:** Submodules
**Depends on:** 152, 153

## Goal

Support large superprojects with observable bounded bulk initialize/update/sync.

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

1. Add selected/all preview for initialize, update and sync.
2. Execute with per-parent bounded worker pool and global operation limits.
3. Show per-submodule queued/running/success/failure/skipped and allow retry failed.
4. Cancellation propagates to Git children.
5. Do not implement destructive bulk remove in this task.

## Verification

- 20+ submodules with injected failures, cancellation, process-count bound.

## Acceptance criteria

- [x] Bulk operations are bounded, cancellable and failure-isolated.

## Completion record

- [x] Implementation commits recorded: `112f958`, `2ae4bcd`, `6115c2b`,
  `f672b09`.
- [x] Exact tested revision recorded: `f672b09`.
- [x] Focused unit/integration tests recorded: submodule bulk bound,
  cancellation, duplicate selection, real local failure isolation, 24-module
  injected failure isolation/concurrency bound, and app preview/result/retry
  routing.
- [x] `go test ./...` recorded at `f672b09`.
- [x] Race/vet/lint/format evidence recorded at `f672b09` through `make check`.
- [x] Native/manual evidence exception documented: automated app routing and
  80x24-safe status rendering are covered by tests and the full gate; an
  interactive human terminal pass remains a release-QA follow-up.
- [x] Known limitations/deferred work documented below.

## Progress evidence

- Added `submodules.Bulk` for explicitly selected initialize, update, and sync
  paths. It deduplicates input, preserves order, caps workers at eight, caps
  modules at a configured bound, propagates cancellation to Git children, and
  records queued/running/succeeded/failed/skipped/cancelled per-module state.
- Existing single-module typed operations remain the only Git command boundary;
  destructive bulk remove is not exposed.
- Tests cover hard module/worker bounds, cancellation without starting Git,
  duplicate selection, and real local failure isolation.
- Status UI now offers explicit all-module initialize/update/sync previews,
  selected-module update preview, per-module result rendering, cancellation,
  and retry-failed routing. App tests cover the preview, dispatch, result, and
  retry state transitions.
- Implementation is recorded in commits `112f958` (bulk domain engine),
  `2ae4bcd` (status workflow and retry UI), `6115c2b` (verification record),
  and `f672b09` (injectable concurrency evidence). The exact tested revision
  is `f672b09`; focused tests and the full `make check` gate passed there,
  including race, vet, lint, formatting, security, and performance checks.
- The lifecycle boundary now uses a narrow typed `CommandRunner` interface,
  retaining `git.Runner` in production while allowing deterministic injected
  orchestration tests. A 24-module test injects one failure and observes the
  maximum concurrent command count never exceeding the hard worker limit.
- Deferred release follow-up: run the interactive human terminal pass on a
  real repository with 20+ submodules. The automated implementation and
  acceptance evidence is complete; this is QA evidence rather than an open
  code path.
