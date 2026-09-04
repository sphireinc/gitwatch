# Task 170: Expand GitHub integration to pull request lifecycle

**Phase:** GitHub
**Depends on:** 125, 161

## Goal

Move optional GitHub support from read-only visibility to a practical pull-request workspace without coupling core Git to GitHub.

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

1. Extend provider-neutral interfaces first; GitHub remains one provider implementation.
2. Load open PRs with number, title, head/base, author, draft, mergeability summary, review state and checks summary using bounded pagination.
3. Add PR detail with commits/files and selected file diff where provider API supports it.
4. Support create PR from current branch with explicit base/title/body. If push/upstream is missing, offer the existing Git push workflow instead of hidden push.
5. Support PR checkout only after validating provider-declared refs and explicit user action.
6. Cache provider data with TTL/background workers that never block local status refresh.
7. Provider unavailable/auth/rate-limit states must degrade without affecting Git.

## Verification

- No auth, expired auth, rate limit, large pagination, non-GitHub remote.

## Acceptance criteria

- [ ] PR create/inspect/checkout works while provider remains optional and failure-isolated.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
