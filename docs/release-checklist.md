# Release checklist

`./scripts/release-check.sh` is the repeatable local gate for evidence that can run on the current host. It covers the Go test suite, race detector, vet, build, help/version/config inspection, a real disposable Git demo repository, and release archive checksum verification.

The following acceptance evidence remains platform/operator-owned and must be returned before a public v1 tag:

- native Windows run with real path, mouse, resize, stage, unstage, and quit checks;
- macOS and Linux terminal recordings at the supported sizes;
- manual confirmation that filesystem events and polling both refresh after external edits;
- clean-machine installation and Git-missing behavior;
- signed tag/GitHub Release publication and package-manager updates.

This distinction prevents automated local gates from being presented as human acceptance on platforms not available to the operator.
