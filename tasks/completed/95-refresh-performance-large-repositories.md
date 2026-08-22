# Task 95 — Refresh performance and large-repository scalability

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 6–8, 25, 77, 84, and 92

## Objective

Keep gitwatch responsive in large worktrees and multi-repository sessions by
measuring and reducing refresh cost without weakening Git-authoritative truth.

## Required work

- Establish benchmarks for porcelain parsing, snapshot construction, path
  wrapping, table filtering/sorting, diff loading, history paging, registry
  refresh, and watcher reconciliation at realistic sizes.
- Capture allocations, latency distributions, peak memory, process counts,
  refresh coalescing, and UI message backlog rather than relying on averages.
- Identify safe reuse or incremental projection opportunities while keeping
  immutable snapshot semantics and generation ordering intact.
- Add explicit budgets for status refresh, render preparation, history/patch
  parsing, multi-repo workers, plugin output, and diff payloads.
- Ensure a watcher storm, repeated resize, rapid filtering, and concurrent
  operations cannot create unbounded goroutines, channels, queues, or memory.
- Add cancellation and shutdown benchmarks/tests for every worker family.

## Acceptance criteria

- Baseline and optimized benchmark results are checked in or reproducibly
  generated, with hardware/toolchain context recorded.
- Large fixtures remain navigable, filterable, and selectable without dropped
  input or visible stale state.
- Performance budgets fail deterministically in tests/CI when regressions exceed
  the documented tolerance.
- No optimization changes Git command authority, path byte preservation,
  refresh-after-mutation behavior, or error semantics.

## Verification and documentation

Update `docs/performance.md`, add benchmark-budget tests, run race tests and
the release gate, and provide manual responsiveness evidence separately from
automated benchmark evidence.

**Status:** Complete — added a 10,000-entry porcelain-v2 allocation budget, expanded reproducible benchmark recording guidance, preserved existing parser/history/registry budgets, and verified the focused workload suites.
