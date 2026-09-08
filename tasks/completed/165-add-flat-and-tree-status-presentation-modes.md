# Task 165: Add flat and tree status presentation modes

**Phase:** History and inspection
**Depends on:** 124

## Goal

Add collapsible directory navigation without changing the underlying authoritative flat status snapshot.

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

1. Create `internal/ui/filetree` as a pure presentation index built from immutable status paths.
2. Support expand/collapse directory, collapse all, expand all and preserve selection by path identity across refresh.
3. Aggregate staged/unstaged/untracked/conflict counts on directory rows.
4. Apply filtering to authoritative file rows first, then rebuild the tree so directory counts match visible children.
5. Do not add a second status parser or tree-specific Git state.
6. Handle Windows display separators without corrupting Git path identity.

## Verification

- Deep paths, thousands of entries, renames, selection preservation, Windows path rendering.

## Acceptance criteria

- [x] Tree mode is purely a view over the same live status snapshot.
- [x] Flat/tree toggle never changes refresh semantics.

## Completion record

- [x] Implementation commit recorded: `c96a1a1` (`feat: add collapsible status tree presentation`).
- [x] Exact tested revision recorded: `c96a1a1`.
- [x] Focused unit/integration tests recorded: `GOCACHE=/tmp/git-watch-165-focused-cache ... go test ./internal/ui/filetree ./internal/app`; passed with tree arithmetic, app routing, filtering, selection, and source-snapshot coverage.
- [x] `go test ./...` recorded: passed through `make check`.
- [x] Race/vet/lint/format evidence recorded where applicable: `make check` passed formatting, golangci-lint with 0 issues, `go test -race ./...`, `go vet ./...`, `git diff --check`, security, and performance checks.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented: interactive 80x24, `NO_COLOR`, reduced-motion, mouse, and native Windows acceptance still require maintainer-run terminal sessions; automated tests cover the pure presentation and app routing contracts.

## Local progress evidence

- Added `internal/ui/filetree`, a pure presentation index that receives the
  existing filtered/sorted status entry indexes, preserves path selection, and
  aggregates staged, modified, untracked, and conflict counts for directories.
- Added `O` flat/tree switching, `Enter` directory toggling, `[`/`]` collapse/
  expand-all controls, keyboard page/home/end behavior, and mouse selection
  parity while keeping mutation and diff lookup keyed by the underlying file
  entry.
- Focused filetree and app tests cover nested paths, filtered indexes,
  aggregation, collapse/expand, selection preservation, and authoritative
  source-entry preservation. Full repository gates and manual 80x24/native
  acceptance remain pending; that manual evidence is an explicit release
  follow-up because this environment does not provide native Windows or a
  maintainer-operated terminal acceptance session.
