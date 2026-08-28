# Task 116 — Coordinate lower-left context panes

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Tasks 105, 107, and 115

## Objective

Allow the lower-left portion of the status workspace to display one selected
context pane: commit tree, unpushed commits, or a read-only branch summary.

## Requirements

- Keep the modified-file list above the context pane and keep the right details
  or diff panel unchanged.
- Preserve the existing commit-tree separator at the top of the lower pane.
- Add responsive layout rectangles and independent scrolling/focus for each
  context pane.
- Render a bounded branch summary using loaded branch data, including current
  branch, upstream, ahead/behind, and branch rows; branch mutations remain in
  the existing full branches workspace.
- Define empty, loading, error, detached, unborn, and no-upstream states.
- Preserve status file selection, wrapping, mouse hit testing, resize behavior,
  and narrow-terminal minimum heights.

## Acceptance

Pane switching does not alter repository state, selection or right-panel sizing;
each pane is bounded and scrollable; existing commit-tree behavior remains
available; layout and UI regression tests pass.

**Status:** Complete

## Implementation summary

Added the switchable lower-left context region for commit tree, unpushed
commits, and read-only branch summary. The existing file list and right-side
details/diff panel remain intact; the lower region has shared separator,
responsive sizing, scrolling, mouse focus, and explicit empty/loading/error
states.
