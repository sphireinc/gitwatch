# Task 74: Implement secure plugin runtime

Status: Complete

Progress: Added manifest-validated direct-argv plugin execution with context cancellation, bounded stdout/stderr capture, exit-code/duration reporting, output-limit tests, a bounded capability handshake over the public newline-delimited protocol, strict response-capability subset validation, and bounded restart supervision tests. Runtime execution now classifies timeout/cancellation failures; byte-stream RPC and manager lifecycle controls remain outside this task’s current buffered command boundary and in Task 75 where UI lifecycle is presented.

## Objective
Implement discovery, enable/disable, startup handshake, timeouts, crash isolation, capability grants, structured logging, and version mismatch handling. Plugins must be opt-in and unable to execute arbitrary gitwatch internals outside the public contract.

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

Completion summary: Secure plugin execution validates manifests and grants before process start, launches with direct argv, bounds stdout/stderr, reports exit code and duration, classifies cancellation/timeouts, validates handshake protocol/API/capability subsets, and retries only within an explicit bounded supervision policy. Hostile output, denial, malformed handshake, capability escalation, and restart behavior are tested. The runtime intentionally retains a buffered request/response boundary; streaming transport and manager UI are separate follow-ups. Full test, race, vet, formatting, and diff-check gates pass.
