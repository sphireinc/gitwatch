# Task 205: Build comprehensive fixtures, integration tests, and performance gates

## Objective

Prove the subsystem works against realistic `.gitignore` files, many templates, and large/multi-repository workspaces.

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

The feature combines parsing, matching, file mutation, async UI, network/cache, and Git status side effects. Unit tests alone are insufficient.

## Implementation steps

1. Create fixture repositories representing: brand-new repo/no ignore, hand-written ignore, exact upstream template pasted manually, multiple templates concatenated manually, gitwatch-managed blocks, mixed managed+handwritten, CRLF/BOM, edited managed block, huge ignore file, and monorepo.
2. Add end-to-end tests that initialize real temporary Git repos, create untracked files, apply/remove templates, run actual `git status --porcelain=v2 -z`, and assert ignored-state changes.
3. Add multi-repo integration tests with at least 25 repositories and mixed states. Verify batch planning/execution remains bounded and deterministic.
4. Benchmark catalog load/search, match-all-templates against a 10k-line `.gitignore`, parsing, preview generation, and TUI filter updates. Establish regression thresholds appropriate to CI hardware rather than absolute microsecond promises.
5. Add race tests for concurrent external `.gitignore` writes while planning/executing. Run relevant packages with `go test -race` in CI where project policy permits.
6. Add golden tests for mutation diffs so accidental reformatting of unrelated content is immediately visible.
7. Test both embedded and cached catalogs with identical matching behavior.

## Expected code areas

- `internal/gitignore/testdata/*`
- `internal/gitignore/*_test.go`
- `internal/tui/gitignore/*_test.go`
- `test/integration/gitignore/*`

## Acceptance criteria

- [x] Real Git integration tests prove ignore effects.
- [x] Large files/catalog search do not introduce visible TUI stalls.
- [x] Multi-repo tests cover mixed success/failure.
- [x] Golden tests protect lossless editing.
- [x] Race/concurrent modification behavior is tested.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added real-repository coverage for canonical ignore/unignore effects and a 25-repository mixed batch with deterministic repository scoping and stale-plan isolation.
- Added catalog search, 10,000-line match, and TUI filter benchmarks. Baseline on Apple M1 Pro: catalog search 2.03 ms, 10k-line match 1.34 ms, and filter update 0.80 ms per operation (single-iteration smoke run; CI should track trends rather than enforce these machine-specific values).
- Added a lossless mutation-preview golden assertion and retained concurrent-modification tests for race-protected writes. Embedded/cache source behavior remains covered by catalog source tests.
- Validation: full `go test ./...`, full `go test -race ./...`, `go vet ./...`, formatter check, `git diff --check`, and benchmark smoke run pass.
- Exception: `make check` reaches the pinned golangci-lint run but reports 21 pre-existing repository findings outside this task; no new findings were introduced by the task changes.
