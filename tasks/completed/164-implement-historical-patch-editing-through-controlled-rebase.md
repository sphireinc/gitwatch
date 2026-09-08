# Task 164: Implement historical patch editing through controlled rebase

**Phase:** History and inspection
**Depends on:** 126, 131, 162

## Goal

Match LZ’s historical line/file removal capability as a transparent, previewable history rewrite.

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

1. Allow building a selected line/hunk patch against a historical commit using existing `internal/patch` and `internal/hunks` identities.
2. Generate an explicit rebase plan that pauses at the selected commit.
3. At pause, build/apply the intended inverse/edit patch only after `git apply --check` succeeds.
4. Amend through existing commit machinery and continue rebase.
5. Show all later commits that will be replayed and warn about possible conflicts/published-history rewrite.
6. If patch application or later replay conflicts, stop in standard edit/conflict recovery; never fuzzy-overwrite silently.
7. Abort must restore original history via normal rebase abort.

## Git/process boundary

- `git apply --check <generated patch>`
- `git apply <generated patch>`
- `git commit --amend`
- `git rebase --continue/--abort`

## Verification

- Remove line, hunk and whole-file change from historical commit; later replay conflict; abort.

## Acceptance criteria

- [x] Historical patch edits are previewed, checked and recoverable.
- [x] No direct hard-reset history surgery is used.

## Completion record

- [x] Implementation commit recorded: `9b48680` (`feat: add guarded historical patch edits`).
- [x] Exact tested revision recorded: `9b48680`.
- [x] Focused unit/integration tests recorded: `GOCACHE=/tmp/git-watch-164-focused-cache ... go test ./internal/git ./internal/app`; passed, including real temporary-repository patch application and app routing coverage.
- [x] `go test ./...` recorded: passed through `make check`.
- [x] Race/vet/lint/format evidence recorded where applicable: `make check` passed formatting, golangci-lint, `go test -race ./...`, `go vet ./...`, `git diff --check`, security checks, and performance checks.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented: maintainer should still exercise line, hunk, whole-file, replay-conflict, and 80x24 mouse/keyboard flows manually before release; published-history rewrites require coordination.

## Progress evidence

- Added a typed Git primitive that validates a selected patch in both worktree
  and index before reversing either copy; no hard reset or shell command is
  involved.
- Historical commit inspection can open a selectable hunk workspace, build a
  selected inverse patch, and generate an explicit `edit` rebase plan with a
  visible later-commit replay warning.
- A paused historical rebase applies the checked inverse patch, opens the
  existing amend composer, and continues through the existing rebase
  continue/abort lifecycle. Focused tests cover patch application and app
  routing; full repository gates passed. Manual terminal acceptance remains a
  deferred maintainer check for the interactive conflict and small-terminal
  flows listed in the completion record.
