# Task 146: Implement safe semantic undo

**Phase:** Recovery and undo
**Depends on:** 145, 143

## Goal

Offer Undo only where gitwatch can prove a safe recovery point; refuse when repository state has diverged.

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

1. Define undo policy per operation instead of a generic reset button.
2. Examples: abort active sequencer; undo a just-created local commit while preserving worktree/index via soft/mixed semantics; restore pre-rebase branch ref only when reflog/current-state checks prove no newer work is lost.
3. Before undo compare current HEAD/index/worktree assumptions with recorded post-operation state. If diverged, refuse automatic undo and open guided recovery.
4. Never use unconditional `reset --hard`.
5. Require confirmation naming old/new HEAD/ref and effect on index/worktree.
6. Execute through operation engine, refresh authoritative state, and record undo as a journal entry.

## Git/process boundary

- `Operation-specific git reset --soft/--mixed only where policy proves safety`
- `git rebase/merge/cherry-pick/revert --abort for active operations`

## Verification

- Undo immediately after local commit, refusal after unrelated commit/change, sequencer abort, multi-repo isolation.

## Acceptance criteria

- [ ] Undo preserves working content by default.
- [ ] Stale/unsafe recovery is refused rather than guessed.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
