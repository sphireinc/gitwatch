# Task 202: Add optional upstream catalog refresh with safe cache fallback

## Objective

Let users refresh GitHub’s template catalog without making network access a runtime requirement or weakening reproducibility.

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

The embedded snapshot is the guaranteed baseline. A runtime refresh may provide newer templates, but a bad network response, malformed archive, or cache corruption must never make the manager unusable.

## Implementation steps

1. Implement a catalog source layer with `EmbeddedCatalog` and optional `CachedCatalog`. At startup choose the newest valid catalog according to explicit metadata; fall back to embedded on any cache error.
2. Expose `Check for template updates` / `Refresh catalog` from the manager. Perform all network work asynchronously with timeout/cancel and bounded download size.
3. Resolve GitHub `main` to a concrete commit, then fetch exactly that commit’s archive. Record commit SHA and per-file hashes. If GitHub API resolution is unavailable/rate-limited, surface a friendly failure and retain the existing catalog.
4. Store cache in the platform-appropriate user cache directory, never inside the repository being watched. Use atomic cache replacement and a versioned manifest.
5. Validate the same path/archive/security rules as the maintainer sync tool. Share code instead of duplicating validators.
6. When a newer catalog activates, recompute template matches for open repositories lazily/bounded. Do not trigger Git working-tree mutations.
7. Offer `Use bundled catalog` and display active source/commit in the UI. Do not silently phone home on every startup; automatic checks must be opt-in/configurable with a sane interval.

## Expected code areas

- `internal/gitignore/catalog/source.go`
- `internal/gitignore/catalog/refresh.go`
- `internal/config/*`

## Required tests

- Offline startup.
- Corrupt cache fallback.
- Oversized/malicious archive rejection.
- Catalog refresh cancellation.

## Acceptance criteria

- [ ] Feature remains fully functional offline.
- [ ] Cache corruption automatically falls back to embedded catalog.
- [ ] Runtime refresh records an immutable upstream commit.
- [ ] No repository files are changed by catalog refresh.
- [ ] Automatic network checking is opt-in/configurable.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
