# Task 135: Build cherry-pick progress workspace

**Phase:** Cherry-pick and history selection
**Depends on:** 134, 143

## Goal

Provide visible progress and recovery instead of reducing multi-commit cherry-pick to a toast.

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

1. Create a workspace showing selected commits in application order with pending/current/completed/skipped/conflicted states.
2. Show source ref, target branch, original HEAD, current/result HEAD and remaining count.
3. Expose Continue/Skip/Abort only when current Git state allows them.
4. On conflict route to unified conflict resolver and return to progress after resolution.
5. Keep the status summary live and allow navigation away/back without losing the active operation.
6. Allow Ctrl-P to reopen any active cherry-pick for the selected repository.

## Verification

- 80x24 and wide layouts.
- Conflict at first/middle/last selected commit.
- Navigate to another workspace and back.

## Acceptance criteria

- [ ] User can see exactly which commit failed and what remains.
- [ ] No modal traps the user.
- [ ] External resolution appears automatically.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
