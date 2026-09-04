# Task 125: Extend multi-repository state for advanced Git sessions

**Phase:** Foundation
**Depends on:** 122, 123, 124

## Goal

Make advanced workflows multi-repo aware from day one rather than retrofitting them after single-repository implementations ship.

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

1. Extend repository registry summaries with operation/session badges: rebasing, merging, cherry-picking, reverting, bisecting, conflicted, dirty, ahead/behind, provider/check attention.
2. Keep registry summaries cheap. Do not load full rebase plans/diffs/conflict blobs for every repo; load details only for the active repository.
3. An active operation in repository A must remain visible in the repository dashboard while the user is working in repository B.
4. Serialize conflicting writes per repository while allowing unrelated repositories to refresh/fetch/operate concurrently under existing bounded worker limits.
5. Add command-palette jump targets for repositories requiring attention.
6. Define attention priority: unresolved conflicts/sequencer state > failed operation > dirty/diverged > stale provider/network metadata.
7. Preserve missing/broken repository isolation; one bad repo remains one bad row.

## Verification

- 20 disposable repositories with mixed clean/dirty/rebase/conflict states.
- Concurrent independent operations in two repositories.
- Broken/missing repo does not block healthy refresh.

## Acceptance criteria

- [x] Repositories dashboard surfaces advanced-operation state without entering
  the repo through operation and attention badges.
- [x] No global singleton is introduced for advanced Git UI state; rows derive
  from each repository's snapshot and scoped result.
- [x] Per-repo write serialization and bounded cross-repo concurrency remain
  intact through the existing registry and operation infrastructure.

## Status

Complete — extended registry summaries and the repositories view with
repository-scoped operation, conflict, failed-operation, dirty/diverged, and
provider-stale attention badges. Added attention-aware filtering and
command-palette jump targets while retaining bounded refresh workers and
independent broken-repository rows.

## Completion record

- [x] Implementation commit recorded: `f05eda9` (`feat: surface
  multi-repository operation attention`).
- [x] Exact tested baseline recorded: current `main` at `45f621b` plus the
  Task 125 working-tree changes.
- [x] Focused tests: registry dashboard priority/filter tests, 20-repository
  mixed-attention isolation test, repository-view badge test, and app
  command-palette attention-jump test passed.
- [x] `GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_COUNT=1
  GIT_CONFIG_KEY_0=commit.gpgsign GIT_CONFIG_VALUE_0=false
  GOCACHE=/tmp/gitwatch-go-cache go test ./...` passed.
- [x] `GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_COUNT=1
  GIT_CONFIG_KEY_0=commit.gpgsign GIT_CONFIG_VALUE_0=false
  GOCACHE=/tmp/gitwatch-go-cache go test -race ./...` passed for the current
  implementation; focused registry race coverage also passed after the final
  test addition.
- [x] `GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_COUNT=1
  GIT_CONFIG_KEY_0=commit.gpgsign GIT_CONFIG_VALUE_0=false
  GOCACHE=/tmp/gitwatch-go-cache go vet ./...` passed.
- [x] `gofmt` and `git diff --check` passed.
- [ ] Lint: pinned `make check` lint remains unable to download golangci-lint
  v2.12.0 because `proxy.golang.org` is unavailable.
- [ ] Native/manual evidence: dashboard and palette interaction changed;
  native terminal evidence remains operator-owned.
- [x] Known limitations/deferred work documented: provider attention is a
  registry input for future provider refresh integration, operation failures
  are represented by an explicit scoped flag, and detailed operation views
  remain deferred to later tasks.
