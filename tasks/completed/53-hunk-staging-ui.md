# Task 53: Create interactive hunk staging TUI

## Objective
Add a patch-mode diff view with gutter selection, current hunk emphasis, selected-line indicators, hunk counters, context expansion, mouse selection, keyboard navigation, help hints, and animation that never obscures patch content.

Progress: The app exposes an asynchronous Hunk workspace from the loaded diff with selected-line markers, hunk counters, keyboard selection/all/invert controls, partial stage/unstage actions, guarded discard, mouse changed-line selection, keyboard file/hunk navigation, and a bounded viewport that keeps the active line visible. Context expansion reloads the authoritative Git diff asynchronously at 3, 8, or 20 lines. Patch content remains static during operations so animation cannot obscure content or block input.

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

**Status:** Complete — the Hunk workspace is integrated with asynchronous Git loading, bounded rendering, keyboard and mouse selection/navigation, context expansion, partial stage/unstage, and guarded discard. Patch content is never replaced by animation or a blocking progress surface.

## Completion summary

- Added hunk/file navigation (`n`/`p`, `N`/`P`), changed-line mouse selection, and viewport scrolling for long patches.
- Added asynchronous context expansion (`c`) cycling through 3, 8, and 20 Git context lines.
- Kept Git operations off the update/render path and retained authoritative refresh after mutations.
- Added focused view/app tests plus repository-wide test, race, vet, and diff checks.
