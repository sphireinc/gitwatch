# Task 114 — Close a diff when its status path disappears

## Status

Complete.

## Objective

Ensure the right-side diff pane cannot continue displaying a worktree file
after an authoritative refresh shows that the file is no longer present in the
status list, such as after the file is committed externally.

## Acceptance criteria

- Applying a snapshot that no longer contains the currently inspected path
  closes the diff pane.
- Pending asynchronous diff results for the removed path are invalidated and
  cannot repopulate the pane.
- The refreshed status-file list remains authoritative and usable.
- A path that remains in the status snapshot does not have its diff closed just
  because another file changed.
- Add regression coverage for the disappearing-path scenario.
- Run formatting, linting, tests, race tests, vet, diff, security, and
  performance checks.
- Move this task to `tasks/completed/` after implementation.

## Implementation summary

Snapshot application now checks whether the active diff path remains in the
authoritative status entries. If it has disappeared, the diff is closed and its
request generation is advanced so late results are rejected.
