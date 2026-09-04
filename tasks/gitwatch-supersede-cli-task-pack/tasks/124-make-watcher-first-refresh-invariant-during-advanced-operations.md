# Task 124: Make watcher-first refresh invariant during advanced operations

**Phase:** Foundation
**Depends on:** 123

## Goal

Prevent the new workbench from becoming command-driven. Live filesystem-driven status must remain active and authoritative in every workspace.

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

1. Audit `internal/watch`, refresh coordination, workspace routing, and mutation completion before landing advanced workflows.
2. Add a reusable integration assertion: an external worktree/index/ref change becomes visible without pressing manual refresh while any workspace is open.
3. Keep worktree and necessary Git metadata watching active during rebase/merge/cherry-pick/conflict UIs. Coalesce event storms instead of stopping watchers.
4. Treat watcher events as hints only. Continue the existing low-frequency reconciliation path to recover from missed/drop events.
5. Treat transient lock/read failures during checkout/rebase as retryable refresh conditions with bounded backoff; do not poison repository state permanently.
6. Keep `r` as an escape hatch, never a required workflow step.
7. Add an event-storm budget: checkout/rebase producing thousands of fsnotify events must collapse to bounded status invocations using dirty-bit/dedup logic.

## Verification

- External edit/add/checkout/branch switch/conflict resolution while advanced workspace is open.
- 10k event storm with assertion that status process count remains bounded.
- Polling fallback reproduces the same final transitions when fsnotify is unavailable.

## Acceptance criteria

- [ ] Live status remains active in every new workspace.
- [ ] No advanced workflow owns a private status cache.
- [ ] Event storms recover to one final authoritative snapshot without unbounded Git processes.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
