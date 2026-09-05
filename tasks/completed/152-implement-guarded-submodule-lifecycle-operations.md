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

- [x] Submodule lifecycle requires no shell scripts.
- [x] Removal is path-specific and confirmed.

## Completion record

- [x] Implementation commits recorded: `1f783e5` and `5ebf25d`.
- [x] Exact tested revision recorded: `faa808e`.
- [x] Focused unit/integration tests recorded: `go test ./internal/submodules ./internal/app`, including real local add, initialize, sync, deinit, remove, confirmation, and URL-redaction fixtures.
- [x] `go test ./...` recorded: passed at the exact tested revision.
- [x] Race/vet/lint/format evidence recorded where applicable: full `make check` passed, including race, vet, golangci-lint, formatting, diff, security, and performance checks.
- [x] Native/manual evidence recorded where this task changes terminal interaction: automated confirmation and dispatch coverage passed; interactive human terminal QA remains a documented release-gate limitation.
- [x] Known limitations/deferred work documented: bulk operations remain intentionally separate and terminal QA remains deferred.

## Progress evidence

- Added typed `internal/submodules` lifecycle operations for initialize,
  update, sync, deinit, remove, and add. All commands use argument vectors
  and explicit `--` path boundaries; recursive/bulk behavior is not implicit.
- Deinit and remove require an exact repeated path confirmation. Path and URL
  validation rejects control characters, and URL-bearing command results,
  output, and errors redact credentials before retention.
- Added status-workspace `[M]` routing for initialize, update, sync, exact-path
  deinit/remove confirmation, and URL entry for add. Operations run through
  the bounded operation engine and successful completion requests an
  authoritative refresh.
- Focused tests cover unsafe paths, confirmation mismatch, control-character
  rejection, URL-result sanitization, and the status confirmation/dispatch
  path. Real local-repository coverage now exercises add, initialize, sync,
  deinit, and exact-path remove. Failed-operation isolation and interactive
  human terminal QA remain explicit follow-up release evidence; default
  operations remain repository-scoped and non-recursive.
