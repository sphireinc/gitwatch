# Task 203: Wire `.gitignore` mutations into live watcher/status refresh correctly

## Objective

Ensure the primary live filesystem-driven status experience remains authoritative when ignore rules are changed.

## Non-negotiable constraints

- **Do not weaken gitwatch's live model.** Filesystem events are hints; the authoritative repository state remains the existing `git status --porcelain=v2 -z --branch` refresh pipeline. Editing `.gitignore` must feed back into that pipeline immediately and must not create a second status model.
- **Multi-repository support is mandatory.** Every state object and mutation must be scoped by repository identity/path. Never assume one global active repository.
- **Preserve user-authored `.gitignore` content byte-for-byte wherever it is not intentionally changed.** Do not sort, normalize, wrap, reformat, or rewrite unrelated content.
- **Never shell-interpolate paths or template names.** Reuse gitwatch's argv-based process runner and existing path/safety boundaries.
- **All writes require previewable intent and race protection.** Re-read/hash the target before write; abort if it changed after preview.
- **Managed template removal must be exact and reversible.** Never delete hand-written rules merely because they resemble an upstream template.
- **The feature must work offline.** The embedded catalog is always usable; runtime upstream refresh is additive and optional.
- **Do not make UI rendering perform filesystem, network, or Git work.** Background commands return Bubble Tea messages into the model.

## Context and required behavior

Changing `.gitignore` can immediately change which untracked files Git reports. The manager must not fake those effects. The file write should produce filesystem events, and the operation should also schedule a canonical refresh as a correctness fallback because atomic rename/event coalescing can vary by OS.

## Implementation steps

1. Ensure `.gitignore` create/write/rename/delete events are watched or cause the repository watcher to invalidate status. Audit Linux/macOS/Windows fsnotify behavior around atomic rename.
2. After a successful manager transaction, enqueue the same repository refresh message/path used by manual refresh and other mutations. This is a refresh trigger, not a separate status calculation.
3. If watcher topology itself excludes ignored directories, ensure a changed `.gitignore` causes any necessary watcher re-evaluation without recursively watching newly ignored directories forever. Status truth still comes from Git.
4. Debounce the transaction-triggered refresh with concurrent filesystem events so one apply does not create a refresh storm.
5. While the gitignore manager overlay is open, allow repository status updates to continue in the background. Do not freeze the underlying live model.
6. On external `.gitignore` edits detected while the manager is open, invalidate preview/match state and display `file changed externally; refresh preview` instead of applying stale plans.

## Expected code areas

- `internal/watch/*`
- `internal/status/*`
- `internal/tui/gitignore/*`

## Required tests

- Integration: untracked file disappears from status after adding matching ignore rule.
- Integration: removing rule makes file reappear.
- Watcher + explicit refresh coalescing.
- External edit while overlay open.

## Acceptance criteria

- [ ] Ignored/unignored file changes appear through canonical Git status.
- [ ] Atomic `.gitignore` writes are detected on all supported OSes.
- [ ] No synthetic status filtering is added.
- [ ] External edits invalidate stale plans.
- [ ] Refresh storms are avoided.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
