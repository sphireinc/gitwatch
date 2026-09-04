# Task 197: Build the searchable multi-select template browser TUI

## Objective

Create the primary `.gitignore` catalog browser with instant search, multi-selection, match indicators, and pinned installed combinations.

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

The user requirement is explicit: type `php` instead of scrolling; select multiple combinations at once; combinations matching the current `.gitignore` appear at the top with an asterisk. The browser must remain useful on 80×24 terminals and beautiful on larger screens.

## Implementation steps

1. Add a dedicated gitignore workspace/modal reachable from the command palette and a direct keybinding selected through the existing keymap registry. Prefer `I` only if it is unbound; do not steal an existing binding.
2. Render rows with status indicator, display name, category/source path, and concise match info. Proposed visual semantics: `*` full installed/matched, `~` partial, `+` selected to add, `-` selected to remove, `!` malformed/attention. Do not use color as the sole differentiator.
3. Sort full matches first, then repository-detected recommendations, then all other catalog entries. Within each group sort deterministically. When search is active, preserve the same group priority among matching results.
4. Implement immediate fuzzy filtering as the user types. Search names, aliases, category and source path. Show result count and current query.
5. Space toggles selection without closing the browser. Support selecting multiple templates across categories and filtered views; selection survives changing the search query.
6. Provide tabs or filters for `All`, `Common`, `Global`, `Community`, `Installed/Matched`, and optionally `Recommended`. Do not force users through nested directory trees.
7. Right-side/details pane should show template source, bundled upstream commit, current state, match coverage, update availability, and a scrollable preview of the template contents.
8. Show a footer with context-aware actions: add selected, remove selected, adopt/manage eligible unmanaged match, preview, clear selection, refresh catalog, help.
9. Mouse interaction must have keyboard parity. Clicking a row selects/focuses; checkbox/status hit targets may toggle selection, but destructive actions still go through preview/confirmation.

## Expected code areas

- `internal/tui/gitignore/*`
- `internal/tui/keymap/*`

## Required tests

- Bubble Tea model tests for filtering, selection persistence, sorting and status indicators.
- Golden/render tests at 80×24, 120×40, and wide layouts where the project uses snapshot tests.

## Acceptance criteria

- [ ] `php` search can locate PHP immediately.
- [ ] Multiple combinations can be selected before preview/apply.
- [ ] All full matches are pinned to the top and show `*`.
- [ ] Partial matches are visually distinct.
- [ ] 80×24 layout remains functional.
- [ ] Keyboard-only users can perform every action.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Completion record

- Added the repository-scoped `internal/ui/gitignoreview` model with embedded-catalog search, alias/category/source matching, deterministic full/partial/recommended ordering, tabs, persistent multi-selection, semantic indicators, keyboard controls, mouse row parity, and bounded details/preview rendering.
- Added Bubble Tea routing through the existing workspace and command palette, with direct `I` access and an asynchronous `.gitignore` read/match command. Rendering remains free of filesystem, Git, and network work.
- Added tests covering PHP search, selection persistence, match ordering and indicators, repository isolation, and mouse/keyboard parity.
- Validation passed: `gofmt`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check`.
- `make check` reached lint but could not run because the sandbox denied the Go build cache path under `/Users/JuanSanchez/Library/Caches/go-build`; this is an environment exception, not a lint result.
