# Release checklist

This checklist separates repeatable automation from native human acceptance and publication. A passing local command is evidence only for the commit and host on which it ran.

## Automated candidate gate

Run on the exact release commit:

```sh
make check
./scripts/secret-scan.sh --history
VERSION=1.0.0 ./scripts/release-check.sh
```

The automated gate covers formatting, lint, tests, race detection where supported, vet, security boundaries, representative performance budgets, an isolated source install, CLI/config inspection, a disposable real Git repository, cross-platform archive creation, dependency-license packaging, extraction, embedded build identity, and checksums.

Tagged pushes matching `v*.*.*` run `.github/workflows/release.yml`. The workflow builds macOS amd64/arm64, Linux amd64/arm64, and Windows amd64 archives, verifies their contents and `SHA256SUMS`, produces an SBOM and build provenance, and publishes through the protected release environment.

## Native operator acceptance

Record the commit/tag, OS and architecture, terminal and dimensions, Git version, watch mode, and evidence link for every run.

- [ ] macOS: launch, keyboard, mouse, selected-file diff, stage/unstage, external filesystem refresh, polling refresh, resize, color/`NO_COLOR`, reduced motion, and clean quit.
- [ ] Linux: the same complete terminal workflow.
- [ ] Windows: the same complete terminal workflow using native paths and terminal behavior.
- [ ] Linked worktree, submodule, detached HEAD, unborn branch, conflict, rename, Unicode, spaces, quotes, leading hyphen, long supported path, and large-worktree cases.
- [ ] Clean-machine archive installation and source installation.
- [ ] Missing Git and non-repository diagnostics.
- [ ] No orphan child process or altered terminal state after normal quit, cancellation, or failure.
- [ ] No open release blocker, critical security issue, known data-loss defect, or unresolved license concern.

CI runtime smoke tests prove startup and basic Git discovery on each runner OS. They do not replace native mouse, resize, filesystem notification, terminal restoration, or clean-machine acceptance.

## Publication

- [ ] Freeze `CHANGELOG.md` into a dated release section and finalize `docs/release-v1.0.0.md`.
- [ ] Confirm canonical module path, repository metadata, issue labels, discussions/support links, security reporting, branch rules, and least-privilege Actions settings.
- [ ] Create and verify the signed `v1.0.0` tag from the accepted commit.
- [ ] Approve the protected release environment and review generated notes, checksums, SBOM, provenance, archives, license files, and signatures.
- [ ] Verify `go install github.com/sphireinc/git-watch/cmd/gitwatch@v1.0.0` from outside the source checkout.
- [ ] Publish and test package-manager metadata.
- [ ] Publish the announcement and genuine demo assets.
- [ ] Monitor security, crash, data-loss, and install reports after launch.

Do not call the release accepted while any required row remains unchecked or lacks evidence.
