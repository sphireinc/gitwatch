# Task 204: Harden path, content, and terminal safety

## Objective

Treat upstream template content and repository paths as untrusted input throughout display and mutation flows.

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

`.gitignore` templates are plain text, but gitwatch renders them in a terminal and writes them into repositories. Repository filenames and existing `.gitignore` contents may contain control characters or maliciously crafted marker-like text.

## Implementation steps

1. Sanitize/escape control sequences in all TUI previews. Never allow template content or existing comments to inject terminal escape sequences.
2. Enforce repository-root containment on all `.gitignore` target calculations. Do not accept `..`, alternate roots, or user-supplied paths for this feature.
3. Refuse NUL/binary `.gitignore` files. Put a configurable but generous maximum file size for interactive parsing; for oversized files show read-only status with guidance rather than loading unbounded content.
4. Validate template IDs and marker metadata before using them as map keys, paths, or display strings. Template IDs are data, never filesystem paths after catalog ingestion.
5. Audit Windows path/case handling and repository roots on drive boundaries. Atomic-write implementation must account for platform rename semantics.
6. Do not execute content from GitHub templates. Upstream refresh is data-only. No hooks/scripts are imported.
7. Fuzz the document parser, marker parser, archive importer, and match engine.

## Expected code areas

- `internal/gitignore/security/*`
- `internal/tui/render/*`

## Required tests

- ANSI/control-character fixture.
- Path traversal template IDs/archive entries.
- NUL file.
- Oversized file behavior.
- Go fuzz targets.

## Acceptance criteria

- [ ] Terminal content cannot inject raw escape behavior.
- [ ] Writes cannot escape repository root.
- [ ] Binary/oversized files fail safely.
- [ ] Fuzzing finds no panic or unbounded allocation in target packages.
- [ ] Windows path cases are covered.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
