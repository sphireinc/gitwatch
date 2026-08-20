# Task 55: Implement guarded partial discard

Status: Complete

## Objective
Allow selected working-tree hunks/lines to be discarded only behind a high-friction confirmation showing file and hunk scope. Prefer Git-supported patch application semantics. Add tests proving unrelated lines are untouched.

Progress: Selected working-tree hunks require the literal `discard` confirmation and use Git reverse patch application after check validation; failure preserves the selection and success refreshes authoritative status. A real scenario verifies discarding only the remaining hunk while preserving the staged change and unrelated lines.

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
Implementation notes:

- Discard is available only for working-tree patches and requires typing the exact word `discard` after the selected file/hunk scope is visible.
- Reverse application is checked with Git before mutation; successful operations trigger the authoritative status refresh and failures preserve selection/state.
- Real temporary-repository coverage verifies separated hunk discard and preservation of staged/unrelated content. Binary and rename patches are intentionally not discardable through the line-selection UI.

**Status:** In progress — partial discard has a typed high-friction confirmation requiring exact path/hunk/line scope and a distinct confirmation word; selected reverse-patch execution and unrelated-line integration tests remain.
