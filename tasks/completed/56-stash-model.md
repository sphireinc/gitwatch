# Task 56: Build stash data and command layer

## Objective
Parse stash list with stable refs, timestamps, messages, branch context, and object IDs. Add typed operations for create, apply, pop, drop, branch-from-stash, and inspect. Detect dirty/conflicting states before mutation.

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

**Status:** Complete

## Completion summary

- Implemented typed stash list/create/apply/pop/drop/branch/show operations with ref and name validation.
- Added explicit include-untracked creation options and checked apply/pop/branch operations that reject dirty worktrees before mutation.
- Integrated branch-from-stash UI name entry with status, stash, and branch refresh after success.
- Added application tests for stash action routing, confirmation, conflict feedback, and branch-from-stash routing.
- Added real temporary-repository coverage for dirty-state rejection, apply, drop, pop, and branch-from-stash.
- Verified with `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
