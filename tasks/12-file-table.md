# Task 12 — High-density interactive file table

**Priority:** P0

Implement selectable/scrollable rows with status glyphs, path, rename source when relevant, staged/worktree indicators, optional diffstat columns, and conflict labels. Add sort modes: path, status, staged-first, changed-most; fuzzy filter via `/`; preserve selection by stable path identity across refreshes when possible.

**Acceptance:** 10k rows remain navigable; selection does not randomly jump during ordinary refresh; filter and sort are keyboard-accessible.

**Status:** Complete — selectable virtual row indexes, stable-path selection, path/status/staged/changed sorting, subsequence fuzzy filtering, scroll offsets, and 10,000-row tests are implemented.
