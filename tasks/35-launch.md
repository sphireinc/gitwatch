# Task 35 — v1.0.0 launch

**Priority:** P0

Run `RELEASE_CRITERIA.md` line-by-line. Update changelog, version, release notes, screenshots/GIFs, package metadata, and checksums. Tag signed release if project policy supports it. Publish GitHub Release and installation channels. Create post-launch issue labels/milestones for v1.1 ideas rather than slipping them into v1.

**Acceptance:** v1 artifacts install, launch, detect a repo, refresh live, stage/unstage safely, show diff/help, and exit cleanly on all supported platforms.

**Status:** In progress — v1.0.9 was published on 2026-09-22 from signed tag `v1.0.9`, which resolves to `951f3f64c64fef31c3fc088577f14713e1175ffb`. GitHub verified the tag signature, and [release workflow 35696232803](https://github.com/sphireinc/gitwatch/actions/runs/35696232803) completed build/verification, SBOM generation, artifact attestation, and publication successfully. The release contains five platform archives, release metadata, `SHA256SUMS`, and an SPDX SBOM. GitHub tracking for future ideas is set up with the `v1.1-candidate` label and the non-committal [v1.1 Ideas milestone](https://github.com/sphireinc/gitwatch/milestone/1).

The release is not fully accepted: native macOS/Linux/Windows operator runs, clean-machine archive and source installation, upgrade behavior, terminal/process restoration, and exact-candidate data-loss/blocker review remain pending in the [release checklist](../docs/release-checklist.md) and [beta validation matrix](../docs/beta-validation-matrix.md). Publication does not substitute for those acceptance gates. See the [current release record](../docs/release-signoff.md) for the distinction between the historical prepublication candidate and the published tag.
