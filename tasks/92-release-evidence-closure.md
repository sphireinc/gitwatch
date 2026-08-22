# Task 92 — Close v1 release evidence and beta sign-off

**Priority:** P0
**Lane:** v1 release closure
**Dependencies:** Tasks 34, 35, and 89; `docs/release-checklist.md`; `docs/beta-validation-matrix.md`

## Objective

Turn the existing automated and partial operator evidence into a complete,
auditable v1 release decision. This task is evidence and release engineering,
not an opportunity to add features. Any defect discovered during validation
must become a separate focused task or a release-blocking fix with its own
commit.

## Required work

- Select and record one exact candidate commit and version for all evidence.
- Run the pinned automated gate with the documented Go and linter versions.
- Complete native macOS, Linux, and Windows validation for every applicable row
  in the beta matrix, including mouse, resize, watch/poll, terminal restore,
  unusual paths, clean repositories, conflicts, worktrees, and large fixtures.
- Capture terminal dimensions, emulator, OS/architecture, Git version, watch
  mode, commands, result, and evidence links for every row.
- Classify findings as blocker, critical, normal, or deferred; do not silently
  convert a failed required row into a documentation-only note.
- Reconcile `tasks/34-beta.md`, `tasks/35-launch.md`, and `tasks/89-post-v1-beta.md`
  with the actual evidence. Preserve historical observations as historical.
- Produce a final release decision record with explicit unresolved risks and
  a named owner for anything deferred to v1.x.

## Acceptance criteria

- Every required matrix cell is `pass` with exact-candidate evidence, or the
  release is explicitly blocked with the reason recorded.
- No known blocker, critical security issue, data-loss issue, or terminal-state
  corruption issue remains open.
- Automated artifacts, checksums, provenance, SBOM, notices, and release notes
  all refer to the same source commit and version.
- A clean checkout can reproduce the candidate gate and install/launch test.
- The release decision is reviewable without relying on agent conversation
  history.

## Verification and documentation

Run `make check`, the history secret scan, release checks, and the native matrix.
Update the release checklist, beta matrix, changelog, release notes, and task
status. Do not mark this task complete based only on CI success.

**Status:** Planned
