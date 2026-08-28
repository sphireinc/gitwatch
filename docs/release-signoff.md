# Release sign-off record

This file records evidence for the exact candidate below. Automated evidence is
complete for the candidate; native operator rows remain explicitly pending, so
this record is not a v1 publication approval.

- Release commit: `c0a7d46` (`fix: honor release check version`)
- Release tag:
- Candidate date: 2026-08-28
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
- [x] `VERSION=1.0.6 ./scripts/release-check.sh` on a clean release candidate
- [ ] CI matrix and native runtime smoke checks for the release candidate
- [x] release archive extraction, identity, dependency-license, and SHA256 verification
- [ ] SBOM and build provenance

Evidence links/output: local command output for candidate `c0a7d46`; CI and
release workflow links must be attached before publication. The release-check
run generated and verified the five `gitwatch_1.0.6_*` archives, release
metadata, and checksums. The exact `make check` invocation also passed on the
current docs-only descendant `a1fc799`; rerun it on the final tagged candidate
before signing.

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

**BLOCKED — not approved for public v1 publication.** Automated repository
quality gates and the full-history secret scan pass for the recorded candidate.
The release remains blocked until Tasks 34, 35, and 89 have exact-candidate
macOS, Linux, and Windows operator evidence, clean-install/upgrade evidence,
release artifact verification, and publication approvals.
