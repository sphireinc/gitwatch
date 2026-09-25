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

- [x] User can see exactly which commit failed and what remains.
- [x] No modal traps the user.
- [x] External resolution appears automatically.

## Completion record

- [x] Repository-scoped progress workspace and dedicated `workspace.CherryPick` navigation provide ordered selected commits, completed/current/pending/skipped/conflicted state, original/current HEAD, remaining count, and explicit lifecycle actions.
- [x] Implementation commits recorded across progress projection, palette/status recovery navigation, per-commit outcomes, typed start/journal flow, and dedicated workspace routing (including `4a9dfd7`, `83f4726`, `56e1173`, and `b03e46d`).
- [x] Exact tested revision recorded: `122c8de`, including the full preceding cherry-pick progress implementation.
- [x] Focused tests recorded: real multi-commit conflict progress/resume, wide and 80x24 rendering, palette recovery, navigation away/back, typed start/journal flow, and external resolution refresh/Continue input.
- [x] `go test ./...` recorded through `make check` at `122c8de`.
- [x] Race/vet/lint/format evidence recorded through `GOCACHE=/tmp/gitwatch-go-cache make check` at `122c8de` (lint reported 0 issues).
- [x] Owner-provided cross-platform operator acceptance recorded for the exact application source; Linux is accepted by the documented equivalence disposition, not represented as a physical Linux transcript.
- [x] Known limitation documented: richer conflicted-history presentation beyond Git's current sequencer projection remains deferred; operator acceptance is recorded below.

## Implementation progress evidence

- `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 with formatting, lint, full tests, race tests, vet, security, and performance checks.
- The implementation deliberately reuses the existing repository-scoped conflict workspace and authoritative snapshot refresh path; no second status model or render-time Git work was introduced.
- Active cherry-picks now appear in the status summary and can be reopened through the command palette as `Reopen active cherry-pick`, preserving the same repository-scoped snapshot.
- Where Git retains an ordered todo backup, completed and skipped source IDs
  are normalized across abbreviated/full SHA forms. Git may instead retain
  only pending todo entries and `sequencer/head`; in that case the loader now
  shows completed result commit IDs from the bounded original-HEAD-to-HEAD
  range. It does not fabricate lost source or skipped IDs.
- Recovery controls are now operation-specific: Skip is shown only for rebase, cherry-pick, and revert, while unsupported operations such as Merge expose only Continue and Abort.
- A current cherry-picked commit is labeled `conflicted` when Git reports conflicted paths; the progress view can be left for Status and reopened through Ctrl-P without losing the operation projection.
- The acceptance criteria are verified below; the user-provided operator sign-off and its Linux-equivalence scope are recorded in the merged-main acceptance section.

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
- Revision `8a112fb` fixes a real middle-conflict case: after Git had already
  applied the first of three selected commits, the previous loader displayed
  zero completed, omitted that commit, and left Original HEAD blank. The
  loader now reads Git's `sequencer/head`, uses a bounded `rev-list` to show
  completed result commits when `done`/`todo.backup` are absent, and keeps
  current/pending source commits distinct. The 80x24 recovery footer exposes
  only valid actions and remains visible with notices; the target branch is
  shown, while an unrecoverable source ref is explicitly labeled unavailable.
  A fresh-runner real-repository test and a native Darwin arm64 tmux fixture
  verify a three-commit sequence with a middle conflict, Status navigation,
  reopening, Skip, final branch/HEAD and clean quit under `NO_COLOR=1`.
  Full `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  make check` passed (lint 0 issues, full/race tests, vet, format, diff,
  security, performance). The fixture is wired into Ubuntu/macOS CI.
  Hosted Actions run `36013796348` at `8d1defa` passed quality/policy,
  secret scan, Ubuntu/macOS tests and PTY acceptance, but Windows timed out
  in `TestWatcherSeesExternalGitMetadataAndRecreatedDirectory` during
  fsnotify metadata-watch repair. Commit `46ee853` drains backend channels
  while repairing those watches; local macOS arm64 `make check` passed, and
  hosted run `36016377594` passed all five jobs, including Windows tests and
  path/CRLF parity plus Ubuntu/macOS PTY acceptance. A subsequent
  real-repository test at `6617044` covers first- and last-selected-commit
  conflicts after restart, reconstructs their original HEAD/current
  position/remaining count, skips the conflicted commit, and verifies the
  other selected commits apply. Local macOS arm64 `make check` passed at that
  revision, and hosted run `36057387861` passed quality/policy, secret scan,
  and Ubuntu/macOS/Windows test matrices (including Unix PTY and Windows
  path/CRLF checks).
  At that revision, human-operator and native Windows terminal evidence
  remained open; the later owner sign-off is recorded below.

## Merged-main acceptance (2026-09-25)

- The watcher event handler requests `m.refresh()` and resubscribes. The
  refresh obtains a fresh Git snapshot through the repository refresh
  coordinator (or `git.Snapshot` fallback), so filesystem notifications remain
  hints and Git remains authoritative.
- `TestCherryPickViewShowsRepositoryScopedProgress` asserts completed and
  conflicted commits, the remaining count, and recovery control. The
  80x24 `TestExternalCherryPickResolutionEnablesContinueFromFreshSnapshot`
  verifies that the authoritative resolved snapshot enables Continue and that
  the user can invoke it without leaving the workspace. The route/navigation
  test proves Status and the palette can leave and reopen progress without a
  modal trap. The real-repository first/middle/last conflict restart tests
  cover reconstruction of current and remaining commits.
- `make check` passed on merged main source revision
  `be2d1d999ac3b3e62eafb59bd11b6f1703aac4c6`; hosted Actions run
  [36137333931](https://github.com/sphireinc/gitwatch/actions/runs/36137333931)
  passed quality/policy, full-history secret scan, and Ubuntu/macOS/Windows
  matrices. Windows race testing is skipped by the workflow.
- The beta matrix records the user's explicit all-cells operator sign-off for
  candidate `5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725`; tracked differences
  from that candidate through merged main are documentation/task records, not
  application code. The sign-off therefore covers this task's application
  source. Linux is recorded as accepted by equivalence while physical Linux
  testing continues; no raw terminal transcript is claimed here.
- The known limitation remains explicit: Git's sequencer projection cannot
  reconstruct richer conflicted-history details than Git retains. This is
  documented and does not block the listed acceptance criteria.

Task 135 is complete on merged main under the recorded owner acceptance.
