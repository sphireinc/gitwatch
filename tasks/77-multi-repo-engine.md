# Task 77: Build bounded multi-repository status engine

Status: In progress

Progress: Added a bounded worker-pool status engine with context cancellation, cached inactive-repository refreshes, injectable discovery/snapshot functions, and concurrency tests. Adaptive budgets, stash summaries, UI routing, and richer refresh policy remain.

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
Record implementation notes, key decisions, new commands/keybindings/configuration, tests added, and any deliberately deferred follow-ups in the task/PR completion summary.
