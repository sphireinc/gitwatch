# Beta validation matrix

This file is the evidence sheet for the first stable release and full-workbench
beta. Automated checks run in CI and `./scripts/release-check.sh`;
operator-owned rows must identify the exact commit/tag and their evidence
provenance. An owner-approved platform-equivalence decision may be recorded as
an acceptance exception, but must not be described as a platform run that did
not occur.

The latest hosted automation check for candidate `e4d8faf` (2026-09-25),
[Actions run 36104615360](https://github.com/sphireinc/git-watch/actions/runs/36104615360),
passed quality/policy, full-history secret scanning, and the three-platform
test matrix. It does not itself establish operator acceptance. The later
owner sign-off below is specific to candidate `5b2a8e9`.

## Candidate `5b2a8e9` operator sign-off (2026-09-25)

For exact commit
[`5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725`](https://github.com/sphireinc/gitwatch/commit/5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725),
the user provided an explicit operator sign-off that all listed areas pass on
macOS and Windows. The user also directed that all Linux cells be accepted as
passing by platform equivalence to macOS while an expanded physical Linux
testbed covering multiple distributions is in progress. Accordingly, every
cell below is marked `pass¹` for this candidate.

This is owner-provided acceptance provenance, not an independently captured
per-row transcript. In particular, the Linux `pass¹` cells do not claim that a
physical Linux operator run has already occurred. Reopen any affected cell if
the expanded Linux testing identifies a regression.

An exact-HEAD scripted PTY run on `354573e` exercised a linked worktree with
`apps/web` and `services/api` subtrees, external filesystem refresh, diff
rendering, and a stage/unstage round trip. Final porcelain-v2 confirmed only
the three intended modified files and no staged residue. This automated
fixture evidence did not itself replace operator observation; the later
candidate-specific owner sign-off is recorded above.

The same binary also passed the read-only `scripts/pty-smoke.sh` on the actual
gitwatch checkout at `53658fd`; the working tree remained clean. That check
covers scripted startup, 80x24 resize/help, and quit only.

Use Go 1.25.10 and golangci-lint v2.12.0 for candidate-gate evidence. Each
manual cell applies only to the exact candidate commit recorded with its
evidence; an observation from another or unidentified build remains pending.

Automated evidence for candidate `c0a7d46` (2026-08-28) passes the pinned
lint, tests, race tests, vet, security checks, performance budgets, and the
full-history secret scan. That evidence did not change cells for that earlier
candidate; the later owner sign-off above is specific to `5b2a8e9`.

The reproducible fixture harness was also run on 2026-08-28 on Darwin arm64
with Git 2.33.0. Its bounded capture covered mixed index/worktree changes, a
rename containing Unicode, spaces, and an untracked path, and reset cleanly.
This validates the fixture and Git-state assertions only; it does not close
the native gitwatch rendering, input, resize, or terminal-restoration rows.

| Area | macOS | Linux | Windows | Evidence required |
| --- | --- | --- | --- | --- |
| Clean install, build, launch, version/help | pass¹ | pass¹ | pass¹ | archive and source install transcript; exact tool versions |
| Status dashboard and clean state | pass¹ | pass¹ | pass¹ | clean, modified, staged, untracked, branch/divergence recording |
| File selection, filter/sort, details, and diff | pass¹ | pass¹ | pass¹ | keyboard and mouse; staged/unstaged, binary, rename, conflict; wide/narrow layouts |
| Single and bulk stage/unstage | pass¹ | pass¹ | pass¹ | before/after Git status and authoritative refresh on success/failure |
| Guarded restore/discard | pass¹ | pass¹ | pass¹ | cancel/no-change and exact-scope confirmed action evidence |
| Hunk/line stage, unstage, and discard | pass¹ | pass¹ | pass¹ | separated hunks, CRLF, unsupported binary/rename/copy refusal |
| Filesystem watch, reconciliation, and polling | pass¹ | pass¹ | pass¹ | external file/index/ref/merge changes, visible fallback, responsive input |
| Commit composer and execution | pass¹ | pass¹ | pass¹ | validation, normal commit, hook failure/draft retention, amend confirmation, refresh |
| Stash manager | pass¹ | pass¹ | pass¹ | preview, create/include-untracked, apply/pop/drop, conflict and refresh |
| Branch manager | pass¹ | pass¹ | pass¹ | search/sort, switch/create/rename/upstream/delete, occupancy and confirmations |
| Worktree manager | pass¹ | pass¹ | pass¹ | list/add/open/remove/prune, locked/prunable state and branch occupancy |
| History and commit inspector | pass¹ | pass¹ | pass¹ | paging/search/graph, parent/path inspection, tags, checkout/branch/revert guards |
| Remote workflows | pass¹ | pass¹ | pass¹ | redacted URLs, fetch, pull strategies, push/tracking/tag/force-with-lease, cancel/failure |
| GitHub integration enabled/disabled | pass¹ | pass¹ | pass¹ | graceful unavailable state; PR/check/review, cache, open/copy, sanitized failure |
| Multi-repository dashboard | pass¹ | pass¹ | pass¹ | bounded discovery/refresh, filter/sort, favorites/groups, switch and persistence |
| Plugin host and public SDK | pass¹ | pass¹ | pass¹ | API-1 example, permissions, enable/reload, widget/command state, crash/timeout isolation |
| Command palette and notifications | pass¹ | pass¹ | pass¹ | search/disabled reasons/actions; notice delivery/dismissal/quiet mode |
| Configuration and keybindings | pass¹ | pass¹ | pass¹ | validation/inspection, precedence/migration, invalid/future config and collisions |
| Resize, terminal capability, and accessibility | pass¹ | pass¹ | pass¹ | keyboard/mouse parity, minimum size, `NO_COLOR`, contrast, full/reduced/off motion |
| Repository/path edge cases | pass¹ | pass¹ | pass¹ | linked worktree, submodule, detached/unborn, conflict, unusual names, symlink, nesting |
| Missing dependency and failure diagnostics | pass¹ | pass¹ | pass¹ | Git/non-repo/bare/provider/plugin/cancel/timeout exit state and message |
| Shutdown and repository switching | pass¹ | pass¹ | pass¹ | no child/goroutine leak or terminal damage after quit, Ctrl-C, switch, and failures |
| Large repository/workbench workloads | pass¹ | pass¹ | pass¹ | candidate `make check` performance output plus native responsiveness evidence |

¹ Owner-provided sign-off for candidate `5b2a8e9`; Linux is accepted by the
explicit equivalence decision described above while physical multi-distribution
testing continues.

Use `./scripts/demo-repo.sh` to create the disposable fixture. Record the
terminal, OS version, terminal emulator, Git version, commit under test, and
whether every row passed. A `pending` row is not a release sign-off.

The lower-left context-pane family additionally requires disabled/enabled,
commit-tree/unpushed/branch-summary switching, wide/narrow, resize, scroll,
external-commit/ref changes, and keyboard/mouse evidence. Use the bounded
100-commit defaults unless a configured limit is explicitly recorded.

Colorized commit-tree evidence normally additionally covers dark, light,
high-contrast, and `NO_COLOR` terminals, decorated merge graphs, malformed
fixture output, visible-width bounds, and absence of raw controls. Automated
parser/theme tests alone do not replace native visual evidence; the current
candidate's matrix disposition follows the owner sign-off above.

For repeatable native evidence, use [`native-harness.md`](native-harness.md),
`scripts/native-fixture.sh`, and `scripts/native-capture.sh`. The fixture
asserts Git state through porcelain output; the operator records rendering,
input, resize, process cleanup, and terminal restoration separately.

Historical macOS keyboard-launch/diff observations are recorded in
[`operator-macos.md`](operator-macos.md), but that session did not preserve an
exact tested commit. It is useful context, not release-candidate evidence; the
current `pass¹` disposition comes from the separate owner sign-off for
`5b2a8e9` above.

Optional GitHub and plugin rows require both graceful disabled/unavailable
behavior and enabled behavior with disposable, non-secret data. Automated unit,
integration, race, security, and performance checks support these rows but do
not by themselves replace native terminal observation. The current candidate's
owner-approved disposition is recorded above.

Tasks 34, 35, 89, and 90 remain in progress until their own acceptance criteria
and the applicable rows above are complete. Do not move or relabel them based
only on documentation or automated evidence.
