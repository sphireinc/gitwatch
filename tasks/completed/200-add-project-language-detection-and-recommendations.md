# Task 200: Add project/language detection and recommendations

## Objective

Recommend likely templates based on repository contents while keeping selection completely user-controlled.

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

Recommendations should reduce search work in new repositories and monorepos. They must never auto-modify `.gitignore` and must explain why a suggestion was made.

## Implementation steps

1. Create a lightweight detector that inspects bounded repository metadata: marker files (`go.mod`, `package.json`, `composer.json`, `pyproject.toml`, `Cargo.toml`, `.csproj`, `pom.xml`, etc.), selected directory names, and a capped extension sample. Reuse existing repository scan infrastructure where possible; do not recursively scan millions of files on the UI thread.
2. Return recommendation records with template ID, confidence, and reasons such as `composer.json detected` or `68% of sampled source files are .php`.
3. Support multiple recommendations for polyglot/monorepo repositories. Do not force a single language classification.
4. Create a curated mapping layer from signals to catalog IDs and validate every mapped ID against the embedded manifest during tests/build.
5. Pin recommended absent templates after installed/matched templates but before the general catalog. Label them `Recommended`, not `Detected as required`.
6. Do not recommend Global editor/OS templates merely because gitwatch runs on that OS; global preference is a user choice.

## Expected code areas

- `internal/gitignore/recommend/*`

## Required tests

- Fixture repos for Go, Node, PHP/Composer, Python, Rust, Java, .NET and mixed monorepos.
- Large fixture proving bounded sampling.
- Mapping IDs all exist in catalog.

## Acceptance criteria

- [ ] PHP/composer projects recommend relevant PHP/Composer templates.
- [ ] Polyglot repos can receive multiple recommendations.
- [ ] Recommendations never auto-select or auto-apply.
- [ ] Detection is bounded and non-blocking.
- [ ] Every recommendation includes a human-readable reason.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added `internal/gitignore/recommend`, an offline bounded detector covering marker files, source extensions, dependency/build-directory skips, depth/file limits, polyglot results, confidence values, and human-readable reasons.
- Added curated mappings for Go, Node, PHP/Composer, Python, Rust, Java/Gradle, and .NET, with tests validating every mapped catalog ID against the embedded manifest.
- Connected detection to the gitignore browser as recommendation metadata only. Recommendations are labeled, explain their signals, pin before general entries, suppress for full installed matches, and never auto-select or auto-apply.
- Added fixture-style tests for Go, Node, PHP/Composer, Python, Rust, Java, .NET, mixed repositories, bounded sampling, mapping validity, and user-controlled selection.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
- `make check` reached lint but could not run because the sandbox denied the Go build cache path under `/Users/JuanSanchez/Library/Caches/go-build`; this remains an environment exception.
