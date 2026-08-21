# Task 59: Create branch manager

## Objective
Build searchable/sortable local and remote branch views with checkout/switch, create, rename, upstream visibility, ahead/behind badges, last-commit age, merged indicators, and mouse/keyboard actions.

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

Completion summary: Branches now support filtered search by branch/upstream, cyclic sorting by name/ahead/behind/last-commit, current/remote/upstream/divergence/merged/last-commit/worktree metadata, keyboard selection and checkout, and mouse row selection. Bubble Tea routes branch search and sort without blocking, preserves selected branches across refreshes, and documents the new keymap. App and branch-view tests cover filtering, sorting, mouse selection, and metadata rendering; repository-wide quality gates pass.
