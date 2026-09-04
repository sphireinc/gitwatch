# Task 141: Implement safe hunk-level conflict resolution

**Phase:** Merge and conflicts
**Depends on:** 139, 140

## Goal

Go beyond whole-file ours/theirs by applying guarded hunk-level choices with stale-write protection.

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

1. Parse working-file conflict regions only as an edit/presentation target; keep index stages as identity/source.
2. Associate conflict regions with base/ours/theirs text where possible.
3. Implement choose ours/theirs/both/manual-edit per region with atomic file replacement preserving mode.
4. Before write, verify file generation/hash still matches the content used to build the hunk model. If changed externally, refuse and refresh.
5. Keep a bounded in-session undo stack for conflict-file edits. This is not Git-history undo.
6. Do not auto-stage after editing unless user explicitly chooses Resolve and stage.
7. Let watcher events naturally fire and also request an explicit authoritative refresh.

## Verification

- Multiple conflict hunks, concurrent external edit stale-write refusal, CRLF/LF, local edit undo.

## Acceptance criteria

- [ ] No stale conflict view can overwrite a newer external file.
- [ ] Staging remains explicit.
- [ ] Watcher observes resolution edits normally.

## Completion record

- [x] Implementation commits recorded: stale-safe region editor, repository-boundary operation, selected-region workspace integration, and manual editor routing.
- [x] Exact tested revision recorded: final implementation revision is recorded by the completion commit below.
- [x] Focused unit/integration tests recorded: `go test ./internal/conflicts ./internal/git ./internal/ui/conflictview ./internal/app`, including CRLF, stale-edit, atomic-mode, and explicit non-staging coverage.
- [x] `go test ./...` recorded: passed.
- [x] Race/vet/lint/format evidence recorded where applicable: `go test -race ./...` passed for changed packages; one unrelated watcher timing test failed in a repository-wide run and passed with `go test -race ./internal/watch -run TestWatcherDebouncesAndSeesCreatedDirectories -count=3`; `go vet ./...`, `gofmt`, and `git diff --check` passed. Pinned `make check` lint remains blocked by denied Go build-cache access.
- [x] Native/manual evidence recorded where this task changes terminal interaction: automated keyboard/mouse and 80x24/wide coverage passed; human terminal QA remains a documented release-gate limitation.
- [x] Known limitations/deferred work documented: hunk edits are intentionally unstaged; watcher-driven refresh and shared cross-operation lifecycle policy remain coordinated by the existing refresh path and later Task 143.
