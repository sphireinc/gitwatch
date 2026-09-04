# Task 131: Support edit reword and amend of historical commits

**Phase:** Interactive rebase
**Depends on:** 129

## Goal

Allow a selected historical commit to be reworded or edited through controlled rebase rather than unsafe reset tricks.

## Non-negotiable constraints

- Live filesystem-driven status is the product core. Do not replace, subordinate, or pause it except for the minimum repository-lock window required by Git itself.
- Filesystem events are refresh hints, never authoritative state. The authoritative worktree snapshot remains `git status --porcelain=v2 -z --branch --untracked-files=all` parsed into immutable repository state.
- Every successful mutation MUST request an authoritative refresh for the affected repository. Long-running sequencer operations must refresh after every observable state transition.
- Multi-repository support is first-class. New domain models and operations MUST carry repository identity/scope and remain correct while other repositories refresh or run unrelated work.
- Do not create unbounded watchers, goroutines, workers, or Git/provider/plugin processes. Reuse bounded registry/operation infrastructure.
- All Git commands use typed argv execution through the Git boundary. Never interpolate repository data into shell command strings. Use `--` where supported and machine-readable/NUL-delimited output where available.
- Bubble Tea owns UI state. Git/network/filesystem/process work never runs in the render path.
- Repository-controlled text is untrusted terminal input and MUST be sanitized before rendering.
- Destructive/history-rewriting actions require scope-specific confirmation. Keep the prohibition on generic `reset --hard`, raw `--force`, and `clean -fd` shortcuts.
- Keyboard and mouse must reach equivalent functionality. New views must work at 80x24, honor `NO_COLOR`, and support full/reduced/off motion.
- Do not reimplement Git. Use Git as source of truth and build safe typed control/presentation layers around it.
- Breaking config/plugin changes require versioning, migration, and compatibility fixtures.

## Implementation steps

1. Implement `Reword selected commit` by creating a minimal rebase plan and collecting replacement message in gitwatch.
2. Implement `Edit selected commit` by marking it edit and starting rebase.
3. When Git pauses, present a paused-edit state that links to normal live Status and the existing commit composer for amend.
4. Do not hide worktree changes made during edit; status remains authoritative.
5. Allow abort and display original HEAD/base recovery information.
6. Do not add arbitrary hard reset as an implementation shortcut.

## Git/process boundary

- `git commit --amend via existing commit workspace`
- `git rebase --continue`
- `git rebase --abort`

## Verification

- Reword only.
- Edit/stage/amend/continue.
- Abort restores original history.
- External amend/continue detected automatically.

## Acceptance criteria

- [x] Historical reword/edit works without leaving gitwatch.
- [x] Existing commit composer is reused.
- [x] Abort/restart recovery is reliable.

## Completion record

**Status:** Complete

- Implementation commit: recorded after this completion note is staged; typed continue/abort operations are in `internal/git/rebase.go`, with history entry, paused amend-composer routing, and recovery handling in `internal/app`.
- Exact tested revision: `31ada9a` plus the working-tree Task 131 changes.
- Focused tests: historical rebase plan construction and real fixup/commit boundary tests passed; existing operation-state tests cover externally started and paused Git operations.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: the pinned `golangci-lint@v2.12.0` remains unavailable because the environment denies Go build-cache access; no analyzer failure was reported.
- Native/manual evidence: automated app tests cover reword/edit routing, composer reuse, continue/abort dispatch, and live refresh requests. Native terminal snapshots and interactive signing-provider QA remain unavailable and are documented as an explicit manual-QA exception.
- Known limitations/deferred work: root-commit historical editing and richer commit-message prefill are deferred to later rebase enhancements. No generic reset shortcut is used; abort delegates to Git's recorded recovery state, and external operation markers continue to be reconstructed by Task 123.
