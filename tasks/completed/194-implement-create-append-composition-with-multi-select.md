# Task 194: Implement create/append composition with multi-select

## Objective

Build the mutation engine for creating a new `.gitignore` or appending one or many selected combinations to an existing file.

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

The user may select PHP + Composer + macOS + JetBrains in one operation. Each selected template remains an independent managed block. Existing content stays where it is. gitwatch should append a small separation boundary only as needed; it must not reorder user content.

## Implementation steps

1. Create `PlanAddTemplates(snapshot, templateIDs)` returning a complete `MutationPlan` without writing. Validate IDs, reject already-managed exact selections as no-ops, and identify full unmanaged matches as `already present; no append needed` by default.
2. For an absent/empty `.gitignore`, create the document with managed blocks in deterministic selection order (UI order or stable template ID order—choose one and document it).
3. For an existing file, append after the existing final content. Insert exactly enough newline separation to avoid corrupting the previous final rule/comment. Preserve whether unrelated existing content had a final newline, but the resulting managed append may legitimately add one.
4. Do not duplicate a template that is already a managed match. For an unmanaged full match, default action is **Adopt/Leave Existing**, not append duplicate content. UI can offer `Add managed copy anyway` only as an explicit advanced action.
5. Allow multiple selected templates in one atomic write. Either all selected additions apply or none do.
6. Mutation execution must compare the current file SHA-256 to the plan’s before hash immediately before writing. On mismatch, abort with `ConcurrentModification` and force a re-preview.

## Expected code areas

- `internal/gitignore/manage/add.go`
- `internal/gitignore/manage/write.go`

## Required tests

- New file with one/many templates.
- Append to LF/CRLF/no-final-newline files.
- Append after comments and negation rules.
- Concurrent edit between preview and apply.

## Acceptance criteria

- [ ] One operation can create/append multiple combinations.
- [ ] Existing content is preserved byte-for-byte before the insertion boundary.
- [ ] Already-installed combinations are not silently duplicated.
- [ ] Concurrent external edits are never overwritten.
- [ ] An append is atomic.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added `internal/gitignore/manage` with pure `PlanAddTemplates` preview generation and atomic `Apply` execution.
- Added deterministic multi-select ordering, managed duplicate rejection, unmanaged-full adopt/leave warnings, exact prefix preservation, and LF/CRLF boundary handling.
- Added immediate before-hash comparison, symlink rejection, temporary-file sync, atomic rename, and directory sync protection.
- Added tests for new/multi-template creation, existing comments and negation rules, CRLF/no-final-newline append behavior, duplicate prevention, and concurrent external edits.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
