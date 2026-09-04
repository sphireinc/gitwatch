# Task 196: Add preview, diff, and atomic write transactions

## Objective

Make every `.gitignore` mutation inspectable before it touches disk, then execute it atomically with strong race and path safety.

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

This is a source-control tool; modifying `.gitignore` can hide files from `git status`. A bad edit is consequential. Treat add/remove/update like a transaction.

## Implementation steps

1. Generate a unified diff or structured before/after preview from every mutation plan. The preview must highlight managed blocks being added/removed and warnings about shared/ambiguous rules.
2. Before execute, re-stat and re-read the target, verify repository root containment, verify it is the same regular file or still absent, and compare SHA-256. Abort on any difference.
3. Refuse to automatically mutate symlinked `.gitignore` files by default. Surface the resolved target and explain that the user can edit it manually. Do not follow a symlink outside the repository.
4. Write through a temp file in the same directory, preserve mode bits where applicable, flush/fsync as supported, then atomic rename. Clean temp files on failure.
5. Record the operation through gitwatch’s existing operation journal/timeline with repo ID, mutation kind, selected template IDs, before/after hashes, and success/failure. Never log entire private file contents.
6. Provide an undo payload when safe: the exact before bytes and after hash for the immediately applied operation. Undo must only restore if current `.gitignore` still equals the operation’s after hash; otherwise refuse.

## Expected code areas

- `internal/gitignore/manage/plan.go`
- `internal/gitignore/manage/transaction.go`
- `internal/operations/*`

## Required tests

- Atomic write failure cleanup.
- Before-hash mismatch.
- Symlink refusal.
- Undo success and undo refusal after external edit.

## Acceptance criteria

- [ ] Every mutation has a preview representation.
- [ ] Writes are atomic and race-protected.
- [ ] Symlink targets are not silently followed.
- [ ] Operation history identifies what templates changed.
- [ ] Undo cannot clobber subsequent edits.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
