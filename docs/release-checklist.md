# Release checklist

`./scripts/release-check.sh` is the repeatable local gate for evidence that can run on the current host. It covers the Go test suite, race detector, vet, build, help/version/config inspection, an isolated temporary `go install`, a real disposable Git demo repository, and release archive checksum/extraction verification.

Tagged pushes matching `v*.*.*` run `.github/workflows/release.yml`. That
workflow cross-builds the five supported archives (`darwin/amd64`,
`darwin/arm64`, `linux/amd64`, `linux/arm64`, and `windows/amd64`), verifies
`SHA256SUMS`, and attaches the archives to a GitHub Release. Repository owners
must still review the generated release notes and approve publication.

The following acceptance evidence remains platform/operator-owned and must be returned before a public v1 tag:

- native Windows run with real path, mouse, resize, stage, unstage, and quit checks;
- macOS and Linux terminal recordings at the supported sizes;
- manual confirmation that filesystem events and polling both refresh after external edits;
- clean-machine installation and Git-missing behavior;
- signed tag/GitHub Release publication and package-manager updates.

This distinction prevents automated local gates from being presented as human acceptance on platforms not available to the operator.

The CI matrix also runs an OS-specific runtime smoke check: it launches the
built binary for version/help/config validation, creates a real disposable Git
repository, commits a file, modifies it, and requires non-empty porcelain
status output. This proves process startup and basic Git discovery on each CI
OS; it does not replace terminal mouse/resize/watch-mode recordings.
