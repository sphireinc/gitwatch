# Task 54: Implement partial stage and unstage operations

## Objective
Generate/apply patches safely using Git plumbing (`git apply --cached` / reverse operations as appropriate). Never hand-edit the index. Validate patch applicability before mutation, handle CRLF/binary/rename edge cases, refresh after success, and preserve selection on recoverable failure.

Progress: Typed partial patch operations now perform `git apply --check` before applying, with cached stage, cached reverse unstage, and working-tree reverse discard paths wired to the Hunk workspace. A real temporary-repository scenario verifies separated-hunk staging and preserves the remaining unstaged hunk. CRLF text is normalized by the parser and delegated to Git apply; binary, rename, and copy files are explicitly refused by the line-selection engine so they remain available through whole-file diff actions.

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

**Status:** Complete — selection-aware patch generation and typed stdin application for cached/reverse operations validate with `git apply --check` before mutation; CRLF text, unsupported binary/rename/copy metadata, refresh behavior, and separated-hunk integration are covered by the implementation and tests. Binary/rename/copy changes are deliberately refused by the line-selection path rather than mutated partially.

## Completion summary

- Added `ErrUnsupportedPartialPatch` for binary, rename, and copy metadata when a selection touches the file.
- Preserved CRLF handling through parser normalization and Git's patch application.
- Added unit coverage for each refused metadata class and integration coverage for stage/discard refresh behavior.
- Whole-file diff actions remain the supported path for binary and metadata-only changes.
