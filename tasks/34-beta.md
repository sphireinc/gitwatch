# Task 34 — Pre-v1 beta hardening

**Priority:** P0

Tag `v0.9.0`, distribute binaries, test in real repositories including monorepos and worktrees. Collect bugs specifically around missed refreshes, staging semantics, terminal rendering, Windows behavior, and large repositories. Freeze P0 feature additions after beta; only correctness/polish fixes.

**Acceptance:** Zero open blocker/critical issues and no known data-loss issue.

**Status:** In progress — repository discovery, status rendering, reversible stage/unstage and diff flows, explicit polling startup, non-repository diagnostics, strict five-target artifact verification, repeatable release checks, missing-Git classification, and real-repository coverage for spaces, Unicode, quotes, leading dashes, and renames are present. The operator matrix for candidate `5b2a8e9` records owner sign-off across all cells: macOS and Windows are attested green, and Linux is accepted by explicit equivalence while the expanded physical Linux testbed proceeds. The owner confirmed no known private/unreported blocker, critical, or data-loss issue on 2026-09-25; the public tracker showed zero open issues. Only the beta-wide carry-forward to current tested source `34562b5` remains pending because it changes merge-engine error propagation in the recovery path.

## Verified evidence (2026-09-25)

Source revision `a31cf80cce66ac95b83350fcfaae006e735fa1ef` passed hosted Actions run [36096868956](https://github.com/sphireinc/gitwatch/actions/runs/36096868956), including full-history secret scan, quality/policy, and Ubuntu 24.04, macOS 15, and Windows 2025 jobs. This is CI evidence, not native operator acceptance.

On Darwin arm64 with Git 2.33.0, `scripts/native-fixture.sh` prepared a repository containing staged and unstaged README changes, a rename from `space name.txt` to `renamed unicode-é.txt`, and an untracked path with a space. The binary built from the recorded revision. `scripts/pty-smoke.sh` passed startup, 80x24 help, and clean quit; `scripts/pty-large-status-smoke.sh` passed with 14,953 untracked files at 80x24. These are automated PTY checks and do not establish manual input, resize, rendering, or terminal-restoration acceptance.

Watcher regression checks also passed: `TestWatcherSeesExternalGitMetadataAndRecreatedDirectory` with `-count=30`, and with `-race -count=5`. Serena inspection confirmed the test waits for path-specific events after metadata-directory recreation. The GitHub open-issue query returned no issues at this check; that alone does not establish that there are no unreported or externally tracked blocker/critical issues.

**Still required before completion:** explicitly carry the candidate-specific all-cell operator disposition to current tested source `34562b5`, or keep the task evidence scoped to `5b2a8e9`. The matrix records owner sign-off for `5b2a8e9`; Linux is an accepted equivalence disposition, not a claim that physical Linux testing has already occurred. The blocker/data-loss review is complete as of 2026-09-25 based on the owner's confirmation and the public issue tracker; it does not claim that unknown future or unreported issues cannot exist.

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
blocker/data-loss issue exists. The matrix sign-off was recorded separately
after this automated check. Keep Task 34 in progress until the remaining beta
scenarios and blocker/data-loss review are complete.

## Current-host PTY supplement (2026-09-25)

Built `1.0.0-dev` from exact branch HEAD
`5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725` with commit and build date embedded.
On Darwin 25.6.0 arm64, Go 1.27.0, Git 2.33.0, and tmux 3.6a,
`scripts/native-fixture.sh` prepared the mixed staged/unstaged, rename, and
spaced-path repository; its porcelain-v2 inspection matched the intended state.
`scripts/pty-smoke.sh` passed startup, 120x32-to-80x24 resize, help rendering,
and clean quit. `scripts/pty-large-status-smoke.sh` also passed with exactly
14,953 untracked files and the authoritative count visible at 80x24. Redacted
fixture and PTY metadata were retained temporarily under a private `/tmp`
directory. These are automated tmux-PTY observations, not native operator
evidence. At the time of this supplement, the operator cells were pending; the
later owner sign-off for candidate `5b2a8e9` is recorded in the beta matrix.
Task 34 remains in progress for the other acceptance criteria.

## Hosted CI follow-up (2026-09-25)

GitHub Actions run [36104615360](https://github.com/sphireinc/git-watch/actions/runs/36104615360)
for exact branch HEAD `e4d8faf` completed successfully in 5m10s. The run
included quality/policy, full-history secret scanning, and all three platform
test-matrix jobs. This is hosted automated evidence only and does not establish
native operator acceptance. The subsequent owner sign-off for candidate
`5b2a8e9` is recorded separately in the beta matrix.

## Linked-worktree monorepo PTY follow-up (2026-09-25)

Built exact branch HEAD `354573e` with build identity embedded, then ran
`scripts/record-demo.sh` against a disposable linked worktree on Darwin 25.6.0
arm64, Go 1.27.0, Git 2.33.0, and tmux 3.6a. The fixture contained tracked
`apps/web` and `services/api` package trees plus `docs/notes.md`. At 80x24 with
filesystem watching, an external edit appeared without manual refresh and its
diff rendered; the selected file's staged count changed `0 → 1 → 0`. After
clean quit, porcelain-v2 showed exactly the three intended modified files,
`git diff --cached --exit-code` was clean, and `git diff --check` passed. The
linked worktree's `.git` indirection resolved through `git rev-parse`. This is
automated PTY evidence only; it does not establish native operator acceptance
or complete real-production-repository beta coverage.

The same binary also passed `scripts/pty-smoke.sh` against the actual gitwatch
checkout at worktree HEAD `53658fd`: startup, resize to 80x24, help, and clean
quit. The checkout remained clean afterward. The binary was built from
`354573e`; intervening changes were documentation-only. This remains scripted
PTY evidence, not native operator sign-off.

## Merged-main revalidation (2026-09-25)

After PR #9 merged, exact `main` revision
`be2d1d999ac3b3e62eafb59bd11b6f1703aac4c6` passed
`GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache make check`
locally on Darwin 26.6.2 arm64 with Go 1.27.0 and Git 2.33.0. Formatting,
pinned golangci-lint (0 issues), `go test ./...`, `go test -race ./...`,
`go vet ./...`, whitespace checks, security fuzz checks, and performance
budgets all passed.

Hosted Actions run
[36137333931](https://github.com/sphireinc/gitwatch/actions/runs/36137333931)
completed successfully on that exact merge commit. Quality/policy, full-history
secret scan, and Ubuntu 24.04, macOS 15, and Windows 2025 jobs passed. Ubuntu
and macOS passed race tests, Unix runtime smoke, and PTY/large-status
acceptance; Windows passed runtime smoke and path/CRLF parity (the Windows
race job is skipped by the workflow). The public GitHub issues page showed no
open issues at this check; this does not cover unreported or privately tracked
blockers.

This refresh confirms the merged code and automated gates only. Task 34 remains
in progress pending the remaining beta scenarios and explicit blocker/data-loss
review; the candidate-specific owner matrix sign-off remains recorded
separately above.

## Guarded-mutation audit supplement (2026-09-25)

On current `main` (`cd4bce2`), a focused source review traced the UI-to-Git
paths for worktree restore, branch deletion/reset, remote force-with-lease and
tag deletion, linked-worktree removal, stash apply/pop/drop, and merge recovery.
The reviewed UI paths require explicit confirmation for destructive actions;
branch deletion checks the exact branch name and refuses the current branch;
worktree removal is dispatched without `--force`; stash apply/pop use clean-
worktree checks; and recovery reset modes are limited to soft/mixed rather than
hard reset. Merge execution preflights dirty state and does not stash/reset
automatically. `TestMutationCompletionAlwaysRequestsAuthoritativeRefresh`
covers refresh requests for failed mutation completions across operation classes.

Uncached focused guard/refresh tests passed for app, branch, Git, merge, remote,
stash, tag, and worktree packages. The selected app tests covered explicit
remote-tag/force-push, stash, worktree, branch, and mutation-refresh flows. The
broader uncached app-package run could not bind its localhost `httptest` server
in this managed environment (`listen tcp6 [::1]:0: bind: operation not
permitted`); this is an environment limitation, not a reported application
assertion failure. The focused app tests passed separately. No actionable
data-loss or security defect was found in the reviewed paths; this bounded code
review does not establish that unknown issues cannot exist. Task 34's owner
blocker review and final source-scope disposition are recorded below; the task
remains in progress until the beta-wide carry-forward decision is resolved.

On 2026-09-25, the owner confirmed that no privately tracked or unreported
blocker, critical, or data-loss issue is known beyond the public tracker. The
[public issue list](https://github.com/sphireinc/gitwatch/issues) showed zero
open issues at this check. This closes the known-issue review as of this date;
the beta-wide carry-forward from candidate `5b2a8e9` to current tested source
`34562b5` remains pending explicit approval before Task 34 can be marked
complete.
