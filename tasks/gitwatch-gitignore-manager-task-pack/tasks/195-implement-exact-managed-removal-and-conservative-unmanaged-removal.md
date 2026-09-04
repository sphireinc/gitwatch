# Task 195: Implement exact managed removal and conservative unmanaged removal

## Objective

Allow one or many combinations to be removed without disturbing unrelated `.gitignore` material.

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

Managed blocks are owned and can be deleted exactly. Pre-existing/unmanaged matches require a conservative algorithm and explicit preview because gitwatch cannot know the author’s intent. The default must favor leaving extra ignore rules over deleting user-authored rules.

## Implementation steps

1. Implement `PlanRemoveTemplates(snapshot, templateIDs)` and distinguish managed vs unmanaged paths per selected template.
2. For managed templates, delete the exact begin/body/end span plus only separator whitespace that gitwatch itself can prove belongs to that block. Do not collapse unrelated blank lines.
3. For unmanaged full matches, calculate a conservative rule-deletion plan using the ownership index. Never delete a rule occurrence that is ambiguous, shared with an unselected matched template, duplicated in an explicitly user-owned region, or cannot be mapped exactly.
4. Expose an `ambiguous remainder` list. The operation may complete while intentionally leaving shared/ambiguous rules; the UI must state this clearly instead of claiming a perfect removal.
5. Support multi-remove as one plan so ownership analysis sees all selected removals simultaneously.
6. If a managed block was edited/tampered, do not delete it with a normal single-key action. Require a preview that shows the entire exact span and labels it `modified managed block`.
7. After removal, if `.gitignore` would become empty solely because gitwatch removed all managed content, keep an empty file by default. Do not delete `.gitignore` unless the user explicitly chooses `Delete empty file`.

## Expected code areas

- `internal/gitignore/manage/remove.go`

## Required tests

- Remove first/middle/last managed block.
- Remove all managed blocks around handwritten rules.
- Conservative unmanaged removal with shared rules.
- Tampered managed block behavior.

## Acceptance criteria

- [ ] Managed block removal is exact.
- [ ] Unmanaged removal never deletes ambiguous/shared user rules by default.
- [ ] Multi-remove is ownership-aware.
- [ ] Edited managed blocks require elevated confirmation.
- [ ] Empty-file deletion is never implicit.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
