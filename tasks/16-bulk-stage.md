# Task 16 — Bulk stage/unstage actions

**Priority:** P0

Add stage-all and unstage-all with explicit scope text. Stage-all must clearly state whether it includes deletions/untracked files. Add visible pending state and operation summary. Do not overload single-file Space behavior.

**Acceptance:** Commands match documented scope exactly; conflict states and errors leave UI consistent.

**Status:** Complete — bulk stage uses `git add --all --` (including deletions/untracked files) and bulk unstage restores the full index scope with a fallback; operation results expose explicit scope.
