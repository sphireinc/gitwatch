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

- [ ] Tree mode is purely a view over the same live status snapshot.
- [ ] Flat/tree toggle never changes refresh semantics.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
