# Task 05 — Immutable repository snapshot and status semantics

**Priority:** P0

Define domain types for paths, XY status, index/worktree state, conflict state, rename source, submodule flags, branch/upstream divergence, counts, timing, and generation number. Derive user-facing status labels from domain values in one place.

**Acceptance:** Snapshot can represent staged+unstaged same file, deletion, rename, type change, untracked, conflict, detached HEAD, unborn branch, no upstream.

**Status:** Complete — immutable snapshot, branch/divergence, entry state, counts, timing, generation, cloning, and centralized semantic labels added.
