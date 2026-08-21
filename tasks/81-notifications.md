# Task 81: Add operation notifications and attention model

Status: Complete

Progress: Added bounded, thread-safe notifications with severity/kind metadata, dismissible attention badges, quiet-mode suppression, and newest-first ordering helpers. Bubble Tea now routes core operation, conflict snapshot, commit-hook failure, stale-remote, plugin-state failure, and remote success/failure events into notifications, renders sanitized toast notices, shows a global attention badge, and supports `Ctrl-N` dismissal of the newest active notification; configured full/reduced/off motion is carried into the app model for presentation policy.

## Objective
Create non-intrusive toast/activity notifications for completed jobs, conflicts, failed hooks, failed pushes, stale remote state, and plugin errors. Add attention badges to relevant views. Respect reduced-motion and quiet settings.

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

Completion summary: Notifications are bounded and thread-safe, carry severity/kind metadata, suppress attention in quiet mode, preserve sanitized toast text, and expose newest-first active/dismissal behavior. The app emits notifications for completed/failed operations, conflicts, hook failures, stale remotes, plugin state failures, and remote outcomes; every workspace renders the attention badge and `Ctrl-N` dismisses the newest active item. Configuration supports quiet notifications and full/reduced/off motion. Model, config, app-routing, badge, dismissal, and race-enabled repository tests pass all quality gates.
