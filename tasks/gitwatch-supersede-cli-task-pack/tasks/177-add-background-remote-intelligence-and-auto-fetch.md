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

## Progress evidence (2026-09-21)

- Added `internal/remoteintel` with opt-in bounded auto-fetch scheduling, fixed worker concurrency, deterministic per-repository jitter, cancellation-aware waits, active-operation skipping, and exponential failure backoff.
- Added configuration fields for disabled-by-default auto-fetch interval, jitter, backoff, and backoff maximum; validation and schema/documentation coverage are included.
- Added optional `repositories.group_auto_fetch` intervals; when several groups apply, the scheduler uses the most conservative configured interval.
- Integrated the scheduler with the Bubble Tea lifecycle. Auto-fetch runs outside render/update paths, uses typed discovery/status/remote/fetch boundaries, never pulls/rebases/pushes, and refreshes the repository dashboard after successful fetches.
- Persisted the latest in-process auto-fetch result into repository dashboard rows, including visible success/failure/skipped state and stable offline/authentication/rate-limit/remote-error categories; failures contribute a warning without masking critical local conflicts.
- Persisted last auto-fetch timestamp, status, error class, and elapsed milliseconds in the versioned repository registry; registry merges preserve this metadata across discovery refreshes and process restarts.
- Added selected-profile auto-fetch policy overrides with validation, schema, documentation, and app-level selection coverage.
- Exposed measured fetch duration as repository-dashboard latency metadata and expanded Git transport classification for credential prompts/HTTP-style auth failures.
- Focused tests cover a 20-repository concurrency cap, active-operation skipping, failure backoff, cancellation, configuration defaults, and enabled-interval validation.
- Remaining: provider-specific rate-limit/auth signals beyond Git transport text, and native/manual acceptance evidence.
