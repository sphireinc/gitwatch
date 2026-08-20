# Task 25 — Performance engineering and large-repo behavior

**Priority:** P0

Benchmark parser, snapshot diffing, sorting/filtering, rendering, watcher burst handling. Create synthetic repos/status fixtures for 1k/10k/50k paths. Avoid per-frame Git calls, per-row allocations where practical, and rebuilding expensive strings when state is unchanged. Consider virtualization in the table.

**Acceptance:** 10k changed files remains interactively usable; record benchmark numbers in `docs/performance.md`; no UI freeze during status execution.

**Status:** Complete — parser and table benchmarks, 10k-row bounded-index implementation, and performance guidance/commands are documented in `docs/performance.md`.
