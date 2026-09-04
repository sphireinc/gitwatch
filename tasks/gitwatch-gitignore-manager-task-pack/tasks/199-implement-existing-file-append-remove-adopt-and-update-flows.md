# Task 199: Implement existing-file append, remove, adopt, and update flows

## Objective

Turn the template browser into a full manager when `.gitignore` already exists.

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

Users need to append combinations, remove managed combinations, safely handle pre-existing matching combinations, and update old managed blocks when the bundled upstream template changes.

## Implementation steps

1. Action selection must be state-aware: absent templates offer `Add`; managed templates offer `Remove` and, if stale, `Update`; unmanaged full matches offer `Adopt/Manage` and `Conservative Remove`; partial matches offer `Add full managed template`, `Inspect overlap`, and no one-key removal.
2. Implement `Adopt/Manage` only when safe. If an unmanaged template exists as one exact contiguous textual segment, wrap that exact segment in markers without changing its template bytes. If rules are scattered/reordered, do not rewrite the file just to adopt it; explain that it is matched but unmanaged.
3. Implement managed-template update as a block-local replacement: replace only that block’s body/metadata with the current catalog version, previewing upstream diff. Preserve all other file content.
4. Allow mixed multi-select actions only when they can be represented unambiguously. Otherwise group selected items into an explicit plan summary (`2 add, 1 remove, 1 update`) and require preview.
5. Expose a dedicated `Installed/Matched` filter so users can manage what is already represented without searching the entire catalog.
6. After any successful apply, recompute the document snapshot/matches and keep the manager open if the user invoked it as a workspace; otherwise return to status according to existing modal conventions.

## Expected code areas

- `internal/tui/gitignore/manage.go`
- `internal/gitignore/manage/adopt.go`
- `internal/gitignore/manage/update.go`

## Required tests

- Adopt exact contiguous template.
- Refuse adopt for scattered match.
- Update one stale block among handwritten content.
- Mixed action plan summary.

## Acceptance criteria

- [ ] Existing files can append one/many managed templates.
- [ ] Managed templates can be removed exactly.
- [ ] Safe contiguous legacy content can be adopted without semantic edits.
- [ ] Stale managed blocks can be updated independently.
- [ ] Scattered unmanaged matches are never silently rewritten.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
