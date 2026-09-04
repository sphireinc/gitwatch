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

- [ ] Every supported in-progress operation survives gitwatch restart.
- [ ] External CLI operation state appears on watcher/reconciliation refresh.
- [ ] No feature package hardcodes `.git` directory layout.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
