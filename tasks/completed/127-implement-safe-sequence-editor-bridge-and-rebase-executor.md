# Task 127: Implement safe sequence-editor bridge and rebase executor

**Phase:** Interactive rebase
**Depends on:** 126, 123

## Goal

Start interactive rebase with a validated gitwatch plan without opening an external todo editor.

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

1. Create a private internal helper mode, e.g. `gitwatch internal sequence-editor`, used only as `GIT_SEQUENCE_EDITOR`.
2. Before starting Git, write the validated plan to an owner-only random temp file associated with a generated operation ID.
3. The helper receives Git’s todo path, verifies operation ID/context, and atomically replaces the todo content. It must refuse arbitrary use outside an active gitwatch operation.
4. Never use `sh -c`, `cmd.exe /C`, PowerShell command text, or path interpolation. Make helper invocation safe on macOS/Linux/Windows.
5. Capture original HEAD/base and schedule the rebase through `internal/operations`.
6. After process exit, request authoritative status + operation-state refresh. Exit code 0 is not enough to infer final state; Git may pause for edit/conflict depending on invocation.
7. Clean private temp files after completion/abort; tolerate crash leftovers without trusting them on restart.

## Git/process boundary

- `git rebase -i <base>`
- `git rebase -i --root`
- `GIT_SEQUENCE_EDITOR=<gitwatch helper boundary>`

## Verification

- Temp-file permission and operation-ID validation tests.
- Paths with spaces/unicode on all supported OSes.
- Rebase completes, pauses at edit, and conflicts.
- Kill gitwatch after rebase starts; restart reconstructs from Git via Task 123.

## Acceptance criteria

- [x] Interactive rebase starts without external todo editor.
- [x] No shell interpolation is used.
- [x] Paused rebase is represented as in-progress rather than generic failure.

## Completion record

**Status:** Complete

- Implementation commit: recorded after this completion note is staged; the executor and helper boundary are in `internal/git/rebase.go`, with the CLI dispatch in `cmd/gitwatch/main.go` and tests in `internal/git/rebase_test.go`.
- Exact tested revision: `bfc7e87` plus the working-tree Task 127 changes.
- Focused tests: `GOCACHE=/tmp/gitwatch-go-cache go test ./internal/git ./cmd/gitwatch -v` passed, including private handoff, atomic replacement, invalid-plan, and operation-ID tests.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: `make check` reached the pinned lint target, but the environment denied the Go build-cache access required to execute `golangci-lint@v2.12.0`; no analyzer failure was reported.
- Native/manual evidence: not applicable; the task adds a process boundary and does not add terminal UI interaction.
- Known limitations/deferred work: UI scheduling through the operation engine, explicit rebase workspace controls, and cross-platform packaged-binary/manual verification remain deferred. The executor probes Git operation state after process exit so a paused/conflicted rebase is exposed as `Paused` with its sequencer state. Handoff files are owner-only, operation-ID scoped, integrity checked, and removed when the executor returns.
