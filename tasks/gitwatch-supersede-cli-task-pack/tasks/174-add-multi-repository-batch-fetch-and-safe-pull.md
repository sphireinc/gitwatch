# Task 174: Add multi-repository batch fetch and safe pull

**Phase:** Multi-repository differentiation
**Depends on:** 125, 157, 159

## Goal

Make the repositories dashboard an operations console that goes beyond LZ’s single-repo focus.

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

1. Allow selected/group/all repositories batch fetch with a preview of exact repository set/remotes.
2. Add batch pull only with explicit strategies. Default recommendation is ff-only; merge/rebase batch modes require explicit opt-in and per-repo preflight.
3. Never batch-push by default; leave it out unless separately designed later.
4. Execute using bounded worker pool and existing per-repo write serialization.
5. Show per-repo queued/running/success/failure/skipped state live and retry failed subset.
6. One repository conflict/failure must not stop unrelated repositories; it becomes an attention row.
7. Local status refresh remains higher priority than batch network operations.

## Verification

- 50 disposable repos with clean/dirty/diverged/failure mix, cancel batch, process-count bound.

## Acceptance criteria

- [ ] Batch operations are observable, bounded and failure-isolated.
- [ ] Watcher responsiveness remains acceptable during batch work.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
