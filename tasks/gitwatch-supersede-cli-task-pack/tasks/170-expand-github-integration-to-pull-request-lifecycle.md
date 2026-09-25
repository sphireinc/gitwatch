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

- [x] Implementation commit recorded (`91424f1`).
- [x] Exact tested revision recorded (`91424f1`, Darwin arm64).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added the provider-neutral bounded `PullRequestListClient` contract and
  `ParsePullRequests` page parser.
- Added GitHub open-PR page loading with clamped page size (maximum 100),
  preserving the existing optional-provider error and cancellation behavior.
- Added parser, pagination-query, and page-bound tests; focused provider tests,
  provider race tests, and `git diff --check` pass.
- Added bounded provider-neutral PR detail models and GitHub loading for PR
  metadata, commits, and changed files/patches.
- Added an explicit validated create contract and non-retrying GitHub POST;
  it creates only the provider PR and never pushes a local branch.
- Extended the GitHub workspace model to render bounded open-PR lists and
  detail commit/file summaries; the asynchronous app load path requests these
  optional provider details without affecting local status loading.
- Added provider-ref validation and an explicit `[x]` GitHub checkout
  confirmation that resolves the matching configured GitHub remote and routes
  through the existing typed detached remote checkout boundary. Unsafe refs,
  missing remotes, and cancellation are covered by app/provider tests.
- Added a three-field GitHub PR create form (title, body, base) with required
  current-branch validation, explicit confirmation, provider-only creation,
  and reload on success; it never performs a hidden Git push.
- Added TTL caches for bounded open-PR pages and PR detail; expired provider
  data can be rendered as stale without blocking or replacing local Git status
  refresh.
- Remaining: broader provider failure presentation and native/manual acceptance
  evidence.
- At revision `6a23844`, the focused provider, GitHub workspace, and app suites
  passed with `go test ./internal/provider ./internal/ui/githubview
  ./internal/app`, covering pagination/detail parsing, provider error states,
  stale-cache presentation, PR creation, checkout validation, and optional
  app loading. Native/manual acceptance remains open.

## Additional progress evidence (2026-09-25)

- Commit `91424f1` removes early returns when the current-branch PR or checks
  endpoint fails. Current-branch absence is now a normal empty result; open PRs,
  issues, releases, checks, and PR-specific resources load independently and
  display bounded, sanitized per-resource warnings. Provider errors remain
  within the optional GitHub workspace and do not set the core app to error.
- PR create success invalidates the branch-scoped PR cache before reload, and
  late create results are rejected after repository-generation changes. Added
  cache, provider, view, and mocked-provider app tests; docs now explain empty
  current-branch PR state and failure isolation.
- On Darwin arm64 / Go 1.27.0, focused suites
  `go test ./internal/provider ./internal/ui/githubview ./internal/app` and the
  full `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  make check` passed. The full gate reported pinned lint (0 issues), full and
  race tests, vet, formatting, diff, security fuzz, and performance success.
- Hosted CI run `36085163506` for `91424f1` was queued at the time of this
  update. Native/manual terminal acceptance remains outstanding; this task
  remains active.

- Follow-up: hosted Actions run `36085163506` completed successfully for
  `91424f1`. Quality/policy and full-history secret scanning passed, as did
  Ubuntu, macOS, and Windows test/build/runtime jobs. This is hosted evidence;
  native/manual terminal acceptance remains outstanding, so the task remains
  active.
