# Task 04 — Porcelain v2 NUL parser

**Priority:** P0

Implement a strict parser for `git status --porcelain=v2 -z --branch`. Cover ordinary entries, renamed/copied entries, unmerged/conflicted entries, untracked, ignored when enabled, branch oid/head/upstream/ahead-behind headers, spaces and unusual path characters. Parse bytes without line-scanner assumptions that break NUL records.

**Acceptance:** Table-driven fixtures cover every porcelain v2 record type used by Git; fuzz parser; malformed input returns typed parse errors rather than panic.

**Status:** Complete — strict NUL parser covers branch headers, ordinary, rename/copy, unmerged, untracked, and ignored records with typed malformed-input errors and unusual path bytes preserved.
