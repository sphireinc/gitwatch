# Task 07 — Event-driven filesystem watcher

**Priority:** P0

Use `fsnotify` to watch the worktree and relevant Git metadata. Dynamically add watches for created directories; exclude `.git` worktree traversal except explicitly required metadata paths; honor Git ignored files only for display, not necessarily watcher registration. Treat events as refresh hints. Debounce near 75 ms. Handle rename/remove/recreate of watched paths.

**Acceptance:** Editing, creating, deleting, renaming, staging, committing in another terminal, branch switch, checkout, merge-state changes all cause refresh on macOS/Linux/Windows test matrix where feasible.

**Status:** Complete — the production TUI runs the fsnotify watcher with recursive and dynamic directory registration, linked-worktree Git metadata paths, remove/recreate recovery, debounce, cancellation, and authoritative refresh integration. Worktree and external-metadata changes are tested.
