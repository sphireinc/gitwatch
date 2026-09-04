# Task 183: Set and enforce large-scale performance budgets

**Phase:** Hardening
**Depends on:** 124, 125, 154, 174, 178, 180

## Goal

Prove the expanded workbench remains an always-on htop-like tool rather than becoming sluggish as features accumulate.

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

1. Define budgets for status refresh at 1k/10k/50k changed paths, Bubble Tea update/render latency, dashboards at 10/50/100 repos, history/reflog pages, provider refresh and submodule summary.
2. Add local benchmarks without telemetry.
3. Virtualize large lists: status, history, reflog, PRs, operation timeline, repositories, tags, remote branches.
4. Prioritize local status refresh over provider/network/plugin/background operations.
5. Profile allocations, goroutines and child-process count during event storms, checkout/rebase and batch operations.
6. Add regression thresholds with CI variance tolerance rather than brittle exact timings.
7. Document hardware-independent qualitative gates: no unbounded growth, no UI-blocking process calls, bounded queues everywhere.

## Verification

- Benchmarks/stress, goroutine/process leak tests, 50k changed-path fixture, 100-repo registry fixture.

## Acceptance criteria

- [ ] No unbounded list/process/goroutine behavior.
- [ ] Live status remains responsive under documented scale scenarios.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
