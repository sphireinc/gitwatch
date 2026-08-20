# Task 63: Render animated Git log graph

Status: In progress

Progress: Added a deterministic render-neutral lane graph, merge-parent lanes, ref decoration classification (HEAD/branches/tags), and case-insensitive history filtering. Bubble Tea now renders selectable graph rows, appends paginated data, and preserves selection across refresh/filter changes; search input, selected-commit details, and reduced-motion animation remain.

## Objective
Create a high-quality terminal commit graph with lanes, merges, branches/tags/HEAD decorations, scrolling, filtering, search, selected-commit details, and adaptive layout. Graph correctness takes priority over animation.

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
