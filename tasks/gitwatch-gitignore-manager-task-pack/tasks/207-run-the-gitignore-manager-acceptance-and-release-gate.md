# Task 207: Run the `.gitignore` manager acceptance and release gate

## Objective

Complete a feature-level acceptance pass and refuse release until the workflow is safe, fast, offline-capable, and multi-repo aware.

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

This task is a hard gate, not a documentation checkbox. The feature is done only when a user can start from a fresh repo, create a composed `.gitignore`, continue live work, later append/remove/update combinations, and perform the same operations safely from multi-repo mode.

## Implementation steps

1. Execute the full automated test suite, race-sensitive tests, lint/vet, and platform CI on Linux, macOS, and Windows.
2. Manual acceptance: fresh repo → open manager → search `php` → select PHP plus at least one other template → preview → create → verify Git status immediately reflects ignored files without manual restart.
3. Manual acceptance: existing handwritten `.gitignore` → detect a full unmanaged match at top with `*` → append a new managed template → remove only the managed template → prove handwritten bytes remain unchanged.
4. Manual acceptance: two managed templates with overlapping rules → remove one → verify the other block and behavior remain intact.
5. Manual acceptance: stale/older managed block → load newer catalog → preview update → apply only block-local diff.
6. Manual acceptance: multi-repo dashboard → select several repos → batch add one template → preview per repo → execute → inspect partial failure report and per-repo live refresh.
7. Manual acceptance: external process edits `.gitignore` after preview but before apply → gitwatch refuses to overwrite and requests re-preview.
8. Manual acceptance offline: disable network/cache → browse/search full bundled catalog and create/remove templates.
9. Update release notes and capability matrix only after all gates pass.

## Expected code areas

- `docs/release/*`
- `test/integration/gitignore/*`

## Acceptance criteria

- [ ] All platform CI passes.
- [ ] Fresh-repo creation works with multi-select search.
- [ ] Existing handwritten content survives append/remove flows exactly.
- [ ] Overlap removal is reversible/safe.
- [ ] Multi-repo batch behavior is proven.
- [ ] Concurrent edits are protected.
- [ ] Offline use is proven.
- [ ] Live filesystem-driven Git status remains the primary post-mutation truth.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
