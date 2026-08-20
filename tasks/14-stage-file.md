# Task 14 — Stage selected path safely

**Priority:** P0

Implement stage for untracked/modified/deleted/renamed path using `git add -- <path>` semantics via argument vector. Disable action for states where a generic stage is ambiguous or unsafe and explain why. Show pending row state/spinner and refresh after completion.

**Acceptance:** Handles spaces, unicode, leading hyphen, rename, deletion; error toast includes actionable stderr without dumping noise.

**Status:** Complete — stage operation uses direct argv `git add -- <path>`, preserves path bytes through the operation result, and includes unusual-path integration coverage.
