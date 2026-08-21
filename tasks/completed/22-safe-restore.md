# Task 22 — Discard/restore workflow with hard confirmations

**Priority:** P1

Add optional v1 restore/discard action only with a confirmation modal that names the exact path and whether staged/unstaged content is affected. Prefer Git restore commands. Never implement `git reset --hard` or `git clean -fd` as a generic shortcut. Consider requiring a typed confirmation for irreversible untracked deletion; otherwise exclude untracked deletion from v1.

**Acceptance:** No destructive command can execute from one accidental keypress/click.

**Status:** Complete — status-view `R` requires typing exact `yes` after a prompt naming the path and staged/worktree scope; only path-scoped `git restore` is exposed, its index/worktree result is integration-tested, and there is no reset-hard, clean, conflicted-path discard, or untracked deletion shortcut.
