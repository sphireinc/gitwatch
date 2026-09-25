# Task 173: Add GitHub issues releases and navigation hooks

**Phase:** GitHub
**Depends on:** 170, 172

## Goal

Round out high-value repository-provider integration without turning gitwatch into a full GitHub clone.

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

1. Add issue list/search/open-browser and create issue with title/body/labels where supported.
2. Add releases/tags read-only summary and open release/browser actions.
3. Do not duplicate the project’s signed-release pipeline inside TUI unless a future explicit task designs it.
4. Add provider URL actions from commit, branch, tag, PR, issue and workflow run.
5. Keep all provider lists bounded/paginated and provider-neutral where practical.
6. Core repository health remains useful offline/no-auth.

## Verification

- No auth, partial scopes, pagination, provider disabled.

## Acceptance criteria

- [ ] GitHub workspace covers common repository lifecycle without becoming a core dependency.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded (`91424f1`, Darwin arm64).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added bounded provider-neutral issue parsing, pull-request exclusion,
  issue title/body/label validation, paginated issue listing, and typed issue
  creation with non-retrying POST behavior.
- Added bounded release parsing and paginated read-only release listing.
- Added focused parser/client tests covering bounds, invalid data, query
  clamping, issue creation, and release summaries.
- Remaining: add GitHub workspace issue/release summaries and open/create
- Added stale-tolerant cached issue and release loading to the GitHub workspace
  and sanitized bounded issue/release summaries in the terminal view.
- Added a guarded three-field issue form (title, body, comma-separated labels)
  with typed validation, explicit confirmation, non-retrying creation, and
  provider reload after creation.
- Remaining: add open/list navigation hooks for issue and release URLs,
- Added explicit browser navigation for the first bounded open issue (`O`)
  and release (`L`) summaries, with safe missing-URL handling and tests.
- Palette issue/release entries now retain their selected index in the
  repository-scoped GitHub model, and `O`/`L` open the selected item rather
  than always opening the first result. Missing selected URLs remain a safe
  no-op with a user-visible status. App coverage verifies both selected-item
  routes. Remaining provider URL actions for commits, branches, tags, and
  workflow runs, plus native/manual acceptance for disabled/no-auth/
  partial-scope states.

- Added palette actions for the selected local commit, branch, and tag. These
  construct HTTPS provider URLs from the loaded repository identity with
  escaped refs and route through the platform URL opener; no provider request
  or Git mutation is performed. App coverage verifies all three actions.
- Added an explicit `W` action for the selected bounded check/workflow run URL,
  including safe empty-selection and missing-URL handling. Removed the
  unreachable duplicate GitHub `c` branch; `c` remains the review-comment
  action. Focused app coverage verifies the selected check URL route.
  Native/manual provider-state acceptance remains open.

- At revision `7d1e078`, the repository-wide `make check` passed with pinned
  lint reporting 0 issues, full tests, race, vet, security fuzz, and
  performance gates. Provider-disabled, no-auth, partial-scope, and native
  terminal acceptance remain explicit follow-up evidence.

- At tested revision `91424f1`, Darwin arm64 `make check` passed with pinned
  lint (0 issues), full and race tests, vet, formatting, diff checks, security
  fuzz, and performance checks. Hosted run `36085163506` passed quality/policy
  and full-history secret scanning; its Ubuntu/macOS/Windows test jobs were
  still running at this update. Provider-disabled, no-auth, partial-scope, and
  native/manual acceptance remain open.

- Follow-up: hosted Actions run `36085163506` completed successfully for
  `91424f1`, including Ubuntu, macOS, and Windows test/build/runtime jobs.
  Provider-disabled, no-auth, partial-scope, and native/manual acceptance
  remain open.
