# Task 103 — Distribution channels and upgrade-safe release automation

**Priority:** P1
**Lane:** release/distribution
**Dependencies:** Tasks 31, 35, 92, and 102

## Objective

Make installation and upgrade paths predictable beyond raw release archives,
without making package-manager publication a hidden release dependency.

## Required work

- Define supported distribution channels and ownership boundaries for each
  package manager; start with channels that can be maintained reliably.
- Generate metadata from the canonical version, checksums, license/notices,
  supported targets, and release URL rather than duplicating version strings.
- Add isolated install, upgrade, uninstall, and version/help checks for every
  channel, including PATH and shell-completion behavior where applicable.
- Verify archives contain only intended files, preserve executable modes, and
  include provenance/SBOM/checksum assets as promised.
- Document rollback and recovery when a package-manager update is delayed or
  partially published.
- Harden release workflow permissions, approvals, tag provenance, and artifact
  promotion so a failed channel cannot publish a misleading release.

## Acceptance criteria

- A clean machine can install the documented package and run `gitwatch --help`
  and `gitwatch --version` without source checkout artifacts.
- Upgrade from the previous supported release preserves user config and does
  not silently change destructive bindings or plugin trust state.
- Release automation verifies metadata, archive contents, checksums, SBOM,
  provenance, and package-manager manifests before publication.
- A package channel can be disabled or rolled back without deleting releases
  or weakening repository security controls.

## Verification and documentation

Update install docs, release checklist, CI/release workflows, and changelog.
Run exact-version clean-machine tests and record channel-specific evidence.

**Status:** Complete

**Completion summary:** Added generated release metadata tied to the canonical
version, commit, build date, targets, repository, and license; included and
verified it in release checksums; documented maintained archive/source channels,
optional package-manager ownership boundaries, upgrade preservation, and
rollback behavior.
