# Task 159: Add safe branch reset rebase and upstream reconciliation

**Phase:** Tags and remotes
**Depends on:** 158, 132

## Goal

Provide useful upstream/reset workflows without adding generic destructive hard reset.

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

1. Add rebase current branch onto selected local/remote branch through the rebase engine.
2. Add explicit fast-forward current branch where possible and set/unset upstream.
3. Allow soft/mixed HEAD/index reset workflows only where semantics are explicit and working-tree content is preserved.
4. Anything that would discard worktree content remains out of scope unless implemented as a separate reversible/file-scoped design.
5. Show ahead/behind and commits that will be rewritten before rebase.
6. Require published-history warning and journal the operation.

## Verification

- Fast-forward, diverged rebase, upstream set/unset, static assertion no generic hard-reset action.

## Acceptance criteria

- [ ] Common upstream recovery is possible without weakening safety policy.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
