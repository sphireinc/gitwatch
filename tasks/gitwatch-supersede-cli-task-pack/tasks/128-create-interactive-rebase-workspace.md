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

- [ ] User can choose base, inspect exact plan, start/cancel without leaving gitwatch.
- [ ] Watcher-driven status remains live.
- [ ] Published-history risk is visible before execution.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
