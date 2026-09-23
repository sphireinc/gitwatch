# Task 171: Add GitHub review comments and merge workflows

**Phase:** GitHub
**Depends on:** 170, 143

## Goal

Support common review and merge actions while keeping provider state distinct from local Git state.

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

1. Load review threads/comments in bounded sanitized views.
2. Support approve, request changes, comment and reply where scopes/permissions allow.
3. Before merge refresh mergeability/check/review state rather than trusting stale cache.
4. Support merge/squash/rebase provider merge methods with explicit selection and confirmation.
5. Branch deletion after merge is a separate explicit option.
6. After provider merge, refresh provider state; do not pretend local remote refs changed until an actual fetch updates them.
7. Provider errors are typed separately from Git/process errors.

## Verification

- Insufficient permission, stale mergeability, failed check, comment sanitization.

## Acceptance criteria

- [ ] PR review/merge is usable and never fabricates local Git state.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Progress evidence

- Added bounded provider-neutral review comment models, parsing, and request
  validation, including reply/inline target fields and sanitized-size limits.
- Added GitHub list/create review-comment operations with typed provider error
  handling and non-retrying POST semantics.
- Added explicit merge methods (`merge`, `squash`, `rebase`) and expected-SHA
  validation through a typed provider merge contract; local Git state is not
  changed by provider merge responses.
- Wired bounded review comments through the GitHub cache/load path and
  terminal-safe workspace rendering.
- Added merge-method selection, a pre-merge provider refresh, explicit
  confirmation, and post-merge provider reload messaging that explicitly says
  local refs remain unchanged until fetch.
- Added typed approve/request-changes/comment review submissions with bounded
  body validation, explicit UI confirmation, and post-submit provider reload.
- Added a separate, opt-in post-merge remote branch deletion confirmation;
  branch refs are validated before the typed GitHub DELETE request, local refs
  remain unchanged, and provider state is reloaded after the action or a
  cancellation.
- Added state-specific GitHub provider failure hints for missing credentials,
  unauthorized permissions, rate limits, and availability failures while
  preserving sanitized error text and the local-Git-authoritative boundary.
- Added explicit review-comment target selection (`[`/`]`) and reply composition
  through the typed `in_reply_to` provider request, with post-submit comment
  reload and target clearing.
- Focused provider/app/UI tests, provider race tests, vet, and
  `git diff --check` pass.
- Evidence for this slice: commit `2009643`; `go test ./internal/provider
  ./internal/app`, `go test -race ./internal/provider ./internal/app`,
  `go vet ./internal/provider ./internal/app`, and `git diff --check` pass.
- Evidence for provider error presentation: `go test ./internal/ui/githubview`,
  `go test -race ./internal/ui/githubview`, `go vet ./internal/ui/githubview`,
  and `git diff --check` pass.
- Evidence for reply targeting: `go test ./internal/app ./internal/provider
  ./internal/ui/githubview`, `go test -race ./internal/app
  ./internal/ui/githubview`, `go vet ./internal/app ./internal/provider
  ./internal/ui/githubview`, and `git diff --check` pass.
- Remaining: native/manual acceptance evidence.
- At revision `7d1e078`, the repository-wide `make check` passed with pinned
  golangci-lint reporting 0 issues, full tests, full race, vet, diff checks,
  security fuzz, and performance benchmarks. Native/manual terminal evidence
  remains the only task-specific gap recorded here.
- At revision `f0b249e`, the focused provider, GitHub workspace, and app suites
  passed in both normal and race modes, covering review parsing/submission,
  comment sanitization and reply targeting, explicit merge confirmation,
  branch-deletion separation, stale refresh, and typed provider failure states.
  Native/manual terminal evidence remains open.
