# Task 03 — Repository discovery and Git capability probe

**Priority:** P0

Resolve worktree root, git dir, common dir, bare/non-bare state, submodule/worktree context, Git version, and HEAD mode. Define behavior when launched from a nested directory. Reject bare repositories in v1 with a useful message unless a later task explicitly adds support.

**Acceptance:** Works in normal repo, nested cwd, detached HEAD, linked worktree, and submodule fixture.
