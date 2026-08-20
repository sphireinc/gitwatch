# Task 65: Add history navigation actions

Status: In progress

Progress: Added explicit-target checkout, branch-at-commit, tag listing, SHA-confirmed revert primitives, and option-like target rejection. The History workspace now requires explicit `x` then `y` confirmation naming the selected commit before detached checkout, and refreshes status after success; branch-at-commit, tag navigation, revert UI, and end-to-end mutation coverage remain.

## Objective
Implement checkout/switch from refs, create branch at commit, tag navigation, and guarded revert. Keep reset/rebase/cherry-pick out unless separately implemented with explicit workflows. All history mutations must identify target SHA/ref before confirmation.

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
