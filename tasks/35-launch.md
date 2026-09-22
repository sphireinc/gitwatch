# Task 35 — v1.0.0 launch

**Priority:** P0

Run `RELEASE_CRITERIA.md` line-by-line. Update changelog, version, release notes, screenshots/GIFs, package metadata, and checksums. Tag signed release if project policy supports it. Publish GitHub Release and installation channels. Create post-launch issue labels/milestones for v1.1 ideas rather than slipping them into v1.

**Acceptance:** v1 artifacts install, launch, detect a repo, refresh live, stage/unstage safely, show diff/help, and exit cleanly on all supported platforms.

**Status:** In progress — the exact `VERSION=1.0.9 ./scripts/release-check.sh` gate passed at candidate `0f10d5d`, producing and verifying five-target checksummed archives, release metadata, dependency-license/SBOM input, and build identity. Source-install/version/help/config checks, OS-specific CI runtime smoke checks, unusual filename/rename coverage, semantic status colors, and `NO_COLOR` coverage are present; final cross-platform/manual acceptance, signed tagging, and publication remain.
