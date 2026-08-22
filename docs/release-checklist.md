# Release checklist

This checklist separates repeatable automation from native human acceptance and publication. A passing local command is evidence only for the commit and host on which it ran.

## Automated candidate gate

Use Go 1.25.10 and the repository-pinned golangci-lint v2.12.0, then run the gate on the exact release commit. Record the versions with the evidence:

```sh
go version
cat .golangci-lint-version
make check
./scripts/secret-scan.sh --history
VERSION=1.0.0 ./scripts/release-check.sh
```

`go version` must report `go1.25.10`; `.golangci-lint-version` must report `v2.12.0`.

The automated gate covers formatting, lint, tests, race detection where supported, vet, security boundaries, representative performance budgets, an isolated source install, CLI/config inspection, a disposable real Git repository, cross-platform archive creation, dependency-license packaging, extraction, embedded build identity, and checksums.

Tagged pushes matching `v*.*.*` run `.github/workflows/release.yml`. The workflow builds macOS amd64/arm64, Linux amd64/arm64, and Windows amd64 archives, verifies their contents and `SHA256SUMS`, produces an SBOM and build provenance, and publishes through the protected release environment.

## Native operator acceptance

Record the commit/tag, OS and architecture, terminal and dimensions, Git version, watch mode, and evidence link for every run. Every workflow row below must pass separately on macOS, Linux, and native Windows; a pass on one platform does not carry to another. Optional integrations must be exercised both disabled/unavailable and enabled with disposable, non-secret test data.

### Platform completion

- [ ] macOS: every core, workbench, integration, accessibility, installation, and shutdown row below passes on the exact candidate.
- [ ] Linux: every row passes on the exact candidate.
- [ ] Windows: every row passes on the exact candidate using native paths and terminal behavior.

### Core status and mutation workflows

- [ ] Launch in clean and mixed-state repositories; verify branch/upstream/divergence, status counts, activity, watcher mode, forced refresh, and clean-state presentation.
- [ ] Select files by keyboard and mouse; filter and sort status rows; open staged and unstaged diffs/details in wide and narrow layouts; verify binary, rename, conflict, and stale-result behavior.
- [ ] Stage and unstage one path and the explicit bulk scope; verify unusual paths and an authoritative refresh after every success, failure, cancellation, or conflict result.
- [ ] Exercise guarded restore/discard confirmation and cancellation; prove a cancelled action changes nothing and a confirmed action names only the intended path/content.
- [ ] Stage, unstage, and discard selected hunks/lines; verify separated hunks, CRLF text, and safe refusal of unsupported binary/rename/copy partial patches.
- [ ] Trigger external create, edit, delete, rename, index, commit, branch, and merge-state changes in filesystem mode and polling mode; verify visible fallback and reconciliation without blocking input.

### Advanced workbench workflows

- [ ] Commit composer: staged-scope preview, subject/body editing, validation, normal commit, hook failure with preserved draft/output, amend confirmation, metadata toggles, and authoritative refresh.
- [ ] Stashes: list/preview, create with and without untracked files, apply, pop, confirmed drop, conflict/error reporting, and post-operation refresh.
- [ ] Branches: search/sort, switch, create, rename, set/unset upstream, normal/force deletion confirmations, divergence/merged state, and linked-worktree occupancy.
- [ ] Worktrees: list metadata, add, open, confirmed remove, prune, locked/prunable behavior, branch occupancy, and return/switch behavior.
- [ ] History: bounded paging, search, graph/refs, commit inspection, parent/path filtering, tag loading, confirmed checkout, branch-at-commit, and exact-SHA revert.
- [ ] Remotes: redacted URL display, fetch, each explicit pull strategy, normal/tracking/tag push, force-with-lease confirmation, progress/cancellation, conflicts, authentication failures, and refresh.
- [ ] Command palette and notifications: search/ranking, disabled reasons, navigation/actions, async completion, conflict/hook/remote/plugin notices, dismissal, quiet mode, and reduced/off motion.

### Optional integrations and multi-repository workflows

- [ ] GitHub disabled/offline behavior leaves core Git usable; enabled behavior shows current-branch PR/check/review state, cache refresh, sanitized errors, validated browser opening, and URL copying without credential disclosure.
- [ ] Multi-repository dashboard: bounded discovery, filtering/sorting, favorites/groups, stale/error state, concurrent refresh limits, repository switching, linked worktrees, and persisted private metadata.
- [ ] Plugins: manifest discovery, version-one protocol compatibility, permission display, enable/disable/reload, command and widget state, malformed or hostile output, timeout/crash isolation, and a public SDK example against the candidate host.

### Environment, edge cases, and lifecycle

- [ ] Keyboard and mouse parity, command help, responsive resizing, supported minimum size, color and `NO_COLOR`, high-contrast-safe semantics, and full/reduced/off motion.
- [ ] Linked worktree, submodule, detached HEAD, unborn branch, no upstream, conflict, rename, Unicode, spaces, tabs, quotes, leading hyphen, long supported path, symlink, nested repository, and large-worktree cases.
- [ ] Configuration validation/inspection, environment and CLI precedence, schema migration, invalid/future configuration diagnostics, and keybinding-collision rejection.
- [ ] Clean-machine archive installation and source installation, including version/help output and the packaged license/notices.
- [ ] Missing Git, bare repository, non-repository, unavailable provider/plugin, cancellation, timeout, and recoverable Git failure diagnostics.
- [ ] No orphan watcher, plugin, provider, or Git child process and no altered terminal state after normal quit, Ctrl-C, repository switch, cancellation, startup failure, or operation failure.
- [ ] No open release blocker, critical security issue, known data-loss defect, or unresolved license concern.

CI runtime smoke tests prove startup and basic Git discovery on each runner OS. They do not replace native mouse, resize, filesystem notification, terminal restoration, or clean-machine acceptance.

Use the [native harness](native-harness.md) for those operator-owned checks;
capture only sanitized evidence and remove temporary fixtures after each run.

Tasks 34, 35, 89, and 90 remain explicitly in progress. This checklist and the [beta validation matrix](beta-validation-matrix.md) record their outstanding operator evidence; documentation changes alone do not complete those tasks.

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
