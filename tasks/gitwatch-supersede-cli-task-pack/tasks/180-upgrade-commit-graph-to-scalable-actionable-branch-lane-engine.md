# Task 180: Upgrade commit graph to scalable actionable branch-lane engine

**Phase:** Multi-repository differentiation
**Depends on:** 161, 179

## Goal

Make the history graph competitive with LZ while retaining bounded loading and semantic color safety.

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

1. Separate DAG topology/lane assignment from rendered glyph/color segments.
2. Handle merges, lane reuse, ref/tag/head decorations and pagination continuity. Support octopus merges where feasible without corrupt topology.
3. Keep raw Git SGR/control sequences out of terminal; theme owns final styling.
4. Enable range selection and commit basket actions directly in graph/history.
5. Wire compare, cherry-pick, rebase-base selection, branch/worktree/tag actions to existing typed engines.
6. Keep max-commit/page bounds and cancellation.
7. Add ASCII/NO_COLOR graph fallback.

## Verification

- Golden DAG fixtures including merges/crisscross/octopus, pagination lane continuity, narrow/NO_COLOR.

## Acceptance criteria

- [ ] Graph is actionable and remains bounded under large history.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
