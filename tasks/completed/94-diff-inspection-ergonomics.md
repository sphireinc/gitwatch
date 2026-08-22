# Task 94 — Improve staged/unstaged diff inspection ergonomics

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 18, 51–55, and 93

## Objective

Make selected-file diff inspection useful for large and busy changes while
preserving the existing right-pane and narrow-overlay model. The feature must
remain on-demand, cancellable, and read-only.

## Required work

- Provide explicit staged/unstaged mode switching when both forms exist.
- Add bounded diff scrolling with visible position/context indicators and
  keyboard/mouse wheel parity.
- Add diff search with case-sensitive/insensitive behavior documented and a
  clear no-match state; search must not block the UI or load an unbounded copy.
- Preserve binary, rename, copy, conflict, no-change, malformed-output, and
  stale asynchronous-result states.
- Keep headers, hunk markers, additions, deletions, context, and wrapped lines
  distinguishable in colorless and high-contrast themes.
- Define and enforce maximum diff bytes/lines, with an honest truncation notice
  and a safe way to request more only when the configured budget permits it.
- Ensure closing, switching files, refreshing, resizing, or quitting cancels
  or invalidates obsolete diff work.

## Non-goals

Do not edit files from the diff pane, implement an embedded Git diff engine,
add syntax highlighting that obscures Git markers, or silently load the entire
history of a file.

## Acceptance criteria

- A user can inspect both index and worktree versions, search, scroll, resize,
  and return to status without losing selection.
- Diff requests are bounded, cancellable, generation/request-scoped, and never
  render stale content for a different path or mode.
- Long lines wrap consistently with the configured panel width and preserve
  visible additions/deletions.
- Unit tests cover search, paging, budgets, cancellation, stale responses,
  binary/rename/error states, and mouse coordinates; integration tests verify
  real Git output.

## Verification and documentation

Add `docs/diff-view.md` with key bindings, budgets, and limitations. Run full
tests, race tests, vet, lint, benchmarks, and native keyboard/mouse evidence
on wide and narrow terminals.

**Status:** Complete — added bounded diff byte/line configuration, explicit staged/unstaged switching, cancellable request-scoped loading, viewport search, truncation notices, aligned panel rendering, documentation, and focused budget/search coverage.
