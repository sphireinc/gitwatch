# Task 147: Implement redo and guided recovery

**Phase:** Recovery and undo
**Depends on:** 146

## Goal

Add redo only for operations whose typed intent and current repository state make replay provably safe; otherwise guide the user through reflog recovery.

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

1. Store replayable intent on gitwatch-generated undo entries.
2. Validate current state matches the expected post-undo state before replay.
3. For non-replayable cases show reflog candidates, compare actions, and create-branch recovery rather than pretending generic redo exists.
4. Do not synthesize shell commands.
5. Expose Undo/Redo through operation timeline and command palette with disabled-reason text.

## Verification

- Commit undo/redo, redo refusal after external commit, rebase guided recovery fallback.

## Acceptance criteria

- [x] Redo exists only where safety can be established.
- [x] Unavailable redo has an actionable recovery path.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added `internal/redo` with a typed replay policy for successful `undo commit`
  journal records. It requires repository/ref identity, the expected post-undo
  HEAD, no active sequencer, and an exact post-undo snapshot fingerprint before
  allowing `reset --soft` to the recorded commit.
- Added journal `R` confirmation/execution routing through the operation engine,
  authoritative refresh, and semantic redo result recording.
- When the selected journal operation is not safely replayable, `R` now opens
  the existing repository-scoped reflog workspace with compare (`d`) and
  create-branch (`B`) recovery guidance; app coverage verifies this route.
- `TestExecuteRedoReplaysSuccessfulSoftUndo` exercises a real repository
  round-trip; unit coverage verifies the typed command and stale-state refusal.
- Replay is intentionally limited to a successful gitwatch soft-undo record;
  other operations route to the repository-scoped reflog workspace for compare
  and branch recovery rather than receiving generic redo behavior. Native/manual
  terminal evidence remains outstanding.
- The guarded redo slice is committed at `0ec98a9`, with guided reflog routing
  added at `c8deeb4`; the full `make check` gate passed at both validation
  points. Additional external-commit refusal coverage is recorded below;
  native/manual terminal evidence remains open.
- At revision `d6ad55f`, added `TestExecuteRefusesRedoAfterExternalCommit`: after
  a real soft undo, it creates an external commit and proves redo returns
  `ErrDiverged` without changing the external `HEAD` or repository status.
  Focused `go test ./internal/redo -count=1` and
  `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache make
  check` passed on Darwin arm64; lint reported 0 issues and the gate completed
  full/race tests, vet, security, performance, formatting, and diff checks.
  GitHub Actions run
  [35868810670](https://github.com/sphireinc/gitwatch/actions/runs/35868810670)
  passed quality/policy, secret scanning, and Ubuntu, macOS, and Windows jobs.
  Native/manual terminal evidence remains the completion gate.
