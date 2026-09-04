# Task 198: Implement the new-repository `.gitignore` creation flow

## Objective

Give repositories with no `.gitignore` an excellent first-run creation experience using one or many catalog combinations.

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

A freshly initialized repository should not require leaving gitwatch to create `.gitignore`. This should feel like a native setup action, while never blocking the primary live status dashboard.

## Implementation steps

1. When the active repository has no root `.gitignore`, expose a non-intrusive status/dashboard affordance such as `No .gitignore · I create`. Do not display a modal automatically on startup.
2. Opening the manager in this state launches directly into multi-select catalog mode with project recommendations pinned near the top.
3. Allow selecting any combination of Common, Global, or Community templates. Label Global templates clearly because GitHub describes them as editor/tool/OS rules often suitable for global ignores; do not forbid project-local use.
4. Before creation show a complete preview of the resulting file and selected source templates. Confirm once for the atomic multi-template create.
5. After write, return focus to the repository status view and trigger the standard authoritative status refresh. Files newly ignored by the created file should disappear/change state through the normal status pipeline, not through manual TUI filtering.
6. If another process creates `.gitignore` while the wizard is open, the hash/absence precondition fails and the UI must reload as an existing-file flow rather than overwrite it.

## Expected code areas

- `internal/tui/gitignore/create.go`
- `internal/tui/status/*`

## Required tests

- No-file create flow.
- Concurrent external creation.
- Status refresh message emitted after success.

## Acceptance criteria

- [ ] A brand-new repo can create `.gitignore` without leaving gitwatch.
- [ ] Creation supports multiple templates in one operation.
- [ ] No startup-blocking wizard is introduced.
- [ ] Concurrent creation is safe.
- [ ] Post-create status is sourced from Git, not synthetic UI changes.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added `PlanCreateTemplates` and `Create` to make no-file creation explicit, multi-template, previewable, and refusal-safe when the target is no longer absent.
- Added the app flow for the non-blocking missing-file affordance, asynchronous creation preview, one-time confirmation, atomic guarded write, focus return to Status, and the standard authoritative refresh command after success.
- Added concurrent external-creation protection and reload-compatible error handling; an external `.gitignore` is never overwritten by the creation plan.
- Added tests for multi-template no-file creation, external creation races, and the complete Bubble Tea route through preview, confirmation, write, and refresh emission.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
- `make check` reached lint but could not run because the sandbox denied the Go build cache path under `/Users/JuanSanchez/Library/Caches/go-build`; this remains an environment exception.
