# Task 191: Define the gitwatch managed-block format

## Objective

Create a stable marker format for combinations added by gitwatch so future removal and updates are exact.

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

Do not invent opaque state in a separate database as the sole source of ownership. Ownership metadata must travel with `.gitignore` itself so the file remains understandable when copied to another machine or opened outside gitwatch.

## Implementation steps

1. Adopt a human-readable format similar to:

`# >>> gitwatch:gitignore begin id=root/PHP source=github/gitignore commit=<sha256-or-commit>`
`<upstream content verbatim>`
`# <<< gitwatch:gitignore end id=root/PHP`

Use a versioned grammar (`format=1`) if needed. Final exact spelling must be centralized, not duplicated across packages.
2. Store stable template ID, source identity, upstream commit, and template content hash in begin metadata. End marker must repeat the template ID so truncated/crossed blocks are detectable.
3. Insert template body bytes exactly as bundled except line-ending adaptation to the target document. Do not remove template comments or deduplicate rules inside the block.
4. Ensure markers themselves are valid `.gitignore` comments and have no behavioral effect on Git.
5. Provide `EncodeManagedBlock`, `ParseManagedBlock`, and validation helpers. Unknown future marker versions must be preserved but treated as not safely editable.
6. Document why duplicate rules across two managed templates are intentionally tolerated: reversibility and exact ownership are more important than cosmetic deduplication.

## Expected code areas

- `internal/gitignore/managed/*`
- `docs/gitignore-manager.md`

## Required tests

- Marker round-trip.
- Unknown format version.
- Template ID mismatch between begin/end.
- Two adjacent managed blocks with overlapping rules.

## Acceptance criteria

- [ ] A managed block can be parsed back to the exact template ID and source hash.
- [ ] Unknown/malformed blocks are not mutated.
- [ ] Markers do not alter Git ignore semantics.
- [ ] Template bodies remain independently removable even when rules overlap.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added the centralized versioned `internal/gitignore/managed` block grammar with stable template ID, source, upstream commit, content hash, and repeated end-marker ID metadata.
- Added exact body preservation with newline adaptation only for the target document boundary.
- Added strict parsing and validation for unknown formats, malformed metadata, and begin/end ID mismatches; duplicate/overlapping blocks remain independently removable.
- Documented the format and reversibility rationale in `docs/gitignore-manager.md`.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
