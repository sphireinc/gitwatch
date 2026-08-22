# Task 99 — Multi-repository resilience, persistence, and workspace switching

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 76–79, 95, 97, and 98

## Objective

Make the multi-repository dashboard reliable for real project collections,
including missing paths, nested repositories, linked worktrees, slow/failed
repositories, concurrent edits, and persisted favorites/groups.

## Required work

- Define discovery roots, recursion boundaries, symlink policy, ignore rules,
  deduplication, and maximum repository counts in configuration.
- Persist only intended private metadata (favorites, groups, last selection,
  optional display labels) atomically with restrictive permissions.
- Keep one repository's missing Git, invalid config, lock, or refresh failure
  from blocking other repositories or the global UI.
- Show per-repository age, mode, error, active work, and last successful refresh.
- Preserve selection and workspace state across refresh, reorder, filtering, and
  switching while invalidating stale operations and diffs safely.
- Bound discovery, worker count, memory, output, and refresh frequency.

## Acceptance criteria

- A collection containing healthy, missing, nested, linked, slow, and failing
  repositories remains navigable and accurately classified.
- Repository switching cancels or scopes all old work and cannot display old
  repository content in the new workspace.
- Persistence survives restart, partial writes, permission errors, and schema
  migration without corrupting the repository or hiding core status.
- Tests cover ordering, deduplication, limits, errors, cancellation, stale
  rows, persistence recovery, and concurrent refresh budgets.

## Verification and documentation

Update multi-repository and configuration docs, add disposable collection
fixtures, run race/performance gates, and collect native switching evidence.

**Status:** Complete

**Completion summary:** Hardened multi-repository registry persistence with a
versioned envelope, legacy-array migration, private atomic replacement and
sync, bounded configured discovery limits, symlink/VCS/vendor boundaries, and
independent per-repository refresh/error handling. Existing registry metadata
and selection flows remain compatible; native switching evidence remains part
of the release matrix.
