# Task 19 — Conflict-aware UX

**Priority:** P0

Represent unmerged records prominently. Show conflict type (both modified, deleted by us/them, etc.) where derivable. Do not offer discard-like shortcuts from generic status actions. Stage resolved files normally after user edits them externally. Add a conflict filter.

**Acceptance:** Merge-conflict fixtures render correctly and no unrelated operation destroys conflict stages.

**Status:** Complete — conflict type labels, conflict-only table filtering, and a generic-operation guard for unmerged entries are implemented; conflict state remains explicit until the user resolves it externally.
