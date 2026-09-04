# Task 187: Define the `.gitignore` management domain and invariants

## Objective

Create the domain model and architectural contract for a first-class gitwatch `.gitignore` manager. This task establishes terminology, state, error types, repository scoping, and mutation boundaries before any UI is written.

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

The feature is not “download a template.” It is a reversible composition system over a user-owned file. A template combination is one upstream `.gitignore` file from `github/gitignore`. Multiple combinations may be active at once. A combination may be **managed** (added by gitwatch with markers), **matched/unmanaged** (gitwatch can confidently detect its rules in a pre-existing file), **partial**, or **absent**.

Define explicit states so later UI never guesses based on strings. The user-facing asterisk means a full match: managed or confidently matched. Partial matches must use a different indicator and must never be presented as fully installed.

## Implementation steps

1. Create `internal/gitignore/domain` (or equivalent existing package layout) with types such as `TemplateID`, `Template`, `TemplateCategory`, `TemplateMatch`, `MatchKind`, `ManagedBlock`, `DocumentSnapshot`, `MutationPlan`, `MutationKind`, and `RepositoryGitignoreState`. Template IDs must be stable across display-name changes and should derive from upstream relative paths, e.g. `root/PHP`, `global/macOS`, `community/Java/Gradle`.
2. Define match kinds at minimum: `ManagedExact`, `UnmanagedFull`, `Partial`, `Absent`, and `InvalidManagedBlock`. Do not collapse these into booleans.
3. Define a mutation plan that contains repository ID/root, target path, before SHA-256, before bytes/newline metadata, selected template IDs, exact edits, resulting bytes, and human-readable warnings. The plan is immutable after preview.
4. Define typed errors for concurrent modification, unsafe/symlink target, malformed managed markers, unknown template, catalog unavailable, read-only repository, and ambiguous unmanaged removal.
5. Document the ownership rule: only content inside valid gitwatch-managed blocks is considered owned by gitwatch. Detection of unmanaged template rules does not grant ownership.
6. Add architecture documentation describing how the manager integrates with repository-scoped operations and how it returns control to the canonical status refresh after any write.

## Expected code areas

- `internal/gitignore/domain/*`
- `docs/gitignore-manager.md`

## Required tests

- Table tests for stable template ID parsing/formatting and invalid IDs.
- Serialization/round-trip tests for any persisted catalog/state structures.

## Acceptance criteria

- [ ] Domain types compile without importing TUI packages.
- [ ] Every mutation can be traced to one repository and one before-file hash.
- [ ] Managed, unmanaged-full, partial, and absent states are distinguishable.
- [ ] Ownership rules are documented and testable.
- [ ] No UI code is introduced in this task.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Implemented `internal/gitignore/domain` without TUI, filesystem, Git, or network dependencies.
- Added stable upstream-path template IDs, explicit match kinds and ownership checks, repository-scoped snapshots, SHA-256/newline metadata, immutable-by-copy mutation plans, and typed safety errors.
- Added serialization round-trip, ID validation, ownership, snapshot, and mutation-copy tests.
- Added [the gitignore manager domain contract](../../docs/gitignore-manager.md), including canonical status refresh integration and race-protected write requirements.
- Validation passed: `gofmt`, `GOCACHE=/tmp/gitwatch-go-cache go test ./...`, `GOCACHE=/tmp/gitwatch-go-cache go test -race ./...`, `GOCACHE=/tmp/gitwatch-go-cache go vet ./...`, and `git diff --check`.
- Exception: `make check` reached lint but could not run because the sandbox denied access to Go's existing cache at `/Users/JuanSanchez/Library/Caches/go-build`; formatting and all other gates passed independently.
