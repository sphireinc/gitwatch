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
`VERSION=1.0.6 ./scripts/release-check.sh` candidate gate passed locally,
including tests, race tests, security/performance checks, five-target archive
verification, checksums, SBOM/dependency packaging, and build identity. Native
macOS, Linux, and Windows terminal evidence remains operator-owned and is not
claimed here.
