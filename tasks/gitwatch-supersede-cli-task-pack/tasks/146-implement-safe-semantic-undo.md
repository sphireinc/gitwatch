# Task 146: Implement safe semantic undo

**Phase:** Recovery and undo
**Depends on:** 145, 143

## Goal

Offer Undo only where gitwatch can prove a safe recovery point; refuse when repository state has diverged.

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

1. Define undo policy per operation instead of a generic reset button.
2. Examples: abort active sequencer; undo a just-created local commit while preserving worktree/index via soft/mixed semantics; restore pre-rebase branch ref only when reflog/current-state checks prove no newer work is lost.
3. Before undo compare current HEAD/index/worktree assumptions with recorded post-operation state. If diverged, refuse automatic undo and open guided recovery.
4. Never use unconditional `reset --hard`.
5. Require confirmation naming old/new HEAD/ref and effect on index/worktree.
6. Execute through operation engine, refresh authoritative state, and record undo as a journal entry.

## Git/process boundary

- `Operation-specific git reset --soft/--mixed only where policy proves safety`
- `git rebase/merge/cherry-pick/revert --abort for active operations`

## Verification

- Undo immediately after local commit, refusal after unrelated commit/change, sequencer abort, multi-repo isolation.

## Acceptance criteria

- [x] Undo preserves working content by default.
- [x] Stale/unsafe recovery is refused rather than guessed.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added an operation-specific undo policy in `internal/undo` that only permits
  a recorded successful local commit to be undone with typed `reset --soft`.
  The policy requires repository identity, branch/ref, old/new object IDs,
  no active sequencer, and an exact post-operation index/worktree fingerprint.
- Added semantic journal metadata for the post-operation fingerprint on commit
  and cherry-pick records, plus a journal `u` confirmation flow and bounded
  operation-engine execution that refreshes authoritative status and records
  the undo result.
- `TestExecuteUndoImmediatelyPreservesWorktreeContent` verifies a real local
  repository returns HEAD to the prior commit while preserving a subsequent
  worktree edit; policy unit tests cover divergence and active-operation
  refusal. `TestExecuteRefusesUnrelatedCommitAfterRecordedOperation` and
  `TestPlanRefusesForeignRepository` cover stale HEAD and repository-scope
  isolation.
- The implementation deliberately refuses journal Undo while a sequencer is
  active; users get the operation-specific abort action from the shared
  sequencer workspace instead of an unsafe reset. Active-operation refusal is
  covered by a real merge-conflict repository test.
- The focused implementation commit is `99495e9`; the full `make check` gate
  passed at that revision after the stale-HEAD and repository-scope tests were
  added. Native/manual terminal evidence remains the explicit outstanding
  completion gate.

- Revision `32b1756` adds a real merge-conflict integration test around
  `undo.Execute`. It reconstructs the repository snapshot with an active
  sequencer and verifies the policy returns `ErrActiveOperation` before any
  reset can be attempted; normal and race-focused tests pass. Native/manual
  terminal evidence remains open.
- Revision `7a78054` strengthens the refusal tests to assert `ErrDiverged` for
  changed HEAD/content and `ErrActiveOperation` for a live merge, and confirms
  the refused operation leaves that merge and its HEAD intact. It adds
  `TestExecuteUndoIsIsolatedFromOtherRepository`, which performs a real soft
  undo in repository A and verifies repository B's HEAD and worktree are
  unchanged. `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  make check` passed on Darwin arm64, including lint (0 issues), full and race
  test suites, vet, format, security, performance, and diff checks. GitHub
  Actions run
  [35867831486](https://github.com/sphireinc/gitwatch/actions/runs/35867831486)
  passed quality/policy, secret scanning, and Ubuntu, macOS, and Windows jobs.
  Native/manual terminal evidence remains required before task completion.
