# AGENTS.md — Contributor and Agent Contract

## Objective
Maintain `gitwatch` as a production-quality interactive Git workbench. Preserve the responsive TUI, authoritative Git integration, guarded mutations, accessibility behavior, and testable package boundaries when adding or changing features.

## Working rules
1. Read `README.md`, `ARCHITECTURE.md`, `UX_SPEC.md`, and any issue or task governing the change before changing code.
2. When working from `tasks/`, follow numeric order unless the task documents a dependency exception.
3. Each task must end with tests, formatting, linting, and an update to task status.
4. Prefer pure state transformation and testable packages. Keep Bubble Tea models thin where business logic can live elsewhere.
5. Never parse human-formatted Git output when a porcelain/NUL-delimited format exists.
6. Never invoke Git through `sh -c`, `bash -c`, PowerShell string concatenation, or any shell command string.
7. Preserve filenames byte-for-byte as far as Go/Git APIs allow. Test spaces, tabs, unicode, quotes, leading hyphens, renames, and unusual paths.
8. Every mutation command must trigger an authoritative status refresh after completion.
9. Watcher events are hints, not truth. Git command output is authoritative.
10. UI animation must never block input, Git operations, shutdown, or status refresh.
11. Respect terminal capability and `NO_COLOR`; provide reduced-motion configuration.
12. No telemetry in v1.

## Quality gates for every task
- `make check`
- `go test ./...`
- `go test -race ./...` on supported dev platforms where race detector is available
- `go vet ./...`
- formatter clean
- no ignored errors in core paths
- no goroutine leaks in tests that create watchers/processes
- new user-visible behavior documented

## Completion convention
Do not mark an issue or task complete until every acceptance criterion is true or an explicit exception is documented. Commit completed changes as focused, independently verified slices.
