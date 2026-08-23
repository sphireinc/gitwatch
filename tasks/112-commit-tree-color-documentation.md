# Task 112 — Document and release-validate colorized commit trees

**Priority:** P2
**Lane:** v1.x UX and release evidence
**Dependencies:** Task 111

## Objective

Document the colorized graph and collect compatible v1.x release evidence.

## Requirements

Update `README.md`, `ARCHITECTURE.md`, `UX_SPEC.md`, `docs/commit-tree.md`,
`docs/beta-validation-matrix.md`, and `docs/release-checklist.md` with the Git
contract, safe parser, semantic theme behavior, color-independent semantics,
wrapping, fallback behavior, supported themes, troubleshooting, and native
evidence requirements. Update keymap/config docs only if behavior changes.
Do not document raw ANSI sequences as a user-facing contract.

Verify disabled/default behavior, CLI/config compatibility, supported Git
versions, malformed-output fallback, `NO_COLOR`, high contrast, wide/medium/
narrow terminals, merge/decorated graphs, resize, scrolling, and absence of
raw controls or secrets in fixtures/screenshots/bundles. Separate automated
results from maintainer-run visual evidence.

## Acceptance

Public documentation is internally consistent, reproducible validation commands
and expected results are recorded, all quality gates pass, and remaining human
visual QA is explicitly identified.

**Status:** Planned
