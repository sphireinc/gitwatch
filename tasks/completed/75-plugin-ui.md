# Task 75: Build plugin manager and extension surfaces

Status: Complete

Progress: Added bounded manifest discovery with symlink skipping, immutable enable-state updates, and a visually distinct plugin list/detail view showing health, capabilities, explicit permission state, and declared command/panel/status-widget extension counts. The `E` Plugins workspace loads/reloads discovered manifests asynchronously, Space toggles the selected plugin with explicit status feedback, and enable state persists in a private atomic JSON file asynchronously.

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
Status: Complete

Completion summary: The plugin manager provides an asynchronous discovery/reload workspace, immutable enable/disable updates with private atomic persistence, keyboard navigation, health/error presentation, capability and permission-state display, and visually identifiable extension-surface summaries for commands, panels, and status widgets. Plugin UI is descriptive only; execution remains isolated behind the runtime contract. Plugin manager/view tests plus repository-wide quality gates pass. Fine-grained per-capability grants and actual third-party panel rendering remain intentionally deferred to a future protocol revision.
