# Task 177: Add background remote intelligence and auto-fetch

**Phase:** Multi-repository differentiation
**Depends on:** 174, 175

## Goal

Provide optional remote awareness without surprising history changes or compromising live local status.

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

1. Add conservative configurable auto-fetch interval per profile/group, preferably disabled or modest by default.
2. Use bounded workers and per-repo jitter to avoid remote thundering herd.
3. Never auto-pull, auto-rebase or auto-push.
4. Back off when offline/auth failing/rate-limited and skip repositories with active sequencer/history rewrite where appropriate.
5. After fetch, update ahead/behind and remote-health metadata.
6. Local filesystem/status refresh has higher scheduling priority than background network work.
7. Expose last auto-fetch result in repository health/timeline.

## Verification

- 20 repos with concurrency cap, offline/backoff, active rebase skipped, cancellation.

## Acceptance criteria

- [ ] Auto-fetch improves awareness without modifying worktree/history.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
