# Task 134: Implement typed cherry-pick engine

**Phase:** Cherry-pick and history selection
**Depends on:** 123, 133

## Goal

Implement resumable cherry-pick for single commits, ordered commit sets, ranges, and merge commits.

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

1. Create `internal/cherrypick` as a typed adapter over `internal/sequencer`; UI must not build Git argv.
2. Support one SHA and ordered SHA lists. Prefer explicit SHA argv for multi-select; range syntax is allowed only after resolving/previewing its commits.
3. Detect merge commits before execution and require explicit mainline parent selection; never guess `-m`.
4. Capture original HEAD and selected SHAs for operation journal/recovery.
5. Execute through `internal/operations` with per-repo write serialization.
6. On conflict/pause refresh sequencer state; on success refresh status, history, divergence and registry summary.
7. Expose continue/skip/abort through common sequencer actions.

## Git/process boundary

- `git cherry-pick <sha>...`
- `git cherry-pick -m <parent> <merge-sha>`
- `git cherry-pick --continue`
- `git cherry-pick --skip`
- `git cherry-pick --abort`

## Verification

- Single, ordered multi-commit, merge-mainline, empty cherry-pick, conflict, abort.
- Malicious commit subject rendering sanitized.

## Acceptance criteria

- [x] Cherry-pick is argv-based and resumable.
- [x] Merge commits always require mainline selection.
- [x] Every state transition returns to authoritative refresh.

## Completion record

**Status:** Complete

- Implementation commit: `f832c05` (`feat: add typed cherry-pick engine`). The typed adapter is in `internal/cherrypick/engine.go` with Git integration in `internal/git`.
- Exact tested revision: `0266b2b` plus the working-tree Task 134 changes.
- Focused tests: real-repository tests passed for ordered multi-commit cherry-pick, original-HEAD journaling, and mandatory merge-mainline selection.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: the pinned `golangci-lint@v2.12.0` remains unavailable because the environment denies Go build-cache access; no analyzer failure was reported.
- Native/manual evidence: no new terminal workspace was introduced; native conflict UI verification remains deferred to Tasks 135 and 138-143.
- Known limitations/deferred work: the progress workspace and unified conflict resolver are later tasks. The engine exposes continue/skip/abort and probes authoritative Git state after each command; UI scheduling and operation-history presentation remain downstream integration work.
