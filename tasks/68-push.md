# Task 68: Implement push workflows

Status: In progress

Progress: Added explicit branch push and opt-in force-with-lease command primitives with validated remote/ref arguments. The remotes workspace now asynchronously pushes the selected remote/current branch and refreshes after completion; ref movement preview, set-upstream/tag workflows, and strong force-push confirmation remain.

## Objective
Add push current branch, set-upstream push, tag push where explicitly selected, and force-with-lease behind strong confirmation. Never offer raw --force as the default. Preview local/remote ref movement before destructive/non-fast-forward operations.

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
