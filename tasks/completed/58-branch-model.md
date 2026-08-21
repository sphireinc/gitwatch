# Task 58: Build branch/ref model

## Objective
Model local branches, remote-tracking branches, current HEAD, detached HEAD, upstream, ahead/behind, last commit, merged state, and worktree occupancy. Use for-each-ref/rev-list style machine-readable Git output rather than scraping human output.

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

Completion summary: Branch parsing and listing use typed `for-each-ref`/`rev-list` argv with NUL-delimited fields, preserving current/detached/remote/upstream identity, numeric divergence, last-commit timestamp/subject, merged state, and worktree occupancy. Checkout/create and other branch operations remain typed and validate names. Branch parser and view tests cover metadata rendering; repository-wide quality gates pass.
