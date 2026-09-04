# Task 152: Implement guarded submodule lifecycle operations

**Phase:** Submodules
**Depends on:** 151

## Goal

Match practical submodule lifecycle features with explicit deletion and URL-change safety.

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

1. Add initialize, update, sync, add, deinit/remove and URL update.
2. Never recursively delete a submodule directory via generic filesystem command; use Git-supported lifecycle operations.
3. Require exact path/name confirmation for destructive remove/deinit flows.
4. URL entry/display must redact credentials after submission.
5. Recursive/bulk behavior is a separate explicit action, not implicit default.
6. Refresh both superproject and nested repository state after operations.

## Git/process boundary

- `git submodule update --init -- <path>`
- `git submodule sync -- <path>`
- `git submodule deinit -- <path>`
- `git rm -- <path>`
- `git submodule add <url> <path>`

## Verification

- Lifecycle using local bare remotes, removal confirmation, failed update isolation.

## Acceptance criteria

- [ ] Submodule lifecycle requires no shell scripts.
- [ ] Removal is path-specific and confirmed.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
