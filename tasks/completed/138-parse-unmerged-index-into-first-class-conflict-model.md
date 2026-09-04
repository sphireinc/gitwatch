# Task 138: Parse unmerged index into first-class conflict model

**Phase:** Merge and conflicts
**Depends on:** 123, 137

## Goal

Build precise conflict state from Git index stages rather than human-readable error text.

## Non-negotiable constraints

- Live filesystem-driven status is the product core. Do not replace, subordinate, or pause it except for the minimum repository-lock window required by Git itself.
- Filesystem events are refresh hints, never authoritative state. The authoritative worktree snapshot remains `git status --porcelain=v2 -z --branch --untracked-files=all` parsed into immutable repository state.
- Every successful mutation MUST request an authoritative refresh for the affected repository. Long-running sequencer operations must refresh after every observable state transition.
- Multi-repository support is first-class. New domain models and operations MUST carry repository identity/scope and remain correct while other repositories refresh or run unrelated work.
- Do not create unbounded watchers, goroutines, workers, or Git/provider/plugin processes. Reuse bounded registry/operation infrastructure.
- All Git commands use typed argv execution through the Git boundary. Never interpolate repository data into shell command strings. Use `--` where supported and machine-readable/NUL-delimited output where available.
- Bubble Tea owns UI state. Git/network/filesystem/process work never runs in the render path.
- Repository-controlled text is untrusted terminal input and MUST be sanitized before rendering.
- Destructive/history-rewriting actions require scope-specific confirmation. Keep the prohibition on generic `reset --hard`, raw `--force`, and `clean -fd` shortcuts.
- Keyboard and mouse must reach equivalent functionality. New views must work at 80x24, honor `NO_COLOR`, and support full/reduced/off motion.
- Do not reimplement Git. Use Git as source of truth and build safe typed control/presentation layers around it.
- Breaking config/plugin changes require versioning, migration, and compatibility fixtures.

## Implementation steps

1. Create `internal/conflicts` with path, conflict kind, stage-1 base, stage-2 ours, stage-3 theirs blob IDs/modes, working-tree state and resolution state.
2. Load unmerged entries with NUL-safe Git plumbing and correlate with porcelain-v2 conflict status.
3. Represent missing stage entries explicitly for add/delete conflicts.
4. Preserve path bytes and support tabs/newlines/unicode filenames.
5. Do not eagerly load blob content; selected details load on demand with size limits.
6. Expose conflict count/severity into repository snapshot and registry summary.

## Git/process boundary

- `git ls-files -u -z`
- `git status --porcelain=v2 -z --branch --untracked-files=all`

## Verification

- Both-modified, add/add, modify/delete, rename-related fixtures where Git exposes stage data.
- Weird path fixtures.

## Acceptance criteria

- [ ] Conflict UI needs no localized stderr parsing.
- [ ] Missing base/ours/theirs stages are handled correctly.

## Completion record

- [x] Implementation commit recorded: `feat: add first-class unmerged index conflict model`.
- [x] Exact tested revision recorded: implementation commit below.
- [x] Focused unit/integration tests recorded: `go test ./internal/conflicts ./internal/git ./internal/repo`.
- [x] `go test ./...` recorded: passed.
- [x] Race/vet/lint/format evidence recorded where applicable: `go test -race ./...`, `go vet ./...`, `gofmt`, and `git diff --check` passed; `make check` reached lint but the environment denied `/Users/JuanSanchez/Library/Caches/go-build`, so golangci-lint could not start.
- [x] Native/manual evidence recorded where this task changes terminal interaction: not applicable; this is a parser/snapshot boundary.
- [x] Known limitations/deferred work documented: blob contents remain on-demand; conflict UI/resolution actions belong to the common resolver tasks.
