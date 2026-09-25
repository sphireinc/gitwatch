# Release sign-off record

This file preserves the prepublication sign-off snapshot for candidate
`0f10d5d` and records the later v1.0.9 publication separately. The publication
is verified; native operator acceptance remains open and is not implied by the
release workflow.

## Current public release: v1.0.9

- Published: 2026-09-22 06:49:28 UTC
- Signed tag: [`v1.0.9`](https://github.com/sphireinc/gitwatch/releases/tag/v1.0.9), targeting `951f3f64c64fef31c3fc088577f14713e1175ffb`
- GitHub reports the annotated tag signature as valid; the release workflow's signed-tag verification step passed.
- [Release workflow 35696232803](https://github.com/sphireinc/gitwatch/actions/runs/35696232803) succeeded, including archive/SHA256 verification, SPDX SBOM generation and verification, artifact/SBOM attestation, and GitHub publication.
- Published assets: five target archives (macOS amd64/arm64, Linux amd64/arm64, Windows amd64), release metadata, `SHA256SUMS`, and the SPDX SBOM.
- GitHub post-v1 tracking: `v1.1-candidate` label and [v1.1 Ideas milestone](https://github.com/sphireinc/gitwatch/milestone/1) exist; the milestone is explicitly not a schedule or release commitment.
- The published GitHub release notes report the `go-runewidth` update from `0.0.29` to `0.0.30`; see the [release page](https://github.com/sphireinc/gitwatch/releases/tag/v1.0.9) for the exact published notes.

**Acceptance remains in progress.** The release workflow does not provide native interactive evidence. macOS, Linux, and native Windows operator runs, clean-machine installation, upgrade/migration behavior, terminal and child-process restoration, and exact-candidate release-blocker/data-loss review remain pending in the [beta validation matrix](beta-validation-matrix.md) and [release checklist](release-checklist.md).

### Local v1.0.9 install/runtime supplement (2026-09-25)

On Darwin 25.6.0 arm64, the published macOS ARM64 archive matched its release
checksum, extracted with its license/notice files, and reported the signed-tag
identity. The source-installed `@v1.0.9` binary also reported version 1.0.9.
Both binaries passed the scripted 80x24 watcher-refresh, diff, stage/unstage,
and clean-exit demo in a disposable repository. The archive binary's isolated
config check passed; schema-v1 and schema-v2 migration dry-runs/inspection
normalized to schema v3 without rewriting either source file. This automated
host check does not satisfy clean-machine or Linux/Windows operator acceptance,
and does not verify a clean-machine or package-manager upgrade. A separate
stable-tag config migration check accepted a schema-v2 config inspected by the
official v1.0.8 macOS ARM64 binary, then verified v1.0.9's in-memory migration
to schema v3 without rewriting the file. This is configuration compatibility
evidence, not a full installed-binary upgrade acceptance.

## Historical prepublication candidate: `0f10d5d`

The records below describe the candidate and decision state before v1.0.9 was
published. They are retained as history and must not be read as the current
tag identity or as evidence that native acceptance has passed.

- Release commit: `0f10d5d` (`fix: isolate demo fixture signing`)
- Release tag:
- Candidate version: `1.0.9`
- Candidate date: 2026-09-22
- Release owner: project maintainer

## Automated evidence

- [x] Go 1.27 pinned lint (`go run ...@v2.12.0 run`)
- [x] `go test ./...`
- [x] `go test -race ./...`
- [x] `go vet ./...`
- [x] `./scripts/security-check.sh`
- [x] `./scripts/performance-check.sh`
- [x] `./scripts/secret-scan.sh --history`
- [ ] `make check` exact shell invocation (individual constituent gates above passed)
- [x] `VERSION=1.0.9 ./scripts/release-check.sh` on a clean release candidate
- [ ] CI matrix and native runtime smoke checks for the release candidate
- [x] release archive extraction, identity, dependency-license, and SHA256 verification
- [ ] SBOM and build provenance

Evidence links/output: local command output for candidate `0f10d5d`; CI and
release workflow links must be attached before publication. The release-check
run generated and verified the five `gitwatch_1.0.9_*` archives, release
metadata, and checksums. The exact `make check` invocation passed on the
current pre-release descendant `28d34b5`; rerun it on the final tagged
candidate before signing.

## Operator evidence

- [ ] macOS complete terminal run — pending native maintainer evidence
- [ ] Linux complete terminal run — pending native maintainer evidence
- [ ] Windows complete terminal run — pending native maintainer evidence
- [ ] clean-machine archive and source installation
- [ ] upgrade/migration behavior
- [ ] Git-missing and non-repository behavior
- [ ] no orphan child process or altered terminal state
- [ ] no open release blocker or known data-loss issue — blocked until native rows complete

Evidence links/recordings: see `docs/beta-validation-matrix.md` and
`docs/operator-terminal.md`; all native cells remain pending.

## Publication

- [ ] signed release tag
- [ ] protected GitHub Release approval
- [ ] archives, checksums, licenses, SBOM, and provenance
- [ ] installation channel/package metadata
- [ ] issue labels and milestones
- [ ] announcement and demo assets

- Signed by:
- Date:

## Release decision

**Historical decision for candidate `0f10d5d`: BLOCKED — not approved for public
v1 publication at that time.** Automated repository quality gates and the
full-history secret scan passed for that recorded candidate. This snapshot was
superseded by the later v1.0.9 publication; the native acceptance items remain
open as described in the current-release record above.
