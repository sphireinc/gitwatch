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

## Linked-worktree PTY follow-up (2026-09-25)

The existing PTY scripts incorrectly required `<repo>/.git` to be a directory. Reproduction against a valid `git worktree add` linked tree failed with “repository is not initialized” even though `git rev-parse --git-dir` succeeded. `scripts/pty-smoke.sh` and `scripts/record-demo.sh` now validate through `git rev-parse`, and `docs/native-harness.md` documents linked-worktree use.

Built from `9b7acaa` on Go 1.27.0, Darwin 25.6.0 arm64, Git 2.33.0, and tmux 3.6a. The updated 80x24 PTY startup/help/quit smoke passed in the linked worktree. The scripted PTY demo also passed there with filesystem watch enabled: an external edit appeared in status and diff without pressing refresh; the displayed counts transitioned from modified to staged and back to modified; final porcelain status showed only the intended unstaged `docs/notes.md` change, with no staged residue. The script resized the session and exited cleanly. This is automated PTY evidence only and does not fill native operator matrix cells.

`GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache make check` passed on Darwin arm64, including pinned golangci-lint (0 issues), formatting, full tests, race tests, vet, whitespace checks, security fuzzing, and performance budgets. The earlier sandbox-only lint download failure was resolved by allowing the pinned module download; no lint finding was suppressed.

## Latest automated evidence (2026-09-25)

At commit `6e0c06a403e0f9909e019c7959d7fea5d88c2775`, the complete local
`make check` passed on Darwin arm64 / Apple M1 Pro, including formatting,
pinned lint (0 issues), full and race tests, vet, security fuzz checks, and
performance budgets. Hosted Actions run [36103006507](https://github.com/sphireinc/gitwatch/actions/runs/36103006507)
passed its quality/policy, full-history secret scan, and Ubuntu 24.04, macOS
15, and Windows 2025 test/build matrix. Its Linux and macOS jobs also passed
the large-status PTY acceptance; Windows path/CRLF parity passed. The open
GitHub issue query returned no issues at this check. This is current automated
and issue-tracker evidence only; it does not establish that no unreported
blocker/data-loss issue exists and does not fill any native operator matrix
cell. Keep Task 34 in progress until the remaining beta scenarios and exact
candidate native evidence are recorded.
