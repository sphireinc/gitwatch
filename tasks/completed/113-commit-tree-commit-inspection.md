# Task 113 — Inspect a commit from the commit tree

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Tasks 105, 107, and 111

## Objective

Allow a user to click or keyboard-select a commit in the lower-left commit
tree, load that commit's file changes into the upper-left pane, and show the
selected file's commit diff in the right-side details pane.

## Interaction model

- Clicking a commit row focuses and selects that commit; it must not mutate Git
  state or change branches.
- Provide keyboard navigation equivalent to the existing tree scrolling and a
  clear activation key, preserving ordinary file-list behavior and `T` focus.
- Show the selected commit identity, subject, author/date, and a bounded list of
  changed files in the upper-left region.
- Preserve the existing click-to-open-diff behavior for files in that list.
- Selecting a file loads the selected commit's patch for that file in the right
  pane, including staged/unstaged semantics appropriate to a historical commit.
- Provide a clear way to return to worktree status without losing the optional
  commit-tree context.
- Keep the horizontal separator and pane sizing coherent while the upper-left
  region is showing commit files.

## Git/data requirements

Use machine-readable, NUL-delimited Git output wherever available. Use argv
execution only; never parse shell commands or human-formatted output when a
porcelain format exists. Add bounded commands for commit existence, commit
metadata, changed-path enumeration, and per-file patch loading. Preserve path
bytes as far as Go/Git APIs permit, including spaces, tabs, Unicode, quotes,
leading hyphens, renames, and unusual paths. Correctly represent added,
deleted, renamed, copied, binary, submodule, mode-only, and merge-commit files.

Historical commit inspection must be read-only, cancellable, generation/request
scoped, and independent of worktree status refresh. A missing, rewritten, or
unreachable commit must produce a bounded actionable error and leave current
worktree status usable.

## Tests and acceptance

Add real-repository integration tests for linear and merge commits, all changed
file kinds, renames, binary files, unusual paths, empty commits, missing
objects, cancellation, stale results, repository switching, and shutdown. Add
UI tests for click/keyboard selection, focus transitions, scrolling, file
selection, right-pane diff loading, return-to-status, resize, narrow layouts,
`NO_COLOR`, and preservation of existing stage/diff interactions.

The feature must never perform checkout, reset, revert, stage, commit, or any
other mutation. Document the interaction and record native mouse/keyboard and
resize evidence separately from automated checks.

**Status:** Complete

## Completion summary

The Status commit tree now supports keyboard and mouse commit selection,
abbreviated-hash resolution, asynchronous read-only historical inspection, and
loading changed files into the upper-left list. Selecting those files loads a
historical per-file diff in the right pane; `Esc` and `1` restore worktree
status. Historical requests are generation/request scoped and cancellable, and
authoritative worktree refreshes preserve active historical files. NUL-safe
numstat parsing preserves unusual path bytes. App/history integration and
colorless safety tests pass.
