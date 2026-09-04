# Task 132: Complete rebase lifecycle recovery and continuation

**Phase:** Interactive rebase
**Depends on:** 127, 131, 143

## Goal

Make rebase durable: continue, skip, abort, restart recovery and conflict integration must work whether gitwatch or another terminal started it.

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

1. Expose Continue, Skip and Abort whenever Task 123 reports an active rebase and the action is valid.
2. Show current commit, completed/remaining count where recoverable, conflict count, and edit-stop state.
3. Route unresolved conflicts into the unified conflict resolver from Tasks 138-143.
4. After every lifecycle command request status + operation-state refresh and remain in the workflow until Git reports completion.
5. Derive truth from Git after restart; persist only UI preference/selection, not rebase truth.
6. On completion show old/new HEAD and rewritten count; add semantic operation-history record.

## Git/process boundary

- `git rebase --continue`
- `git rebase --skip`
- `git rebase --abort`

## Verification

- Conflict rebase, edit-stop, skip, abort, restart mid-rebase.
- External conflict resolution and continuation.

## Acceptance criteria

- [ ] Any active rebase can be resumed or aborted after restart.
- [ ] Conflict handling uses unified resolver.
- [ ] Completion refreshes status/history/branch divergence.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
