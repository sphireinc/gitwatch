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

**Status:** Complete for this task's acceptance scope, with the context-pane
operator disposition bounded to `34562b5`. Existing configurations retain the
disabled-by-default behavior (`README.md`, `internal/config` tests); built-in
`T`/`P`/`B` shortcuts and safe keymap overrides are documented in `README.md`,
`KEYMAP.md`, and `docs/context-panes.md`. The exact
`VERSION=1.0.9 ./scripts/release-check.sh` gate passed at `0f10d5d`, including
tests, race tests, security/performance checks, five-target archive
verification, checksums, dependency-license/SBOM input packaging, and build
identity. The owner-provided beta matrix disposition for `5b2a8e9` was
explicitly carried to `34562b5` for the context-pane feature because its source
was unchanged through that revision. That owner acceptance covers enabled and
disabled panes, resizing, keyboard/mouse operation, and no-upstream/unpushed
states on macOS and Windows, with Linux accepted by the stated macOS-equivalence
exception. It is not a fresh native transcript or physical Linux test.

The separate release workflow for `9b1cb34` passed archive, checksum, SBOM,
attestation/provenance, and publication checks ([run 36307343578](https://github.com/sphireinc/gitwatch/actions/runs/36307343578));
this does not extend Task 120's owner-approved matrix carry beyond `34562b5`.
Physical multi-distribution Linux testing and the broader, row-by-row native
release evidence remain tracked by `docs/beta-validation-matrix.md` and
`docs/release-checklist.md`; they are broader release gates, not unsatisfied
Task 120 criteria. The remaining operator-owned evidence is explicitly
recorded there.
