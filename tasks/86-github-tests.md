# Task 86: Build GitHub provider contract tests

Status: In progress

Progress: Provider parsing, cache behavior, CLI-token failure handling, secret-redaction tests, recorded success/error/rate-limit fixtures, offline degradation, and context-cancellation classification are present without developer credentials. Full provider contract coverage remains limited to the current read-only PR/check/review surface.

## Objective
Use recorded/fake provider responses for PR/check/auth/rate-limit/error states. Ensure no test requires a developer token. Add redaction tests and graceful degradation when GitHub is offline or unavailable.

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
