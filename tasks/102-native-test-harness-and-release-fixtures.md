# Task 102 — Native terminal test harness and release fixture suite

**Priority:** P0
**Lane:** release quality
**Dependencies:** Tasks 29, 30, 34, 35, 89, 92, and 93

## Objective

Reduce the gap between automated Go tests and the human terminal acceptance
required for releases by creating reproducible native fixtures and evidence
capture without pretending screenshots replace behavioral assertions.

## Required work

- Define disposable repositories for clean, mixed, conflict, rename/copy,
  unusual-byte, worktree, submodule, large, hook-failure, and remote scenarios.
- Add deterministic commands to prepare, reset, and inspect each fixture.
- Provide native runner instructions for macOS, Linux, and Windows terminal
  dimensions, emulators, shells, Git versions, and encoding settings.
- Capture scripted keyboard/mouse/resize sequences where the terminal supports
  it, with manual checkpoints for semantics a script cannot verify.
- Add artifact collection for stdout/stderr, status before/after, screenshots
  or recordings, process lists, and terminal-restoration result.
- Make failures actionable and avoid embedding personal paths, credentials, or
  machine-specific timestamps in public evidence.

## Acceptance criteria

- A maintainer can reproduce every required beta-matrix workflow from a clean
  checkout using documented commands.
- Evidence identifies exact commit, OS, architecture, terminal, dimensions,
  Git version, fixture, and result.
- Automated fixture assertions verify repository state, while native evidence
  verifies interaction, rendering, and terminal lifecycle.
- The harness cleans up repositories, child processes, recordings, and temp
  files on pass and failure.

## Verification and documentation

Update `docs/beta-validation-matrix.md`, operator guides, CI smoke boundaries,
and release sign-off templates. Exercise the harness on every supported OS.

**Status:** Planned
