# Task 166: Add general external editor application and difftool integration

**Phase:** History and inspection
**Depends on:** 142, 161

## Goal

Let users hand off files and diffs to preferred tools while watcher and repository state remain active.

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

1. Add typed config for editor, file opener and difftool as executable + argv token templates.
2. Support opening current file, repository at selected path, selected historical file via private temporary materialization when needed, and Git difftool for revision/path comparison.
3. Use `internal/platform` for process lifecycle and terminal suspend/resume.
4. Temporary materialized content must use private permissions, bounded size and cleanup.
5. After editor/tool exits, request authoritative refresh in addition to natural watcher events.
6. Do not add arbitrary shell strings in this task.

## Git/process boundary

- `git difftool <left> <right> -- <path>`
- `<configured executable> <typed argv tokens>`

## Verification

- macOS/Linux/Windows adapter tests, paths with spaces/unicode, full-screen terminal restoration.

## Acceptance criteria

- [ ] External tools never require disabling/restarting gitwatch.
- [ ] Process invocation remains argv-safe.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Local progress evidence

- Added version-2 typed tool configuration for editor, opener, and difftool
  executable plus argv-token arrays; placeholders are expanded only inside
  individual argv tokens and never through a shell.
- Added `Ctrl-E`, `Ctrl-O`, and `Ctrl-T` status handoffs. Tool completion
  returns to the Bubble Tea model and requests an authoritative refresh;
  directory rows in tree mode cannot accidentally launch a tool for a stale
  file selection.
- Added a typed Git-native `difftool --no-prompt` command builder and focused
  tests for paths/revisions containing spaces and special characters. Focused
  platform/config/Git/app tests, vet, schema parsing, and diff checks pass.
- Native full-screen editor acceptance remains pending for the remainder of
  this task.
- Added bounded `git show` loading from resolved full commit IDs and private
  0700-directory/0600-file materialization with cleanup after tool exit.
  Comparison-side editor/opener handoff now loads the selected historical file;
  comparison-side and status-side difftool commands remain typed argv paths.
