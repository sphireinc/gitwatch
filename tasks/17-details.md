# Task 17 — Selected-file details pane

**Priority:** P0

Show path, previous path for rename, index/worktree status, conflict stage when applicable, file mode/type, submodule state, staged/unstaged booleans, diffstat, last observed change time, and operation hints. Cache expensive metadata by snapshot generation.

**Acceptance:** Details never block main render and update correctly as selection/status changes.

**Status:** Complete — pure selected-entry detail construction and generation-scoped caching expose path/rename/status/mode/staged/unstaged/conflict data and operation hints without render-time I/O.
