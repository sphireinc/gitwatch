# Task 20 — In-memory activity timeline

**Priority:** P1

Create a bounded event log from snapshot diffs and user operations: file became modified, staged, unstaged, removed from status, branch changed, watcher fallback, refresh error, operation success/failure. Keep it session-local; no telemetry or persistent logging by default.

**Acceptance:** Event storms are coalesced; timeline memory is bounded.
