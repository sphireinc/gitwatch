# Task 190: Implement a lossless `.gitignore` document parser

## Objective

Build a document model that can locate managed sections and individual rules while preserving all bytes outside intended edits.

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

A `.gitignore` is user-owned text with semantics where escapes and whitespace can matter. This parser exists primarily for safe editing and detection, not to “beautify” the file. Preserve BOM, newline style, final-newline presence, blank lines, comments, escaped `#`/`!`, and raw rule text.

## Implementation steps

1. Parse bytes into a sequence of lossless line records containing raw bytes/text, logical content, line ending, classification (`blank`, `comment`, `rule`, `managed-marker`), and source offsets.
2. Detect UTF-8 BOM and preserve it. Reject binary/NUL-containing files from automatic management with a clear error.
3. Detect dominant newline style (`LF` or `CRLF`) and preserve mixed newlines exactly for untouched lines. New inserted content should use the dominant style; if the file is empty/new, use LF.
4. Recognize gitwatch begin/end markers strictly. Do not treat lookalike comments as managed blocks unless the full marker grammar and template ID/version metadata validate.
5. Return malformed or nested managed-block markers as `InvalidManagedBlock`; do not attempt automatic repair in this task.
6. Provide offset/range APIs so mutation code can replace or delete exact spans without reconstructing the entire document from normalized strings.

## Expected code areas

- `internal/gitignore/document/*`
- `internal/gitignore/testdata/*`

## Required tests

- Property/fuzz test: parse then render yields original bytes.
- Fixtures for BOM, CRLF, mixed newlines, escaped `#`, escaped spaces, negation rules, and no final newline.
- Malformed/nested marker fixtures.

## Acceptance criteria

- [ ] Parsing and rendering an untouched document is byte-identical.
- [ ] CRLF, BOM, no-final-newline, escaped comments, and mixed blank lines survive round trip.
- [ ] Malformed managed markers are detected instead of silently accepted.
- [ ] No Git ignore rule is semantically normalized for writing.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
