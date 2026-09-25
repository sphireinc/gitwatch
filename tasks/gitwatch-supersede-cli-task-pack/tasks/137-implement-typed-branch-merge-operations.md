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

- [x] Merge strategy is explicit.
- [x] Conflict enters common resolver.
- [x] No implicit stash/reset/force behavior.

## Completion record

- [x] Implementation commit recorded (`06771d6` for the stale-generation guard; earlier feature commits are listed below).
- [x] Exact tested revision recorded (`06771d6`).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded for the post-`34562b5` application source. The owner sign-off for candidate `5b2a8e9` remains candidate-specific and is not carried forward as evidence for this later code change.
- [x] Known limitations/deferred work documented (current-source native/manual acceptance remains open).

## Progress evidence

- The typed merge engine already covers explicit regular, ff-only, no-ff, and
  squash strategies with clean-worktree and active-operation preflight.
- Merge execution and abort now return the authoritative post-command
  `repo.Snapshot`, including conflict and durable-operation state, so callers
  can refresh status without inferring truth from Git stderr.
- The Branches workspace now provides an explicit merge prompt with strategy
  selection (`merge`, `ff-only`, `no-ff`, or `squash`), rejects self-merge and
  dirty-worktree starts, and routes paused conflicts into the common resolver.
- Focused integration tests verify a successful fast-forward snapshot, a
  conflict snapshot entering merge recovery, and an abort snapshot clearing
  the operation.
- Focused app tests cover the clean-worktree guard, strategy validation, and
  branch merge prompt flow.
- Branch merge execution now runs through the repository-scoped operation
  engine, retaining cancellation/serialization lifecycle state; an app
  integration test verifies the real merge and post-merge refresh.
- Merge preflight now rejects merging the current branch into itself and a
  local source branch checked out in another linked worktree, using the
  existing porcelain worktree parser. Integration tests cover both guards.
- The Branches workspace now applies the linked-worktree occupancy guard before
  opening the merge strategy prompt, and app coverage verifies the user-facing
  refusal. Task remains active for native/manual evidence.
- At revision `d7e734b`, the focused merge, application, and integration suites
  passed with `go test ./internal/merge ./internal/app ./internal/integration`,
  covering explicit strategies, dirty/occupied-worktree guards, fast-forward
  and conflict snapshots, abort recovery, and post-merge refresh. Native/manual
  terminal evidence remains open.
- At revision `0c362d1`, added real-repository coverage proving `--no-ff`
  creates a two-parent merge commit, `--squash` leaves staged changes without
  moving `HEAD`, and `--ff-only` refuses divergent history without mutation.
  The Branches workspace now reports the squash outcome accurately: review the
  staged changes and commit them; no merge commit was created. The focused
  merge/app tests and full `GOCACHE=/tmp/gitwatch-go-cache make check` passed
  on Darwin arm64. GitHub Actions run
  [35866061892](https://github.com/sphireinc/gitwatch/actions/runs/35866061892)
  passed quality/policy, full-history secret scan, and Ubuntu, macOS, and
  Windows test jobs. Native/manual terminal evidence remains open.
- The merge request already carried a repository generation, but the typed
  engine did not compare it with the active engine generation before probing
  or mutating Git. `Engine.Execute` now rejects stale-generation requests and
  has a focused test proving the guard runs before Git access. Cross-platform
  local `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache
  make check` passed at `06771d6` (lint 0 issues, normal/race tests, vet,
  formatting, security and performance). Hosted run
  [36059886734](https://github.com/sphireinc/gitwatch/actions/runs/36059886734)
  passed quality/policy, secret scan, and Ubuntu/macOS/Windows tests, including
  Unix PTY acceptance and Windows path/CRLF parity. Native/manual acceptance
  remains open.

## Merged-main implementation audit (2026-09-25)

- The typed merge engine supports regular, fast-forward-only, no-fast-forward,
  and squash strategies. It rejects dirty worktrees, active operations,
  self-merges, stale repository generations, invalid sources, and source
  branches occupied in another linked worktree; it never invokes stash, reset,
  or force behavior.
- The Branches prompt requires an explicit strategy and retains qualified
  remote-tracking names. Merge execution uses the repository operation engine
  and applies an authoritative post-command snapshot. A conflicted result
  enters the shared conflict resolver; squash explains that changes are staged
  without creating a merge commit.
- Coverage includes explicit strategy and clean-worktree checks,
  remote-qualified source handling, stale-generation and worktree-occupancy
  guards, real fast-forward/no-ff/squash outcomes, conflict snapshot and abort,
  and app-level refresh after a real merge.
- The audit found merge recovery could discard repository-discovery or
  post-command snapshot errors. Commit `34562b5` now joins those errors into
  the returned outcome, and `TestAbortReportsRepositoryDiscoveryFailure`
  covers the abort failure path. Error outcomes are surfaced by the app and
  trigger another refresh attempt.
- User-facing guidance now documents the `M` Branches key and merge strategy,
  safety, conflict, abort, and squash behavior in `KEYMAP.md` and
  `docs/advanced-workflows.md`.
- `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache
  make check` passed at application revision `34562b5` on Darwin arm64:
  formatting, lint (0 issues), normal/race tests, vet, security checks, and
  performance budgets.
- Hosted Actions run
  [36143404629](https://github.com/sphireinc/gitwatch/actions/runs/36143404629)
  passed quality/policy, full-history secret scan, and Ubuntu 24.04, macOS 15,
  and Windows 2025 jobs. Unix PTY/large-status and Windows path/CRLF checks
  passed; the Windows race job is skipped by workflow configuration.
- The matrix's owner-provided all-cells sign-off is for candidate
  `5b2a8e9ca35e011f474bc1ddf542d3b98e3aa725`. Since `34562b5` changes
  application code, no terminal evidence for that later source is claimed.
  Task 137 remains active for current-source native/manual acceptance.
