# Task 183: Set and enforce large-scale performance budgets

**Phase:** Hardening
**Depends on:** 124, 125, 154, 174, 178, 180

## Goal

Prove the expanded workbench remains an always-on htop-like tool rather than becoming sluggish as features accumulate.

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

1. Define budgets for status refresh at 1k/10k/50k changed paths, Bubble Tea update/render latency, dashboards at 10/50/100 repos, history/reflog pages, provider refresh and submodule summary.
2. Add local benchmarks without telemetry.
3. Virtualize large lists: status, history, reflog, PRs, operation timeline, repositories, tags, remote branches.
4. Prioritize local status refresh over provider/network/plugin/background operations.
5. Profile allocations, goroutines and child-process count during event storms, checkout/rebase and batch operations.
6. Add regression thresholds with CI variance tolerance rather than brittle exact timings.
7. Document hardware-independent qualitative gates: no unbounded growth, no UI-blocking process calls, bounded queues everywhere.

## Verification

- Benchmarks/stress, goroutine/process leak tests, 50k changed-path fixture, 100-repo registry fixture.

## Acceptance criteria

- [ ] No unbounded list/process/goroutine behavior.
- [ ] Live status remains responsive under documented scale scenarios.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Current implementation evidence

- The bounded status viewport path has deterministic 14,953-entry flat/tree
  coverage, a comparative benchmark, and an allocation regression test with a
  1,000-allocation threshold for viewport row-height measurement.
- The same path now has scale benchmarks and allocation regression coverage at
  1,000, 10,000, and 50,000 changed paths, with the viewport positioned near
  the end of each list.
- `scripts/performance-check.sh` runs the status benchmark alongside the
  existing patch/history/registry workload checks.
- Full multi-repository, provider, history, 50k-path, goroutine/process, and
  native responsiveness evidence remains required; this task stays open.

## Progress evidence (2026-09-22)

- Added `TestEngineRefreshKeepsHundredRepositoriesWithinWorkerBound`, which
  exercises 100 repository refreshes with cooperative delay, verifies all
  results remain repository-scoped and input-ordered, and asserts peak
  concurrency does not exceed the configured eight workers.
- Fixed `registry.Engine.Refresh` to preserve its documented input order while
  retaining bounded worker concurrency and cancellation behavior. The focused
  registry suite passed at commit `6fee975`; the full `make check` gate passed
  on macOS arm64 with pinned lint, tests, race tests, vet, security, and
  performance checks.
- The task remains open for 50k-path end-to-end responsiveness, process/
  goroutine leak evidence, provider/history scale coverage, and native terminal
  acceptance.

- At revision `4d64f7d`, `GOCACHE=/tmp/gitwatch-performance-cache
  ./scripts/performance-check.sh` passed patch, history, registry, 14,953-row
  status, 50,000-row status-scaling, and 50-repository palette benchmarks.
  The 50,000-row bounded status sample remained approximately 115 microseconds,
  46 KB, and 299 allocations per iteration. Leak/process and native evidence
  remain open.

- Added an 80x24/`NO_COLOR`/motion-mode regression for the 14,953-entry status
  presentation path. The test passes normally, under race detection, and with
  vet; native responsiveness and process/goroutine leak evidence remain open.

- Added `TestEngineCancelledRefreshesSettleWithoutGoroutineGrowth`, which runs
  twenty cancelled 64-repository refreshes, accepts only bounded partial
  cancellation results, verifies every result remains repository-scoped, and
  asserts goroutines return within two of baseline. The test passed 20 normal
  repetitions, 5 race repetitions, and vet at the current revision.

- At revision `53f5131`, `./scripts/performance-check.sh` passed with isolated
  caches on macOS arm64 / Apple M1 Pro. The bounded status benchmark measured
  approximately 301 microseconds and 46 KB versus 33.1 milliseconds and 1.43
  MB for the full-scan baseline; the 50,000-entry scale sample remained at
  approximately 214 microseconds, 46 KB, and 299 allocations. Leak/process,
  provider/history scale, and native responsiveness evidence remain open.
