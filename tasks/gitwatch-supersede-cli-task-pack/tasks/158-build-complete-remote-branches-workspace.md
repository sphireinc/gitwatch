# Task 158: Build complete remote-branches workspace

**Phase:** Tags and remotes
**Depends on:** 157

## Goal

Expose remote branches as first-class refs with checkout, tracking, merge, rebase, worktree and deletion.

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

1. Load remote refs with bounded ahead/behind relative to local matches when available.
2. Support checkout into a new local tracking branch, confirmed detached checkout, new branch, new worktree, set upstream, merge, rebase and remote delete.
3. Remote delete confirmation names remote and full branch ref.
4. Reuse branch/merge/rebase engines; UI must not duplicate command construction.
5. Filter/sort by remote, branch name, divergence and last commit.
6. Always display remote-qualified branch names when ambiguity exists.

## Verification

- Multiple remotes with same branch names, remote delete, tracking branch creation.

## Acceptance criteria

- [ ] Remote branch actions reuse typed engines.
- [ ] Ambiguous names cannot target the wrong remote.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
