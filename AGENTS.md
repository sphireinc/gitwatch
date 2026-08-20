# AGENTS.md — Implementation Contract

## Objective
Build `gitwatch` from greenfield repository to production-quality v1. Do not reduce scope by replacing the TUI with a basic list, periodic `git status`, or a thin command launcher.

## Working rules
1. Read `README.md`, `ARCHITECTURE.md`, `UX_SPEC.md`, and the current task before changing code.
2. Work tasks in numeric order unless a dependency exception is documented.
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
- `go test ./...`
- `go test -race ./...` on supported dev platforms where race detector is available
- `go vet ./...`
- formatter clean
- no ignored errors in core paths
- no goroutine leaks in tests that create watchers/processes
- new user-visible behavior documented

## Completion convention
Each task file contains acceptance criteria. Do not mark a task complete until every criterion is true or an explicit exception is documented.
