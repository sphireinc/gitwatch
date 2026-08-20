# Task 63: Render animated Git log graph

Status: Complete

Progress: Added a deterministic render-neutral lane graph, merge-parent lanes, ref decoration classification (HEAD/branches/tags), and case-insensitive history filtering. Bubble Tea renders selectable graph rows, appends paginated data, supports interactive `/` search, and preserves selection across refresh/filter changes; history text is sanitized before terminal rendering. Selected-commit details are covered by Task 64, and full/reduced/off motion now drives a non-blocking selected-lane pulse that never changes graph layout or hides patch content.

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
Implementation notes:

- Graph lanes and decorations remain deterministic and render-neutral; animation is limited to an alternate selected commit marker.
- The pulse is advanced from the existing non-blocking tick path and is disabled for `motion=off`; reduced motion uses the same bounded single-marker update without changing content geometry.
- Tests cover graph correctness, filtering, sanitization, selection preservation, and pulse rendering.
