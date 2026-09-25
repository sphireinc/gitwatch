# Task 175: Create repository health model and dashboard

**Phase:** Multi-repository differentiation
**Depends on:** 125, 151, 155, 172

## Goal

Turn gitwatch’s htop identity into a concrete repository-health surface.

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

1. Create `internal/health` with cheap local metrics: clean/dirty/conflict, ahead/behind, unpushed count, stash count, active sequencer, worktree count, submodule health, last fetch age and signing/config warnings.
2. Provider enrichments are optional cached fields: PR state, checks, review attention.
3. Remote latency is recorded from actual network operations or an explicit probe; never ping remotes every status refresh.
4. Compute semantic attention severity rather than a fake numeric “health score”.
5. Expose health in single-repo header/details and multi-repo dashboard.
6. Every non-live metric records freshness timestamp/source so stale provider/network data is obvious.
7. Local status-derived metrics update from the authoritative snapshot.

## Verification

- Freshness behavior, provider disabled, offline mode, large registry.

## Acceptance criteria

- [x] Dashboard remains valuable offline.
- [x] Live local health is status-derived; cached external health is labeled stale/fresh.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added `internal/health` with semantic local severity (`healthy`, `info`,
  `warning`, `critical`) and explicit metrics for dirty/conflict state,
  ahead/behind/unpushed counts, stashes, worktrees, active operations,
  attention details, freshness timestamp, and source.
- Health derives from the authoritative repository snapshot and remains useful
  offline; auxiliary warnings raise attention without replacing local status.
- Integrated the summary into registry rows and the repositories dashboard,
  including visible severity labels.
- Added focused tests for local freshness, conflict severity, offline healthy
  state, and registry/dashboard integration.
- Remaining: worktree/submodule/config-signing enrichments, cached provider
  freshness labels, actual remote-latency timestamps, single-repository detail
  presentation, and native/manual acceptance evidence.

- Revision `36c845a` populates `Summary.SubmoduleIssues` from the authoritative
  porcelain-v2 submodule field, treats `S...` as clean, and raises warning
  severity with a `submodules` attention marker for changed/modified/untracked
  submodule content. Registry/dashboard tests and race/vet checks pass. The
  remaining worktree/config-signing/provider freshness/detail and native
  acceptance evidence remain open.

- Revision `ac6fd97` adds bounded linked-worktree counting through Git's
  porcelain worktree listing, carries the count through registry health and
  dashboard rows, and renders `worktrees:N` in the repository workspace.
  Repeated normal/race/vet/UI checks pass; config-signing, provider freshness,
  single-repository detail, and native acceptance remain open.

- Revision `612fdba` carries non-secret `commit.gpgsign`/`gpg.format` metadata
  into health and renders configured signing in the repository dashboard.
  Missing or unknown formats become explicit warning attention without reading
  keys or credentials; repeated normal/race/vet checks pass. Provider freshness,
  single-repository detail, and native acceptance remain open.

- Added a single-repository Status health row derived from the authoritative
  snapshot, including severity, source, observation time, attention details,
  and non-secret signing configuration warnings. The responsive layout and
  mouse coordinates were updated for the additional row; focused normal,
  race, vet, and layout checks pass. Provider freshness and native acceptance
  remain open.

- At revision `6b7a4d0`, the repository-wide `make check` passed after the
  Status layout change: pinned lint reported 0 issues, full normal and race
  tests, vet, security fuzz, diff checks, and performance benchmarks passed.

## Progress evidence (2026-09-22)

- GitHub cached collection results now preserve stale-cache state through the asynchronous app message and render an explicit `provider cache: fresh|stale` label in the GitHub workspace. This keeps optional provider data visibly separate from authoritative local Git health.
- Added UI regression coverage for both fresh and stale provider-cache labels. Focused app and GitHub-view tests pass; the full repository gate remains required before commit handoff.
- Remaining: actual remote-latency timestamps, richer single-repository detail presentation, native/manual acceptance, and hosted cross-platform evidence.

## Progress evidence (2026-09-24)

- Revision `f0e289793f699efe54bf9b2b8a7c81edaec7e999` records measured auto-fetch latency for both already-registered and newly discovered repositories; the previous first-seen path omitted the duration.
- The repository dashboard prioritizes remote-fetch status and elapsed milliseconds within an 80-column row, and shows local-health source/observation time, fetch completion time, and explicit cached-provider `(fresh)`/`(stale)` labels in its detail line.
- Focused regression coverage: `TestAutoFetchFinishedRecordsMeasuredLatencyForRegisteredAndNewRepos`, `TestViewShowsMeasuredAutoFetchLatency`, `TestViewShowsCachedCIAttentionAndStaleness`, and `TestComputeOfflineCleanRepositoryRemainsHealthy`.
- Full validation passed on Go `go1.27.0 darwin/arm64` (macOS, Apple M1 Pro): `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache make check` (format, pinned lint, full unit and race suites, vet, diff, security fuzz, performance); the focused race-test selection and `go build -o /tmp/gitwatch-task174.iQFJ30/gitwatch-task175-verified ./cmd/gitwatch` also passed.
- Scripted pseudo-terminal run at 80x24 showed a healthy local-health row and `remote-fetch:fetched latency:1250ms`; this is automated local PTY evidence, not native/operator acceptance. Offline behavior is separately covered by `TestComputeOfflineCleanRepositoryRemainsHealthy`. Full release-pinned Go 1.25.10 and hosted cross-platform evidence were not established by this run. Further single-repository detail and native/manual acceptance remain open.

## Progress evidence (2026-09-25)

- User-facing behavior is now described in the README, provider guide, advanced multi-repository workflow guide, and Unreleased changelog: local status remains authoritative/offline-capable, provider freshness is explicit, and fetch completion/latency is reported without probing on each status refresh.
- Configuration docs now enumerate defaults and nested settings, environment variables, CLI overrides, migration behavior, and the custom-command prompt fields. The published v3 JSON Schema now includes typed prompt definitions; a schema regression test verifies that surface.
- Documentation/schema checks and `make check` pass on Go `go1.27.0 darwin/arm64` (format, pinned lint, full unit/race suites, vet, security fuzz, performance). The Windows app test binary cross-compiles; hosted cross-platform and native/manual interaction acceptance remain open.
