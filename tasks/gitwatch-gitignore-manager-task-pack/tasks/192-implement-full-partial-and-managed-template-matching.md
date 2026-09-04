# Task 192: Implement full, partial, and managed template matching

## Objective

Determine which catalog combinations are already represented in the current `.gitignore` and produce the match state used for the asterisk/pinning behavior.

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

A user may open gitwatch on a repository whose `.gitignore` predates gitwatch. The manager must recognize meaningful matches without claiming ownership. A full unmanaged match means every significant rule from the upstream template is already present in the document; comments are not required for match status. Partial means some but not all significant rules exist.

## Implementation steps

1. Extract each template’s significant rule lines using a read-only semantic representation. Preserve exact escape sequences. Ignore blank lines and comments for coverage matching, but do not trim or rewrite rule content.
2. Build an index from document rule text to line occurrences so matching all templates does not become O(templates × document-lines) with repeated scans.
3. Classify a template as `ManagedExact` when a valid managed block with matching template ID and content hash is present. If the ID matches but content was hand-edited, surface a distinct warning/degraded managed state rather than lying that it is exact.
4. Classify `UnmanagedFull` only when every unique significant rule in that template is present outside or across the document. `Partial` must include coverage counts for UI detail (`17/24 rules present`).
5. If a managed block exists at an older upstream hash, still mark the template installed, but expose `UpdateAvailable` separately when the active catalog version differs.
6. Return match results for all templates in one pass suitable for sorting/searching. Cache only by `(repoID, gitignoreSHA256, catalogVersion)`; invalidation must be trivial.

## Expected code areas

- `internal/gitignore/match/*`

## Required tests

- Managed exact/current, managed older-version, managed edited.
- Unmanaged full match with rules reordered.
- Partial match coverage.
- Overlapping templates sharing rules.

## Acceptance criteria

- [ ] All full matches can be sorted to the top and displayed with `*`.
- [ ] Partial matches never receive the full-match asterisk.
- [ ] Hand-edited managed blocks are called out.
- [ ] Matching scales across the whole catalog without UI lag.
- [ ] No match result implies ownership of unmanaged lines.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
