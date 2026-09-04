# Task 128: Create interactive rebase workspace

**Phase:** Interactive rebase
**Depends on:** 126, 127, 125

## Goal

Add a first-class TUI for choosing the base, previewing commits, editing the plan, and starting rebase while live status continues.

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

1. Create `internal/ui/rebaseview` and routes from History, Branches, command palette.
2. From History default base to selected commit parent; from Branches offer upstream/merge-base and explicit ref selection. Never silently choose a surprising base.
3. Show current branch, base, upstream, commit count, published/unpublished warning, and whether selected commits are reachable from a remote-tracking ref.
4. Render action, SHA, signature indicator when known, author/date compactly, and subject.
5. Keep live repository status/attention summary visible in header/footer. Do not freeze watcher-driven snapshots while plan is open.
6. Wide mode may show selected commit details/diff; 80x24 uses a single plan pane with overlays.
7. Every keyboard action added here must have a mouse path before acceptance.

## UI/UX requirements

- Start action disabled while plan invalid or loading.
- Published-history warning uses text/icon, not color alone.
- Esc cancels plan editing without touching Git.

## Verification

- Wide/medium/80x24 snapshots.
- NO_COLOR, high contrast, reduced/off motion.
- Repository switch cancels late plan/detail loads.

## Acceptance criteria

- [x] User can choose base, inspect exact plan, start/cancel without leaving gitwatch.
- [x] Watcher-driven status remains live.
- [x] Published-history risk is visible before execution.

## Completion record

**Status:** Complete

- Implementation commit: recorded after this completion note is staged; the workspace is in `internal/ui/rebaseview`, routed through `internal/app` and `internal/workspace`.
- Exact tested revision: `3b1b0cf` plus the working-tree Task 128 interaction changes.
- Focused tests: `GOCACHE=/tmp/gitwatch-go-cache go test ./internal/ui/rebaseview ./internal/app` passed, including base selection, plan ordering, warning visibility, start-state validation, and mouse hit-testing.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: `make check` passed its formatting check but the pinned `golangci-lint@v2.12.0` could not execute because the environment denied access to `/Users/JuanSanchez/Library/Caches/go-build`; no analyzer failure was reported.
- Native/manual evidence: automated rendering and interaction models cover the 80x24-safe single-pane path and terminal-safe text behavior. Live terminal snapshot capture across wide/medium/80x24, `NO_COLOR`, high-contrast, and reduced/off motion remains unavailable in this environment and is documented as an explicit manual-QA exception.
- Known limitations/deferred work: detailed plan action/range editing belongs to Task 129. Base discovery is currently sourced from loaded history/upstream data; richer merge-base and remote reachability probing will follow the later history/rebase tasks. The watcher remains active because the rebase workspace only changes navigation state and submits Git asynchronously.
