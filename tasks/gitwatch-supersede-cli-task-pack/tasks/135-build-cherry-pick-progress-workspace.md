# Task 135: Build cherry-pick progress workspace

**Phase:** Cherry-pick and history selection
**Depends on:** 134, 143

## Goal

Provide visible progress and recovery instead of reducing multi-commit cherry-pick to a toast.

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

1. Create a workspace showing selected commits in application order with pending/current/completed/skipped/conflicted states.
2. Show source ref, target branch, original HEAD, current/result HEAD and remaining count.
3. Expose Continue/Skip/Abort only when current Git state allows them.
4. On conflict route to unified conflict resolver and return to progress after resolution.
5. Keep the status summary live and allow navigation away/back without losing the active operation.
6. Allow Ctrl-P to reopen any active cherry-pick for the selected repository.

## Verification

- 80x24 and wide layouts.
- Conflict at first/middle/last selected commit.
- Navigate to another workspace and back.

## Acceptance criteria

- [ ] User can see exactly which commit failed and what remains.
- [ ] No modal traps the user.
- [ ] External resolution appears automatically.

## Completion record

- [x] Repository-scoped progress workspace and dedicated `workspace.CherryPick` navigation provide ordered selected commits, completed/current/pending/skipped/conflicted state, original/current HEAD, remaining count, and explicit lifecycle actions.
- [x] Implementation commits recorded across progress projection, palette/status recovery navigation, per-commit outcomes, typed start/journal flow, and dedicated workspace routing (including `4a9dfd7`, `83f4726`, `56e1173`, and `b03e46d`).
- [x] Exact tested revision recorded: `122c8de`, including the full preceding cherry-pick progress implementation.
- [x] Focused tests recorded: real multi-commit conflict progress/resume, wide and 80x24 rendering, palette recovery, navigation away/back, typed start/journal flow, and external resolution refresh/Continue input.
- [x] `go test ./...` recorded through `make check` at `122c8de`.
- [x] Race/vet/lint/format evidence recorded through `GOCACHE=/tmp/gitwatch-go-cache make check` at `122c8de` (lint reported 0 issues).
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented: richer conflicted-history presentation beyond Git's current sequencer projection and native/manual acceptance remain outstanding.

## Local progress evidence (task remains active)

- `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 with formatting, lint, full tests, race tests, vet, security, and performance checks.
- The implementation deliberately reuses the existing repository-scoped conflict workspace and authoritative snapshot refresh path; no second status model or render-time Git work was introduced.
- Active cherry-picks now appear in the status summary and can be reopened through the command palette as `Reopen active cherry-pick`, preserving the same repository-scoped snapshot.
- Completed and skipped commit IDs are normalized against the sequencer's ordered todo backup, including abbreviated/full SHA forms, so the progress view can render exact per-commit outcomes without relying on toast history.
- Recovery controls are now operation-specific: Skip is shown only for rebase, cherry-pick, and revert, while unsupported operations such as Merge expose only Continue and Abort.
- A current cherry-picked commit is labeled `conflicted` when Git reports conflicted paths; the progress view can be left for Status and reopened through Ctrl-P without losing the operation projection.
- Task 135 remains active until operator-owned native/manual acceptance is recorded.

## Additional implementation evidence

- Added guarded history selection/basket `P` action with explicit confirmation,
  typed `internal/cherrypick` execution through the repository operation
  engine, authoritative post-command snapshot capture, conflict-workspace
  routing, and completion activity journaling.
- `TestCherryPickSelectionRunsThroughEngineAndJournalsCompletion` exercises a
  real feature commit selected from history, cherry-picks it onto `main`, and
  verifies the resulting file and semantic journal record. At that point
  standalone workspace/navigation and native/manual acceptance remained open;
  the dedicated route/navigation coverage below resolves the former.
- The cherry-pick start/journal slice was validated at revision `56e1173`
  through the full `make check` gate.
- Added a dedicated `workspace.CherryPick` route backed by the same
  repository-scoped conflict/progress model, with palette reopen, status
  navigation, and lifecycle input preserved while leaving and returning.
- The Status `C` recovery shortcut now selects the dedicated cherry-pick route
  whenever the authoritative operation kind is cherry-pick; other sequencer
  kinds continue using the unified conflict route.
- At revision `b03e46d`, the focused application, Git-boundary, and integration
  suites passed with `go test ./internal/app ./internal/git
  ./internal/integration`, including cherry-pick progress, restart recovery,
  conflict continuation, and navigation tests. Native/manual terminal evidence
  remained open at that revision.
- At revision `122c8de`,
  `TestExternalCherryPickResolutionEnablesContinueFromFreshSnapshot` verifies
  that unresolved conflicts withhold Continue, then an authoritative snapshot
  reflecting external resolution enables the 80x24 Continue footer and `c`
  input without leaving the progress workspace. The full local
  `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64, including
  lint (0 issues), full and race tests, vet, formatting, security fuzz,
  performance, and diff checks. Hosted Actions run `35864391354` passed
  quality/policy, full-history secret scan, and Ubuntu/macOS/Windows matrices;
  macOS PTY acceptance passed. Native operator acceptance remains open.
