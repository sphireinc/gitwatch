# Task 137: Implement typed branch merge operations

**Phase:** Merge and conflicts
**Depends on:** 123, 125

## Goal

Add first-class merge from local/remote refs with explicit strategies and no hidden destructive behavior.

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

1. Create `internal/merge` request types: source ref, target=current branch, strategy regular/ff-only/no-ff/squash, optional message.
2. Validate source ref, current branch, active sequencer, worktree preconditions and worktree branch occupancy before start.
3. Never auto-stash. If dirty state blocks merge, explain and offer the existing explicit stash workflow.
4. Execute via operation engine; on conflict transition to sequencer/conflict state rather than generic failure.
5. For squash merge explain that changes remain to be committed and no merge commit is created.
6. After completion refresh status, history, branch graph/divergence and registry summary.

## Git/process boundary

- `git merge <ref>`
- `git merge --ff-only <ref>`
- `git merge --no-ff <ref>`
- `git merge --squash <ref>`
- `git merge --abort`

## Verification

- Fast-forward, merge commit, ff-only refusal, squash, conflict, abort, dirty preflight.

## Acceptance criteria

- [ ] Merge strategy is explicit.
- [ ] Conflict enters common resolver.
- [ ] No implicit stash/reset/force behavior.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
