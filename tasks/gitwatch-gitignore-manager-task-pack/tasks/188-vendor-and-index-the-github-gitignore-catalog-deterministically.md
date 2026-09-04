# Task 188: Vendor and index the GitHub gitignore catalog deterministically

## Objective

Add a maintainer-facing sync pipeline that imports `github/github/gitignore` at an explicit upstream commit into gitwatch and produces deterministic embedded assets.

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

GitHub’s `github/gitignore` repository is the canonical upstream for this feature. Its root holds common templates, `Global/` holds editor/tool/OS templates, and `community/` holds specialized templates. The upstream project is CC0-1.0. Preserve provenance even though the license is permissive.

Do not fetch `main` during normal builds. A build must be reproducible from the repository checkout.

## Implementation steps

1. Create `tools/gitignore-sync` as a Go command. It must require `--commit <40-hex-sha>` for deterministic sync. Optionally support `--repo github/gitignore`, but default to that canonical repository.
2. Download or obtain the archive for exactly that commit without invoking a shell. Validate HTTP status, archive paths, decompression limits, and that extracted paths cannot escape the destination. Reject symlinks and non-regular template files.
3. Import every `*.gitignore` from the repository root, `Global/`, and recursively under `community/`. Do not import CI files or unrelated docs as templates.
4. Copy upstream `LICENSE` and record upstream repository URL, commit SHA, sync timestamp, per-template SHA-256, source path, byte length, and category into a generated manifest.
5. Write assets under a deterministic path such as `internal/gitignore/assets/catalog/`. Generated output must sort entries by stable template ID and must be byte-for-byte reproducible for the same upstream commit.
6. Add `//go:generate` only if it can be deterministic and never silently points at moving `main`. Prefer an explicit `make gitignore-sync COMMIT=...` or documented `go run ./tools/gitignore-sync --commit ...`.
7. Add CI validation that regenerates catalog metadata from checked-in assets without network access and fails if manifest hashes do not match.

## Expected code areas

- `tools/gitignore-sync/*`
- `internal/gitignore/assets/catalog/*`
- `internal/gitignore/assets/manifest.json`
- `internal/gitignore/assets/LICENSE.github-gitignore`

## Required tests

- Golden test for generated manifest ordering.
- Archive traversal rejection (`../`, absolute paths, symlinks).
- Hash mismatch detection.

## Acceptance criteria

- [ ] Normal `go build` and tests require no network access.
- [ ] Every embedded template has provenance and a SHA-256.
- [ ] Root, Global, and community templates are represented.
- [ ] Archive extraction is path-safe.
- [ ] The same upstream commit produces identical generated output.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added the commit-pinned `tools/gitignore-sync` command and reusable archive parser.
- Added strict traversal, size, regular-file, and symlink checks for tar and ZIP archives; the default URL is the exact commit ZIP and never moving `main`.
- Generated 309 regular root/Global/community templates, a copied upstream license, and a stable sorted manifest for commit `52f5a2bf5785a851e69936a6f5c54a734b828046`.
- Added offline manifest verification covering provenance, category coverage, ordering, byte lengths, and SHA-256 hashes.
- Documented reproducible invocation and source provenance in `tools/gitignore-sync/README.md`.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
- Explicit source exception: the upstream snapshot stores `Clojure.gitignore`, `Fortran.gitignore`, and `Global/Octave.gitignore` as symlink aliases. They are excluded by the reviewed `--skip-symlinks` generation and are neither dereferenced nor presented as independent template content; strict default parsing still rejects them.
- Generated vendor bytes are canonicalized (LF endings, trailing horizontal whitespace removed, one terminal newline) before hashing so the existing repository whitespace gate remains unchanged; manifest hash/length verification covers the generated bytes.
