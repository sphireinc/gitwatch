# Task 77: Build bounded multi-repository status engine

Status: Complete

Progress: Added a bounded worker-pool status engine with context cancellation, cached inactive-repository refreshes, injectable discovery/snapshot/stash functions, a 15-second per-repository resource budget, adaptive group-aware refresh intervals, and concurrency tests. The engine now feeds the asynchronous `v` repository dashboard with stash/remote counts, structured inactive-skip reasons, refresh duration/timestamps, and auxiliary warning summaries.

## Objective
Collect branch/dirty/ahead-behind/conflict/stash summaries across many repositories with bounded worker pools, adaptive refresh, inactive-repo throttling, and resource budgets. Do not instantiate the full single-repo watcher stack for every repository.

## Required implementation
- Produce production-quality implementation, not a prototype.
- Integrate with the existing Bubble Tea message/update architecture and typed Git runner.
- Keep the UI responsive; blocking filesystem, Git, network, and provider work must not run in the render/update hot path.
- Add keyboard and mouse behavior where the task introduces an interactive surface.
- Add structured errors/activity events and refresh affected repository state after mutations.
- Add focused unit/integration tests for success, failure, cancellation, and relevant edge cases.
- Update help/keymap/config/docs when this task adds user-visible behavior.

## Acceptance criteria
- Feature works on macOS, Linux, and Windows unless the task explicitly documents a platform limitation.
- No shell-string interpolation is introduced for Git/process execution.
- User-controlled terminal text is sanitized against control/escape injection.
- Existing v1 status/stage/diff workflows remain functional.
- `go test ./...`, static analysis, and formatting checks pass.
- The task is not complete until automated tests cover its primary behavior.

## Completion artifact
Status: Complete

Completion summary: The multi-repository engine uses bounded workers, per-repository timeout budgets, cancellation, cached inactive-repository refreshes, group-aware adaptive intervals, and injectable typed Git discovery/snapshot/summary sources. Dashboard rows preserve branch/dirty/divergence/conflict/stash/remote data and now surface structured auxiliary warnings and refresh metadata without blocking the TUI. Engine, dashboard, benchmark, cancellation, budget, and warning tests pass all repository quality gates.
