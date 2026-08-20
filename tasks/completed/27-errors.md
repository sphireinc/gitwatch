# Task 27 — Error recovery and diagnostics

**Priority:** P0

Design user-facing recovery for Git missing, permissions, repo disappears, watcher exhaustion, corrupt/locked index, operation conflicts, command timeout, status parse failure. Add optional debug logging to a file via flag; keep normal UI clean. Sanitize logs for control characters.

**Acceptance:** Recoverable errors do not crash; fatal errors restore terminal and exit nonzero with concise message.

**Status:** Complete — typed Git errors, cancellable operations, concise CLI fatal errors, optional 0600 debug logging, and sanitized diagnostic text are implemented.
