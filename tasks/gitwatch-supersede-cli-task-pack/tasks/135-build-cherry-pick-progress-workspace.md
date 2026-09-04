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

- [x] Initial progress workspace slice implemented in the repository-scoped conflict/recovery view: Git sequencer metadata now supplies the ordered selected commits, completed/current/pending position, original/current HEAD, and remaining count; Continue/Skip/Abort remain explicit lifecycle actions.
- [x] Implementation commits recorded: `4a9dfd7` (progress projection/rendering) and `ee7764b` (palette/status recovery navigation).
- [x] Exact tested revision recorded: `ee7764b`, with the preceding projection commit `4a9dfd7` included in the tested checkout.
- [x] Focused tests recorded: `TestDetectOperationStateReportsCherryPickProgress` exercises a real multi-commit conflicted cherry-pick; `TestCherryPickViewShowsRepositoryScopedProgress` verifies wide progress rendering and recovery affordances; `TestActiveCherryPickCanReopenProgressFromPalette` verifies Ctrl-P recovery routing without conflict files.
- [x] `go test ./...` recorded through `make check`.
- [x] Race/vet/lint/format evidence recorded through `GOCACHE=/tmp/gitwatch-go-cache make check` (lint reported 0 issues).
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented: full standalone workspace navigation, command-palette reopen, per-commit skipped/conflicted history beyond Git's current sequencer projection, and native/manual acceptance remain outstanding.

## Local progress evidence (task remains active)

- `GOCACHE=/tmp/gitwatch-go-cache make check` passed on macOS arm64 with formatting, lint, full tests, race tests, vet, security, and performance checks.
- The implementation deliberately reuses the existing repository-scoped conflict workspace and authoritative snapshot refresh path; no second status model or render-time Git work was introduced.
- Active cherry-picks now appear in the status summary and can be reopened through the command palette as `Reopen active cherry-pick`, preserving the same repository-scoped snapshot.
- Task 135 is not moved to `tasks/completed` until the remaining workspace/navigation and platform acceptance criteria are proven.
