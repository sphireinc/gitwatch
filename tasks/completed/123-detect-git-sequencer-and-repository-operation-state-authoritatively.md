# Task 123: Detect Git sequencer and repository operation state authoritatively

**Phase:** Foundation
**Depends on:** 122

## Goal

Build one detector that reconstructs in-progress Git operations after startup, refresh, repository switch, crash, or external CLI activity.

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

1. Create typed Git-boundary code such as `internal/git/opstate.go` to detect merge, cherry-pick, revert, rebase and bisect state.
2. Use Git-supported commands wherever possible. Narrowly read Git metadata only when Git has no stable command; resolve the actual gitdir with Git so linked worktrees work.
3. Combine operation probes with the same generation as the authoritative porcelain-v2 status snapshot; never render operation state from generation N with worktree state from generation N+1.
4. Never assume gitwatch initiated the operation. A rebase/merge started in another terminal must appear automatically.
5. Unknown or newer Git metadata must degrade to `unknown operation in progress` with diagnostics, not crash or guess.
6. Document every Git metadata file read and add architecture tests preventing feature packages from crawling `.git` directly.

## Git/process boundary

- `git status --porcelain=v2 -z --branch --untracked-files=all`
- `git rev-parse --git-dir --show-toplevel`
- `git rev-parse --verify -q REBASE_HEAD / CHERRY_PICK_HEAD / REVERT_HEAD / MERGE_HEAD as applicable`
- `git bisect log when bisect state is suspected`

## Verification

- Real-repository fixtures for externally-started merge, rebase, cherry-pick, revert, bisect.
- Restart gitwatch mid-operation and reconstruct state.
- Linked worktree fixture where gitdir is not `<root>/.git`.

## Acceptance criteria

- [x] Every supported in-progress operation survives gitwatch restart: fresh
  detector calls reconstruct externally-created merge, cherry-pick, revert,
  rebase, and bisect fixtures.
- [x] External CLI operation state appears on watcher/reconciliation refresh:
  `repo.Snapshot` carries the operation projection at the same generation as
  porcelain-v2 status.
- [x] No feature package hardcodes `.git` directory layout; Git resolves all
  metadata paths through `rev-parse --git-path`.

## Status

Complete — added the authoritative Git-boundary operation detector, integrated
its repository-scoped projection and diagnostics into `repo.Snapshot`, and
covered known operations, unknown metadata, external operation recovery, and
linked worktrees with real repositories.

## Completion record

- [ ] Implementation commit recorded; to be filled with the focused commit
  after the task is moved.
- [x] Exact tested baseline recorded: current `main` at `0e81d3c` plus the
  Task 123 working-tree changes.
- [x] Focused real-repository tests: `go test ./internal/git -run
  'TestDetectOperationState'` and `go test -race ./internal/git ./internal/repo`
  passed for merge, cherry-pick, revert, rebase, bisect, unknown metadata,
  snapshot-generation coupling, and linked worktrees.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go test ./...`
  passed.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go test -race ./...`
  passed.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go vet ./...`
  passed.
- [x] `gofmt` and `git diff --check` passed.
- [ ] Lint: pinned `make check` lint could not download golangci-lint v2.12.0
  because `proxy.golang.org` is unavailable; the installed v2.11.3 binary is
  not a valid substitute and failed module loading.
- [x] Native/manual evidence not applicable: this task changes the internal
  refresh projection and does not add terminal interaction.
- [x] Known limitations/deferred work documented: the detector reports active
  projections and bounded diagnostics; UI presentation and lifecycle controls
  remain deferred to later tasks, and Git remains authoritative after every
  refresh.
