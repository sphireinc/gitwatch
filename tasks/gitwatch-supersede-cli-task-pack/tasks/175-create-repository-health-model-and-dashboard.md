# Task 175: Create repository health model and dashboard

**Phase:** Multi-repository differentiation
**Depends on:** 125, 151, 155, 172

## Goal

Turn gitwatch’s htop identity into a concrete repository-health surface.

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

1. Create `internal/health` with cheap local metrics: clean/dirty/conflict, ahead/behind, unpushed count, stash count, active sequencer, worktree count, submodule health, last fetch age and signing/config warnings.
2. Provider enrichments are optional cached fields: PR state, checks, review attention.
3. Remote latency is recorded from actual network operations or an explicit probe; never ping remotes every status refresh.
4. Compute semantic attention severity rather than a fake numeric “health score”.
5. Expose health in single-repo header/details and multi-repo dashboard.
6. Every non-live metric records freshness timestamp/source so stale provider/network data is obvious.
7. Local status-derived metrics update from the authoritative snapshot.

## Verification

- Freshness behavior, provider disabled, offline mode, large registry.

## Acceptance criteria

- [ ] Dashboard remains valuable offline.
- [ ] Live local health is status-derived; cached external health is labeled stale/fresh.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
