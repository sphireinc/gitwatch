# Task 45: Define post-v1 navigation and feature architecture

## Objective
Introduce a top-level workspace/view model for Status, Commit, Stashes, Branches, Log, Remotes, GitHub, Plugins, and Repositories. Define shared command/message contracts, breadcrumbs, modal ownership, background job state, cancellation, and refresh semantics. Add architecture tests preventing feature modules from bypassing the Git runner.

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

**Status:** In progress — typed workspace views, breadcrumbs, modal ownership, cancellable job state, independent snapshots, and job lifecycle tests are implemented. Bubble Tea now routes asynchronously into the branch and stash views with keyboard navigation and return-to-status behavior; architecture-boundary coverage and the remaining feature surfaces remain.
