# Beta validation matrix

This file is the evidence sheet for the first stable release and full-workbench
beta. Automated checks run in CI and `./scripts/release-check.sh`;
operator-owned rows must include the exact commit/tag and command output or
recording link before a public release is accepted.

Use Go 1.25.10 and golangci-lint v2.12.0 for candidate-gate evidence. Each
manual cell applies only to the exact candidate commit recorded with its
evidence; an observation from another or unidentified build remains pending.

Automated evidence for candidate `5df17b9` (2026-08-22) passes Go 1.27 pinned
lint, tests, race tests, vet, security checks, performance budgets, and the
full-history secret scan. This does not change any native matrix cell below;
native interaction and clean-machine evidence must be recorded on the exact
candidate by a maintainer.

| Area | macOS | Linux | Windows | Evidence required |
| --- | --- | --- | --- | --- |
| Clean install, build, launch, version/help | pending | pending | pending | archive and source install transcript; exact tool versions |
| Status dashboard and clean state | pending | pending | pending | clean, modified, staged, untracked, branch/divergence recording |
| File selection, filter/sort, details, and diff | pending | pending | pending | keyboard and mouse; staged/unstaged, binary, rename, conflict; wide/narrow layouts |
| Single and bulk stage/unstage | pending | pending | pending | before/after Git status and authoritative refresh on success/failure |
| Guarded restore/discard | pending | pending | pending | cancel/no-change and exact-scope confirmed action evidence |
| Hunk/line stage, unstage, and discard | pending | pending | pending | separated hunks, CRLF, unsupported binary/rename/copy refusal |
| Filesystem watch, reconciliation, and polling | pending | pending | pending | external file/index/ref/merge changes, visible fallback, responsive input |
| Commit composer and execution | pending | pending | pending | validation, normal commit, hook failure/draft retention, amend confirmation, refresh |
| Stash manager | pending | pending | pending | preview, create/include-untracked, apply/pop/drop, conflict and refresh |
| Branch manager | pending | pending | pending | search/sort, switch/create/rename/upstream/delete, occupancy and confirmations |
| Worktree manager | pending | pending | pending | list/add/open/remove/prune, locked/prunable state and branch occupancy |
| History and commit inspector | pending | pending | pending | paging/search/graph, parent/path inspection, tags, checkout/branch/revert guards |
| Remote workflows | pending | pending | pending | redacted URLs, fetch, pull strategies, push/tracking/tag/force-with-lease, cancel/failure |
| GitHub integration enabled/disabled | pending | pending | pending | graceful unavailable state; PR/check/review, cache, open/copy, sanitized failure |
| Multi-repository dashboard | pending | pending | pending | bounded discovery/refresh, filter/sort, favorites/groups, switch and persistence |
| Plugin host and public SDK | pending | pending | pending | API-1 example, permissions, enable/reload, widget/command state, crash/timeout isolation |
| Command palette and notifications | pending | pending | pending | search/disabled reasons/actions; notice delivery/dismissal/quiet mode |
| Configuration and keybindings | pending | pending | pending | validation/inspection, precedence/migration, invalid/future config and collisions |
| Resize, terminal capability, and accessibility | pending | pending | pending | keyboard/mouse parity, minimum size, `NO_COLOR`, contrast, full/reduced/off motion |
| Repository/path edge cases | pending | pending | pending | linked worktree, submodule, detached/unborn, conflict, unusual names, symlink, nesting |
| Missing dependency and failure diagnostics | pending | pending | pending | Git/non-repo/bare/provider/plugin/cancel/timeout exit state and message |
| Shutdown and repository switching | pending | pending | pending | no child/goroutine leak or terminal damage after quit, Ctrl-C, switch, and failures |
| Large repository/workbench workloads | pending | pending | pending | candidate `make check` performance output plus native responsiveness evidence |

Use `./scripts/demo-repo.sh` to create the disposable fixture. Record the
terminal, OS version, terminal emulator, Git version, commit under test, and
whether every row passed. A `pending` row is not a release sign-off.

For repeatable native evidence, use [`native-harness.md`](native-harness.md),
`scripts/native-fixture.sh`, and `scripts/native-capture.sh`. The fixture
asserts Git state through porcelain output; the operator records rendering,
input, resize, process cleanup, and terminal restoration separately.

Historical macOS keyboard-launch/diff observations are recorded in
[`operator-macos.md`](operator-macos.md), but that session did not preserve an
exact tested commit. It is useful context, not release-candidate evidence, so
all macOS cells remain pending.

Optional GitHub and plugin rows require both graceful disabled/unavailable
behavior and enabled behavior with disposable, non-secret data. Automated unit,
integration, race, security, and performance checks support these rows but do
not replace native terminal observation.

Tasks 34, 35, 89, and 90 remain in progress until their own acceptance criteria
and the applicable rows above are complete. Do not move or relabel them based
only on documentation or automated evidence.
