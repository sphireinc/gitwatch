# Task 87: Publish plugin SDK, examples and compatibility tests

Status: Complete

Progress: Added dependency-free public `pkg/plugin` manifest/message SDK, checked-in API-1 wire fixtures with fixture-driven compatibility tests, buildable status/command/panel/widget examples with API-1 manifests, and SDK documentation. The v1 compatibility matrix is explicitly API-1 host/plugin interoperability; future API versions must add a new fixture directory and negotiation tests.

## Objective
Create a minimal SDK/protocol package, example status widget, example command, example panel, manifest documentation, capability guide, and compatibility test harness. Plugins should be buildable without importing gitwatch internal packages.

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

## Completion summary
- Implementation: stable dependency-free manifest, message, capability, lifecycle, and handshake contract in `pkg/plugin`; command, panel, widget, and status examples.
- Compatibility: API-1 handshake request/response and status-widget fixtures under `pkg/plugin/testdata/v1/`, validated through the public decoder.
- Documentation: `docs/plugin-sdk.md` records build commands, capabilities, manifest fields, wire messages, and the supported matrix.
- Deferred follow-up: a future API version will require its own fixture directory and negotiation tests; no cross-version contract exists in v1.
