# Task 172: Add GitHub Actions and check-run operations

**Phase:** GitHub
**Depends on:** 170

## Goal

Turn CI visibility into actionable but bounded workflow support.

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

1. List workflow/check runs for current branch/PR with status, conclusion, duration, attempt and URL.
2. Support open logs URL, rerun failed jobs/run and cancel run where API permits.
3. Do not stream unlimited logs into the TUI initially; bounded summaries plus browser/open URL are sufficient.
4. Background-refresh at conservative TTL with rate-limit backoff.
5. Expose cached CI attention in multi-repo dashboard using bounded provider workers.
6. Provider work is lower priority than local status refresh.

## Verification

- Running/success/failure/canceled, rate-limit/backoff, 20-repo provider dashboard.

## Acceptance criteria

- [ ] CI actions never starve filesystem/status refresh.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Progress evidence

- Extended bounded check-run models with provider identity and run-attempt
  fields, preserving status, conclusion, duration, and URL visibility.
- Added typed, non-retrying provider operations for rerunning failed jobs and
  cancelling a workflow run. Empty 204 responses are accepted without
  weakening bounded response handling.
- Added parser bounds and focused tests for identity, attempts, oversized
  pages, invalid IDs, authorization, endpoint paths, and non-retrying action
  requests.
- Added GitHub workspace selection and explicit confirmation for rerunning
  failed completed runs and cancelling in-progress runs; successful requests
  reload provider state. Existing bounded URL opening remains available for
  logs.
- Added explicit provider-state presentation to the GitHub workspace. Provider
  errors now render classified states such as `rate-limited`, `unauthorized`,
  `not configured`, and `unavailable`; bounded `Retry-After` metadata is shown
  when supplied, while local Git status remains independent.
- Focused provider/GitHub-view tests, race tests, app tests, vet, and diff
  checks pass at the pushed revision. Remaining work is multi-repository
  cached CI attention and native/manual evidence that provider work does not
  starve local status refresh.

## Progress evidence (2026-09-22)

- Added repository-scoped cached CI attention projection. Once provider checks are loaded for a repository, the repositories dashboard shows `ci:passing`, `ci:pending`, or `ci:failing`, marks stale cached data explicitly, and includes a bounded attention reason for failed checks or provider errors.
- Unvisited repositories remain unchanged, provider data remains optional, and local Git health/status refresh paths are not replaced or blocked.
- Added focused app and repositories-view tests for repository isolation and stale/failing CI presentation. Full quality-gate evidence is pending for this revision.
- Remaining: broader multi-repository provider worker coverage, native/manual acceptance, and hosted cross-platform evidence.
