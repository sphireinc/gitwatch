# Task 96 — Git version compatibility and capability matrix

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 3, 24, 26, 34, and 92

## Objective

Make supported Git versions and capability differences explicit so gitwatch
fails clearly or selects safe fallbacks instead of assuming one modern Git
behavior.

## Required work

- Define the minimum supported Git version and a tested compatibility range.
- Inventory every Git subcommand, option, porcelain field, exit status, and
  output assumption used by the application.
- Add capability probes for optional features such as restore, worktree,
  switch, stash formatting, branch formatting, diff options, and tracking
  metadata.
- Use machine-readable output wherever available and isolate version-specific
  fallbacks in `internal/git`; never parse human output merely for convenience.
- Add fixtures and real-Git tests for minimum, current, and representative
  older/newer versions, including changed error text and missing optional
  capabilities.
- Document behavior for unsupported, partially supported, and unknown versions.

## Acceptance criteria

- Startup reports Git version and actionable unsupported-capability diagnostics.
- Core status, stage/unstage, diff, and refresh workflows work across the
  declared version matrix or are explicitly unavailable with safe messaging.
- Capability results are cached per repository/session and cannot silently
  authorize a destructive operation.
- Tests verify argument vectors, fallback selection, malformed output, and
  version comparison boundaries.

## Verification and documentation

Update `docs/edge-cases.md`, README requirements, release-check commands, and
the beta matrix. Run the full gate against every locally available Git version;
label container or CI evidence separately from native evidence.

**Status:** Complete

**Completion summary:** Added Git 2.23 minimum-version enforcement, semantic
version parsing, cached per-discovery capability results, startup diagnostics,
and compatibility documentation. Core machine-readable workflows remain
argument-vector based; capability-gated optional behavior is not authorized by
fallback error text.
