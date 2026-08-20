# Task 66: Build remote and tracking model

Status: Complete

Progress: Added typed remote discovery, URL redaction, default-remote selection, reachability/error fields, last-fetch reflog timestamps, and tracking-symbol parsing. Branch discovery now requests Git's exact upstream tracking symbols and populates ahead/behind counts; the workspace loads a redacted remote dashboard asynchronously with branch divergence and stale-fetch state.

Completion summary: Remote discovery uses typed Git arguments and preserves redacted fetch/push URLs, reachability errors, default selection, and last-fetch reflog timestamps. Local branch tracking uses Git's `%(upstream:trackshort)` output for exact ahead/behind counts, and the asynchronous Remotes workspace exposes those values with stale-fetch state. Parser, redaction, tracking, dashboard, integration, race, and static-analysis coverage pass. Workflow mutation details remain in Tasks 67–69.

## Objective
Discover remotes, fetch/push URLs with credentials redacted, tracking relationships, divergence, last fetch state, and default remote. Never display embedded secrets/tokens. Add remote reachability/error state.

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
