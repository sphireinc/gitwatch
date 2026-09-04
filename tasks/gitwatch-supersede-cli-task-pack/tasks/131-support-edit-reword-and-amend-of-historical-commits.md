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

- [ ] Historical reword/edit works without leaving gitwatch.
- [ ] Existing commit composer is reused.
- [ ] Abort/restart recovery is reliable.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
