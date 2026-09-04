# Task 144: Implement bounded reflog loader and browser

**Phase:** Recovery and undo
**Depends on:** 125

## Goal

Expose reflog as a recovery surface and foundation for semantic undo.

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

1. Create `internal/reflog` loading ref selector, SHA, timestamp, actor and reflog subject/action using stable delimiters.
2. Default to HEAD reflog; allow selected local branch reflog.
3. Use bounded pages and cancellable loads.
4. Create Reflog workspace with commit inspection, compare-to-HEAD, create branch at entry and copy SHA.
5. Detached checkout uses existing explicit confirmation.
6. Do not add a casual hard-reset key.

## Git/process boundary

- `git reflog show --format=<machine-readable-delimiters> ...`

## Verification

- Rebase/reset/commit/checkout reflog fixtures, large reflog pagination.

## Acceptance criteria

- [ ] User can inspect and branch from recovery points safely.
- [ ] Parsing is not locale-dependent.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
