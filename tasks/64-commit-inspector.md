# Task 64: Build commit inspector and historical diff views

Status: In progress

Progress: Added parent-relative commit inspection primitives returning changed paths, add/delete/binary stats, full patches, and path-safe filtering, plus commit metadata summary. Full inspector UI, SHA copy, ref jumps, and merge-parent controls remain.

## Objective
Selecting a commit should show metadata, parents, refs, changed-file stats, full patch, and parent-relative diff. Add copy-SHA, inspect parent, jump-to-ref, and path filtering. Support merge commits explicitly.

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
