# Task 34 — Pre-v1 beta hardening

**Priority:** P0

Tag `v0.9.0`, distribute binaries, test in real repositories including monorepos and worktrees. Collect bugs specifically around missed refreshes, staging semantics, terminal rendering, Windows behavior, and large repositories. Freeze P0 feature additions after beta; only correctness/polish fixes.

**Acceptance:** Zero open blocker/critical issues and no known data-loss issue.

**Status:** In progress — repository discovery, status rendering, reversible stage/unstage and diff flows, explicit polling startup, non-repository diagnostics, strict five-target artifact verification, repeatable release checks, missing-Git classification, and real-repository coverage for spaces, Unicode, quotes, leading dashes, and renames are present; the operator matrix in `docs/beta-validation-matrix.md` still requires full macOS, Linux, and native Windows beta evidence.

## Verified evidence (2026-09-25)

Source revision `a31cf80cce66ac95b83350fcfaae006e735fa1ef` passed hosted Actions run [36096868956](https://github.com/sphireinc/gitwatch/actions/runs/36096868956), including full-history secret scan, quality/policy, and Ubuntu 24.04, macOS 15, and Windows 2025 jobs. This is CI evidence, not native operator acceptance.

On Darwin arm64 with Git 2.33.0, `scripts/native-fixture.sh` prepared a repository containing staged and unstaged README changes, a rename from `space name.txt` to `renamed unicode-é.txt`, and an untracked path with a space. The binary built from the recorded revision. `scripts/pty-smoke.sh` passed startup, 80x24 help, and clean quit; `scripts/pty-large-status-smoke.sh` passed with 14,953 untracked files at 80x24. These are automated PTY checks and do not establish manual input, resize, rendering, or terminal-restoration acceptance.

Watcher regression checks also passed: `TestWatcherSeesExternalGitMetadataAndRecreatedDirectory` with `-count=30`, and with `-race -count=5`. Serena inspection confirmed the test waits for path-specific events after metadata-directory recreation. The GitHub open-issue query returned no issues at this check; that alone does not establish that there are no unreported or externally tracked blocker/critical issues.

**Still required before completion:** complete and attach operator evidence for the macOS, Linux, and native Windows rows in `docs/beta-validation-matrix.md`, including exact commit, terminal/emulator, dimensions, interaction/resize behavior, and terminal restoration; test the outstanding real-repository scenarios (including worktrees/monorepos and staging/refresh behavior); and explicitly establish that no blocker/critical or known data-loss issue remains. Do not treat CI, PTY, or an empty GitHub issue list as substitutes for these acceptance checks.
