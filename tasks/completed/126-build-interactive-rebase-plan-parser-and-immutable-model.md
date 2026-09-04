# Task 126: Build interactive rebase plan parser and immutable model

**Phase:** Interactive rebase
**Depends on:** 122

## Goal

Represent Git’s todo plan safely and losslessly enough for reordering and action changes without delegating plan editing to a text editor.

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

1. Create `internal/rebase` with `Plan`, `Entry`, `Action`, commit SHA, subject, original index, and records for comments/blank/unknown commands.
2. Support pick, reword, edit, squash, fixup, drop and preserve exec/break/label/reset/merge or unknown directives even if the UI does not edit them yet.
3. Keep raw commit subjects as data and sanitize only in presentation.
4. Implement pure operations: change action, move entry, move range, validate plan, determine squash/fixup target, compute logical groups.
5. Reject invalid first-entry squash/fixup and any transformation that would silently drop unknown directives.
6. Do not execute Git or touch temp files in this package.

## Verification

- Golden todo files from `git rebase -i --root` and rebase-merges scenarios.
- Round-trip comments/unknown directives.
- Property tests: reorder preserves commit-entry multiset and directive ordering constraints.

## Acceptance criteria

- [x] Plan editing is pure/testable and independent of Bubble Tea.
- [x] Unknown todo directives survive round-trip.
- [x] Invalid squash/fixup plans are blocked before execution.

## Completion record

**Status:** Complete

- Implementation commit: `293cfe3` (`feat: add interactive rebase plan model`). The implementation is in `internal/rebase/plan.go` with tests in `internal/rebase/plan_test.go`.
- Exact tested revision: `bfc7e87` plus the working-tree Task 126 changes.
- Focused tests: `GOCACHE=/tmp/gitwatch-go-cache go test ./internal/rebase -v` passed (5 tests).
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: `make check` reached the lint target, but the pinned `golangci-lint@v2.12.0` could not execute because the sandbox denied access to `/Users/JuanSanchez/Library/Caches/go-build`; no lint failure was reported by the project analyzer.
- Native/manual evidence: not applicable; this task adds a pure domain package and does not change terminal interaction.
- Known limitations/deferred work: Git command execution, interactive-rebase orchestration, UI editing, and native terminal verification remain deferred to later tasks. Unknown records are retained and moved losslessly, while only recognized commit actions are editable.
