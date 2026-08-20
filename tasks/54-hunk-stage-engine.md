# Task 54: Implement partial stage and unstage operations

## Objective
Generate/apply patches safely using Git plumbing (`git apply --cached` / reverse operations as appropriate). Never hand-edit the index. Validate patch applicability before mutation, handle CRLF/binary/rename edge cases, refresh after success, and preserve selection on recoverable failure.

Progress: Typed partial patch operations now perform `git apply --check` before applying, with cached stage, cached reverse unstage, and working-tree reverse discard paths wired to the Hunk workspace. CRLF/binary/rename integration coverage remains.

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

**Status:** In progress — selection-aware patch generation and typed stdin application for cached/reverse operations now validate with `git apply --check` before mutation; CRLF/binary/rename edge handling and refresh integration remain.
