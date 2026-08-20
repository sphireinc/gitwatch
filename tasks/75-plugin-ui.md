# Task 75: Build plugin manager and extension surfaces

Status: In progress

Progress: Added bounded manifest discovery with symlink skipping, immutable enable-state updates, and a visually distinct plugin list/detail view showing health and capabilities. The `E` Plugins workspace loads/reloads discovered manifests asynchronously; permission controls, enable/disable persistence, and extension rendering remain.

## Objective
Create plugin list/details/settings UI, permission/capability display, enable/disable/reload actions, error health, and extension points for commands, panels, row decorations, and status widgets. Make third-party UI visually identifiable.

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
