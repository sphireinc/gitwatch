# Task 53: Create interactive hunk staging TUI

## Objective
Add a patch-mode diff view with gutter selection, current hunk emphasis, selected-line indicators, hunk counters, context expansion, mouse selection, keyboard navigation, help hints, and animation that never obscures patch content.

Progress: The app now exposes an asynchronous Hunk workspace from the loaded diff with selected-line markers, hunk counters, keyboard selection/all/invert controls, partial stage/unstage actions, guarded discard, mouse changed-line selection, and keyboard file/hunk navigation. Context expansion reloads the authoritative Git diff asynchronously at 3, 8, or 20 lines. A dedicated viewport/scroll treatment and final animation polish remain.

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

**Status:** In progress — patch view renders hunk context, line selection markers, selection counters, and keyboard hints; full Bubble Tea viewport/mouse routing remains.
