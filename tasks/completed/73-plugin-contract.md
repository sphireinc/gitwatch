# Task 73: Design versioned plugin system

Status: In progress

Progress: Added versioned manifest validation, capability declarations, host capability negotiation, an out-of-process contract document, newline-delimited handshake request/response schema, stable lifecycle/event/command/panel/status-widget/configuration types, and additive compatibility guarantees in the public SDK. The runtime contract includes explicit capability grants and bounded restart supervision; manager UI remains in Task 75.

## Objective
Define plugin manifest, API version negotiation, capabilities, lifecycle, events, commands, panels, status widgets, configuration schema, and failure isolation. Decide an RPC/out-of-process boundary so third-party plugins cannot corrupt the TUI process. Document compatibility guarantees.

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

Completion summary: The v1 plugin contract is versioned at API 1, validates identifiers/capabilities/config-schema bounds, negotiates exact host grants, and defines newline-delimited messages plus lifecycle, event, command, panel, widget, and configuration schema types. Plugins remain out-of-process and cannot mutate the TUI directly. Public SDK wire/validation/negotiation tests and documentation pass the repository-wide quality gates. Runtime supervision is implemented in Task 74 and manager presentation remains Task 75.
