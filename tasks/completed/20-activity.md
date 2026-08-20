# Task 20 — In-memory activity timeline

**Priority:** P1

Create a bounded event log from snapshot diffs and user operations: file became modified, staged, unstaged, removed from status, branch changed, watcher fallback, refresh error, operation success/failure. Keep it session-local; no telemetry or persistent logging by default.

**Acceptance:** Event storms are coalesced; timeline memory is bounded.

**Status:** Complete — bounded session-local event log and snapshot-diff event derivation cover file, branch, refresh, watcher, and operation activity without persistence or telemetry.
