# Task 201: Integrate `.gitignore` management with multi-repository mode

## Objective

Make the feature repository-aware everywhere and add safe batch workflows for users managing many repositories.

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

gitwatch’s identity includes multi-repository dashboards. `.gitignore` management must not regress into a single-current-working-directory feature. Each repository has independent file state, catalog matches, previews, errors, and operations.

## Implementation steps

1. Bind all gitignore manager models to `RepoID` and immutable repository root. Switching repositories must cancel/ignore stale async results using the project’s existing generation/request ID pattern.
2. Add multi-repo dashboard columns/indicators for `.gitignore`: absent, managed count, unmanaged matches, partial/attention, update available. Keep the default dashboard compact; detailed counts can live in a secondary column or inspector.
3. Allow opening the manager for any selected repository from the multi-repo dashboard without changing process cwd globally.
4. Add an explicit batch action: apply selected templates to selected repositories. This must build one mutation plan per repo, show a per-repo dry-run summary, and require confirmation before executing bounded-concurrency operations.
5. Batch execution must be partial-failure tolerant: one read-only/symlink/conflicted repository does not stop unrelated repositories. Report succeeded/skipped/failed per repo.
6. Do not add a dangerous `apply to all discovered repos` one-key shortcut. Repository selection must be explicit.
7. After each successful repo mutation, enqueue that repo’s normal authoritative Git refresh; do not globally refresh every repository unnecessarily.

## Expected code areas

- `internal/multirepo/*`
- `internal/tui/gitignore/multirepo.go`

## Required tests

- Switch repo while catalog/match computation is in flight.
- Batch apply to success + read-only + concurrent-modification repos.
- No global cwd dependency.

## Acceptance criteria

- [ ] Every manager state is repo-scoped.
- [ ] Multi-repo dashboard can expose `.gitignore` health/status.
- [ ] Batch apply requires explicit repository selection and preview.
- [ ] Failures are isolated per repository.
- [ ] Only affected repositories refresh after mutations.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
