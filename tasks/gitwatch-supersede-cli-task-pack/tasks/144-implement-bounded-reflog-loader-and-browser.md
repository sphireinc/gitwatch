# Task 144: Implement bounded reflog loader and browser

**Phase:** Recovery and undo
**Depends on:** 125

## Goal

Expose reflog as a recovery surface and foundation for semantic undo.

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

1. Create `internal/reflog` loading ref selector, SHA, timestamp, actor and reflog subject/action using stable delimiters.
2. Default to HEAD reflog; allow selected local branch reflog.
3. Use bounded pages and cancellable loads.
4. Create Reflog workspace with commit inspection, compare-to-HEAD, create branch at entry and copy SHA.
5. Detached checkout uses existing explicit confirmation.
6. Do not add a casual hard-reset key.

## Git/process boundary

- `git reflog show --format=<machine-readable-delimiters> ...`

## Verification

- Rebase/reset/commit/checkout reflog fixtures, large reflog pagination.

## Acceptance criteria

- [x] User can inspect and branch from recovery points safely.
- [x] Parsing is not locale-dependent.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added `internal/reflog` with typed bounded page requests, default HEAD
  loading, caller-selected reflog refs, cancellable Git execution, stable
  NUL-delimited parsing, actor/action extraction, and output-size bounds.
- Real-repository tests cover commit reflog records, pagination limits, unsafe
  refs, negative offsets, and malformed timestamps. The workspace/browser and
  recovery-point actions now have an initial bounded browser route through the
  command palette, selection movement, and page loading; compare-to-HEAD and
  detached-checkout safeguards remain outstanding.
- Selected recovery points now reuse the existing commit inspector, exact
  checkout confirmation, and explicit branch-at-commit name flow. App tests
  cover all three actions without adding a hard-reset shortcut.
- The selected recovery point can now be compared to current `HEAD` through
  the existing typed history inspection boundary; the checkout prompt calls
  out that it enters detached `HEAD`. Focused app coverage verifies compare
  scheduling. All code-level recovery-point requirements are now implemented;
  native/manual validation remains outstanding.
- Verification at revision `cda3a1b`: `GOCACHE=/tmp/gitwatch-go-cache
  GOMODCACHE=/tmp/gitwatch-go-mod-cache make check` passed on Darwin arm64,
  including formatting, pinned golangci-lint (0 issues), `go test ./...`,
  `go test -race ./...`, vet, security checks, performance checks, and diff
  checks. Focused `go test ./internal/reflog ./internal/app -count=1` also
  passed. The reflog parser tests cover stable NUL-delimited records,
  timestamps, pagination bounds, malformed input, and unsafe refs; app tests
  cover bounded palette routing and inspect, detached-checkout confirmation,
  branch creation, and compare-to-HEAD flows. Implementation landed in
  `c34c9e7`, `55ca375`, `661095c`, `59e541a`, and `c8deeb4`. CI run
  [35866061892](https://github.com/sphireinc/gitwatch/actions/runs/35866061892)
  passed the full Windows/macOS/Ubuntu matrix at `0c362d1`; the code under test
  is unchanged at `cda3a1b`. Native/manual terminal evidence is still required.
- The bounded loader already capped Git output at `MaxBytes`, but the parser
  itself split its input before checking that same limit. `parse` now rejects
  oversized input before conversion or field allocation; a regression test
  and fuzz property cover the boundary. Focused tests and a two-second fuzz
  run with a fresh Go cache passed, followed by the full local `make check`
  (lint 0 issues, normal/race tests, vet, formatting, security, performance).
  Hosted CI for this hardening change is pending; native/manual acceptance
  remains open.
