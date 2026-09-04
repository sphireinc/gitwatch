# Task 189: Build the offline template catalog and search index

## Objective

Expose the embedded templates through a fast, repository-independent catalog API with rich search metadata and stable ordering.

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

The UI must be able to type “php” and immediately find PHP without scrolling. Search should match names, paths, category, aliases, and useful technology terms. Root templates should generally outrank specialized/community variants when scores are equal, but full matches in the current repository are always pinned above non-matches.

## Implementation steps

1. Use `go:embed` to embed the generated catalog and manifest into the binary. Parse and validate it once at startup or lazily via `sync.Once`; never reparsed on every keystroke.
2. Create catalog APIs: `List()`, `Get(TemplateID)`, `Search(query)`, `ByCategory()`, and `Version()`. Returned template slices must be immutable copies or read-only structures.
3. Build normalized search tokens from display name, upstream filename, relative path, category, and curated aliases. Add aliases for common terms where the filename differs from what users type (for example macOS/OSX, Node/NodeJS, Dotnet/.NET). Keep aliases in gitwatch-owned metadata, not by editing upstream files.
4. Implement deterministic fuzzy ranking. Exact/prefix name matches outrank substring/path/category matches. Search must be fast enough to run on every TUI filter update without spawning goroutines per keystroke.
5. Expose category labels: `Common`, `Global / OS / Editor`, and `Community / Specialized`. Preserve nested community path information for disambiguation.
6. Add a catalog version string containing the upstream commit so the UI/help screen can disclose exactly which snapshot is bundled.

## Expected code areas

- `internal/gitignore/catalog/*`
- `internal/gitignore/assets/embed.go`

## Required tests

- Search ranking golden cases.
- Alias matching cases.
- Catalog load/hash validation.
- Benchmark search across the full bundled catalog.

## Acceptance criteria

- [ ] Searching `php` returns PHP near the top without scrolling.
- [ ] Catalog APIs work with no network connection.
- [ ] Names with punctuation such as C++, .NET, and Objective-C remain searchable.
- [ ] Search ordering is deterministic.
- [ ] The upstream commit is visible through the catalog API.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added `internal/gitignore/assets/embed.go` and `internal/gitignore/catalog` with one-time embedded loading and hash/length validation.
- Added immutable-copy `List`, `Get`, `Search`, `ByCategory`, and `Version` APIs.
- Added deterministic ranking with exact/prefix/substring/path/category precedence and curated aliases for common punctuation/name variants.
- Added category grouping and upstream commit disclosure through `Version()`.
- Added golden ranking, alias, hash-validation, immutability, and full-catalog benchmark coverage.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `git diff --check`, and the full-catalog benchmark.
