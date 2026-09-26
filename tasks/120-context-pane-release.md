# Task 120 — Release the context-pane feature

**Priority:** P2
**Lane:** v1.x release
**Dependencies:** Task 119

## Objective

Close release evidence for the lower-left context-pane family and preserve
backward compatibility for users who leave it disabled.

## Acceptance

- Existing configurations behave unchanged with the new pane family unused.
- Built-in shortcuts and configured overrides are documented.
- Git 2.23+ compatibility, archive builds, checksums, security scans, SBOM,
  and provenance checks pass.
- Native macOS, Linux, and Windows evidence covers enabled/disabled panes,
  resize, keyboard/mouse operation, no-upstream and unpushed states.
- Remaining operator-owned evidence is explicitly recorded before release.

**Status:** In progress — implementation and documentation are complete; the
exact `VERSION=1.0.9 ./scripts/release-check.sh` candidate gate passed at
`0f10d5d`, including tests, race tests, security/performance checks, five-target
archive verification, checksums, dependency-license/SBOM input packaging, and
build identity. The owner-provided matrix disposition covers all listed
terminal, OS, Git, workbench, integration, and workload cells for candidate
`5b2a8e9`; Linux is accepted by explicit macOS equivalence, not represented as
physical Linux testing. On 2026-09-25 the owner approved carrying the
context-pane disposition to `34562b5` because the context-pane source is
unchanged through that revision. This is owner-approved acceptance by source
equivalence, not a fresh native transcript at `34562b5`; physical
multi-distribution Linux testing continues.
