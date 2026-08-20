# Task 15 — Unstage selected path safely

**Priority:** P0

Implement unstage with a Git-version-compatible command, preferring modern `git restore --staged -- <path>` where supported and a safe fallback if required. Preserve working-tree changes. Handle unborn repositories explicitly.

**Acceptance:** Integration tests prove working tree content is unchanged after unstage.

**Status:** Complete — unstage prefers `git restore --staged -- <path>` with a reset fallback and verifies working-tree preservation.
