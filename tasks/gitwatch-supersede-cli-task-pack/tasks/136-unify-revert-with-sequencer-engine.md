# Task 136: Unify revert with sequencer engine

**Phase:** Cherry-pick and history selection
**Depends on:** 123, 143

## Goal

Upgrade existing revert into a resumable multi-commit operation with the same recovery/conflict semantics as cherry-pick.

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

1. Move revert execution behind a typed sequencer/revert adapter.
2. Support selected commit sets with an explicit order preview and warning that order matters.
3. Require explicit mainline parent for reverting merge commits.
4. Expose continue/skip/abort when Git enters revert sequencer state.
5. Reuse conflict workspace and operation timeline.
6. Keep the existing exact-SHA confirmation or replace it only with an equally explicit commit/range confirmation.

## Git/process boundary

- `git revert <sha>...`
- `git revert -m <parent> <merge-sha>`
- `git revert --continue`
- `git revert --skip`
- `git revert --abort`

## Verification

- Single/multiple revert, merge revert, conflict, abort/restart.

## Acceptance criteria

- [ ] Revert uses the same durable operation lifecycle as rebase/cherry-pick.
- [ ] No one-off conflict parser remains.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
