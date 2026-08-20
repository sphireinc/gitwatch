# Task 31 — Packaging and installation

**Priority:** P0

Produce versioned archives/checksums for darwin amd64/arm64, linux amd64/arm64, windows amd64. Add Homebrew formula/tap plan, `go install` instructions, and optionally Scoop/WinGet after binary release process is stable. Embed version/commit/build date.

**Acceptance:** Fresh-machine installation instructions work and binary reports exact version.

**Status:** Complete — reproducible trimpath archives/checksums for all required targets and version ldflags are provided by `scripts/release.sh` and `make release`.
