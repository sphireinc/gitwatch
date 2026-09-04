# Task 161: Implement arbitrary revision comparison

**Phase:** History and inspection
**Depends on:** 133, 125

## Goal

Allow users to compare any two commits, branches, tags, remote refs or reflog entries without checking them out.

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

1. Create `internal/compare` request with left/right revisions resolved to immutable commit SHAs before detail loading.
2. Load metadata, changed files, numstat, rename/copy status where bounded, and selected-path/full patch using existing diff budgets.
3. Allow assigning A/B from History, Tags, Branches, Remote Branches and Reflog.
4. Show summary, changed-file list, selected-file diff and commit metadata.
5. Do not accept repository-controlled raw revision expressions without validation/resolution.
6. Keep compare read-only, cancellable and generation-scoped.

## Git/process boundary

- `git rev-parse --verify <ref>^{commit}`
- `git diff --name-status -z <left> <right>`
- `git diff --numstat -z <left> <right>`
- `git diff <left> <right> -- <path>`

## Verification

- Rename, binary, huge diff, weird paths, merge commits.

## Acceptance criteria

- [ ] Any two resolved revisions can be compared with no checkout.
- [ ] Compare never blocks live status refresh.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
