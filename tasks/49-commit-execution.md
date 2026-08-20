# Task 49: Implement safe commit execution

## Objective
Execute normal commits from the composer. Show exact staged scope, run hooks normally, surface hook output, preserve draft text on failure, refresh repository state on success, and record the resulting commit SHA. Require explicit confirmation when committing with an empty message or unusual state.

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

**Status:** In progress — safe stdin-backed commit execution supports normal/amend/no-edit/signoff/signing/author options, captures hook output through the typed runner, and returns the resulting SHA. The composer now executes asynchronously with `Ctrl-S`, preserves drafts on failure, and triggers an authoritative status refresh after success; confirmation, hook-output presentation, and full integration coverage remain.
