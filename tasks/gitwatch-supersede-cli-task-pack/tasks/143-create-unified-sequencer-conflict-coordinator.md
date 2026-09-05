# Task 143: Create unified sequencer conflict coordinator

**Phase:** Merge and conflicts
**Depends on:** 132, 135, 136, 137, 140, 141

## Goal

Make conflict handling identical across rebase, cherry-pick, revert and merge.

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

1. Create coordinator logic mapping sequencer state + conflict snapshot to valid actions.
2. Derive Continue/Skip/Abort availability from operation kind and current Git state; invalid actions are not rendered.
3. Zero unresolved index entries does not automatically mean Continue is valid; check edit/amend/commit requirements.
4. Route all operation-specific conflict entry points to the same conflict packages.
5. Detect external continue/abort on watcher refresh and update/close views generation-safely.
6. Emit one operation-attention notification path rather than duplicate feature toasts.

## Verification

- Same conflict fixture under merge/rebase/cherry-pick/revert.
- External continue/abort while UI is open.
- Repository switch during conflict.

## Acceptance criteria

- [ ] One resolver/coordinator serves every sequencer workflow.
- [ ] No duplicate conflict parser or lifecycle state machine remains.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Progress evidence

- Centralized recovery action availability in `conflictview.RecoveryActions`.
  Rendering and app lifecycle input now use the same operation-aware decision
  for Continue, Skip, and Abort across rebase, cherry-pick, revert, and merge.
- Continue is withheld when unresolved conflicts remain; for non-rebase
  operations it is also withheld when the authoritative staged count is known
  to be zero. Skip is limited to Git workflows that support `--skip`.
- Focused tests cover unresolved and clean-index gating plus the shared
  lifecycle matrix for all four sequencer workflows.
- Status recovery routing and the progress workspace now include revert and
  merge alongside rebase and cherry-pick; revert uses the same ordered
  progress presentation as cherry-pick.
- A refresh that observes a previously active sequencer disappear with no
  conflicts now closes the recovery workspace to Status and reports the
  externally completed/aborted transition. App coverage exercises this path.
- A revert started from History now follows the same authoritative refresh
  route as merge: when Git leaves a revert sequencer active after a conflict,
  the app opens the shared Revert recovery workspace with the authoritative
  conflict list. App coverage verifies the route and status message.
- Conflict attention notifications are now emitted on entry into a conflicted
  state or operation transition, rather than duplicated on every watcher
  refresh. App coverage verifies the single-notification path.
- Fallback refresh snapshots now carry repository generation metadata and stale
  snapshots are discarded after a repository switch. App coverage verifies
  that an old repository cannot overwrite current recovery state.
- Task remains active: merge/revert entry points, external state transitions,
  repository-switch generation handling, one attention-notification path, and
  native/manual evidence still need completion.
