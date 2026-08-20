# Task 06 — Authoritative refresh engine

**Priority:** P0

Build snapshot acquisition around the Git runner. Enforce at most one active status refresh per repository. Coalesce repeated refresh requests with a dirty flag so a burst cannot create an unbounded queue. Attach duration and monotonic generation.

**Acceptance:** Concurrency tests show no overlapping status subprocesses and no missed final refresh after a burst.
