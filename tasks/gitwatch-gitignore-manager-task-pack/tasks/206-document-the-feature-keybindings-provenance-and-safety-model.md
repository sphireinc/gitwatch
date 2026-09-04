# Task 206: Document the feature, keybindings, provenance, and safety model

## Objective

Ship user and maintainer documentation that explains what gitwatch owns, how template matching works, where templates come from, and how to recover.

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

Because removal behavior differs between managed and pre-existing files, documentation must be explicit. Users should understand that `*` means “full template match,” not necessarily “gitwatch owns these lines.”

## Implementation steps

1. Add README feature documentation with a concise screenshot/GIF slot, searchable multi-select workflow, creation, append, remove, update, and multi-repo behavior.
2. Document indicators exactly: `* managed/full match`, `~ partial`, etc., matching final UI semantics.
3. Document provenance: templates come from `github/github/gitignore`, bundled catalog commit is displayed in-app, upstream license is CC0-1.0, and runtime updates are optional.
4. Document managed marker format and the guarantee that gitwatch only performs exact automatic removal of content it owns. Explain conservative unmanaged removal and why shared rules may remain.
5. Update keymap/help docs and command palette descriptions. Include mouse and keyboard equivalents.
6. Add maintainer docs for syncing to a new upstream commit, reviewing diff size, regenerating manifest, and validating catalog hashes.
7. Document troubleshooting: malformed managed block, symlinked `.gitignore`, concurrent modification, offline catalog, cache reset, and restoring from operation undo.

## Expected code areas

- `README.md`
- `KEYMAP.md`
- `docs/gitignore-manager.md`
- `docs/maintainers/gitignore-catalog.md`

## Acceptance criteria

- [ ] Users can understand the asterisk and ownership distinction.
- [ ] Upstream source and license are disclosed.
- [ ] Maintainers can deterministically update bundled templates.
- [ ] Every UI action has discoverable help.
- [ ] Safety limitations are documented rather than hidden.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
