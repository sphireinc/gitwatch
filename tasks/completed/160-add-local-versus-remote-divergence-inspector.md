# Task 160: Add local-versus-remote divergence inspector

**Phase:** Tags and remotes
**Depends on:** 158, 161

## Goal

Make ahead/behind counts actionable by showing exactly what differs before fetch/pull/rebase/push.

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

1. Open comparison from branch, remotes or repository dashboard for local branch vs upstream.
2. Show unique commits on each side, file/stat summary and bounded full diff.
3. Offer explicit next actions fetch, ff-only pull, merge, rebase, push; each routes through existing engines.
4. Never auto-select a history-changing strategy.
5. Cache generation-scoped and invalidate after ref/fetch changes.

## Verification

- Ahead-only, behind-only, diverged, missing upstream.

## Acceptance criteria

- [x] Ahead/behind is inspectable, not merely a number.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [x] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Reused the completed arbitrary-revision comparison workspace to make the
  Remotes view actionable: `Y` assigns the current local branch first and the
  selected remote's matching tracking ref second, then opens the read-only
  divergence inspector without fetch, checkout, or mutation.
- Added focused routing coverage for local-versus-remote assignment. Unique
  commit lists are now loaded with bounded `git log` ranges and rendered on
  each comparison side alongside the existing file/stat summary. Explicit
  fetch, ff-only pull, merge pull, rebase pull, and push-preview actions now
  route through the existing Remotes engines after selecting the qualified
  comparison remote. The full repository gate passes at the recorded revision.

## Completion evidence

- Implementation commits: `4e78b54`, `f9f7ccc`, `999a92c`.
- Exact tested revision: `999a92c`.
- Focused coverage: local/remote assignment, bounded unique commit summaries,
  changed-file stats and patch output, qualified remote selection, and fetch,
  ff-only pull, merge pull, rebase pull, and push-preview routing.
- Full gate at the tested tree: `make check` passed, including lint,
  `go test ./...`, `go test -race ./...`, `go vet ./...`, formatting, diff,
  security, and performance checks.
- Native/manual exception: automated Bubble Tea tests cover the inspector
  rendering, keyboard assignment, action routing, and refresh paths; a human
  80x24 terminal pass remains release QA follow-up.
- Known limitation: comparison actions are routed through the existing remote
  engines, while the inspector itself remains read-only; fetch/pull/push
  confirmation and conflict recovery continue in their established workspaces.
