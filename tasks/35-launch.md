# Task 35 — v1.0.0 launch

**Priority:** P0

Run `RELEASE_CRITERIA.md` line-by-line. Update changelog, version, release notes, screenshots/GIFs, package metadata, and checksums. Tag signed release if project policy supports it. Publish GitHub Release and installation channels. Create post-launch issue labels/milestones for v1.1 ideas rather than slipping them into v1.

**Acceptance:** v1 artifacts install, launch, detect a repo, refresh live, stage/unstage safely, show diff/help, and exit cleanly on all supported platforms.

**Status:** In progress — v1.0.9 was published on 2026-09-22 from signed tag `v1.0.9`, which resolves to `951f3f64c64fef31c3fc088577f14713e1175ffb`. GitHub verified the tag signature, and [release workflow 35696232803](https://github.com/sphireinc/gitwatch/actions/runs/35696232803) completed build/verification, SBOM generation, artifact attestation, and publication successfully. The release contains five platform archives, release metadata, `SHA256SUMS`, and an SPDX SBOM. GitHub tracking for future ideas is set up with the `v1.1-candidate` label and the non-committal [v1.1 Ideas milestone](https://github.com/sphireinc/gitwatch/milestone/1).

The release is not fully accepted: native macOS/Linux/Windows operator runs, clean-machine archive and source installation, upgrade behavior, terminal/process restoration, and exact-candidate data-loss/blocker review remain pending in the [release checklist](../docs/release-checklist.md) and [beta validation matrix](../docs/beta-validation-matrix.md). Publication does not substitute for those acceptance gates. See the [current release record](../docs/release-signoff.md) for the distinction between the historical prepublication candidate and the published tag.

## Local v1.0.9 install/runtime supplement (2026-09-25)

On Darwin 25.6.0 arm64, the published `gitwatch_1.0.9_darwin_arm64.tar.gz`
archive passed its `SHA256SUMS` check, extracted with the packaged license and
third-party notices, and reported
`1.0.9 (951f3f64c64fef31c3fc088577f14713e1175ffb, 2026-09-22T02:35:07-04:00)`.
An isolated `HOME`/`XDG_CONFIG_HOME` `--config-check` reported valid. A
separate `go install github.com/sphireinc/git-watch/cmd/gitwatch@v1.0.9`
reported `1.0.9 (unknown, unknown)`, as expected for a module install without
release linker metadata.

Both installed binaries passed `scripts/record-demo.sh` against a clean,
disposable Git repository at 80x24: an external edit appeared without manual
refresh, the diff rendered, stage/unstage transitioned the staged count
`0 → 1 → 0`, final porcelain-v2 showed only the intended modified file with no
staged residue, and the TUI exited cleanly. The published binary also migrated
legacy schema-v1 and schema-v2 configuration in memory to v3; config inspection
showed the v3 workspace/visuals fields, and both source-file hashes remained
unchanged. `go test ./internal/config` passed on the current checkout.

For the stable upgrade path, the official v1.0.8 macOS ARM64 archive also
passed its `SHA256SUMS` check and reported commit `67ff7ba23066977345e0a0edfd2ab761ed940462`.
Its isolated `--config-inspect` output was accepted by v1.0.8 `--config-check`
as schema v2, then the v1.0.9 binary dry-ran and inspected it as schema v3.
The schema-v2 file hash remained
`c7ec13c79188967e8c182bad86ba1a381a0cad6954d57579c86e1cf7fd6f98cd` across
the v1.0.9 migration/inspection. This verifies configuration compatibility
between the two stable releases, not a clean-machine executable or package
manager upgrade.

This is isolated local/automated evidence on Go 1.27.0, not a clean-machine or
native operator run; it does not verify clean-machine upgrade behavior, Linux
or Windows installation, or any pending platform matrix row. Task 35 remains
in progress.
