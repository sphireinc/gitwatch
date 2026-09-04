# Task 153: Integrate submodules with repository navigation and dashboards

**Phase:** Submodules
**Depends on:** 151, 125

## Goal

Make a submodule behave like a nested normal gitwatch repository while keeping superproject context.

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

1. Enter on initialized submodule opens it through normal repository/workspace model.
2. Show breadcrumb/back action to superproject.
3. Optionally expose initialized submodules as ephemeral registry entries; do not persist favorites/groups unless user chooses.
4. Parent repository row shows nested dirty/conflicted/in-progress attention.
5. Reuse normal watcher/operation/repository-switch cancellation semantics.
6. Do not create recursive unbounded watchers for every submodule.

## Verification

- Two-level nested navigation, operation in submodule while parent refreshes, missing path.

## Acceptance criteria

- [ ] Submodules reuse normal repository UI/engine.
- [ ] Nested navigation does not introduce unbounded watcher/process behavior.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
