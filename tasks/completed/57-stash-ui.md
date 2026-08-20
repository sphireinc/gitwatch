# Task 57: Create stash manager

## Objective
Build a dedicated stash view with list, metadata/details, patch preview, create-stash dialog, include-untracked option, apply/pop actions, drop confirmation, and conflict reporting. Preserve stash selection after refresh when possible.

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

- Added asynchronous Stashes workspace loading with filtering, empty state, metadata, and patch preview.
- Added mouse row selection and keyboard-equivalent create, apply, pop, and drop workflows.
- Added create message entry with an explicit include-untracked toggle, enabled by default.
- Added confirmation for mutating actions, conflict/error feedback, authoritative status/stash refresh, and selection preservation.
- Added application tests for routing, confirmation, conflict feedback, and mouse behavior.
- Added a real temporary-repository integration scenario covering create, apply, drop, and pop.
- Verified with `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
