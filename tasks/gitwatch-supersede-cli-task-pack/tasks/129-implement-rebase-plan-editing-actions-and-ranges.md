# Task 129: Implement rebase plan editing actions and ranges

**Phase:** Interactive rebase
**Depends on:** 128

## Goal

Reach LZ-level core rebase ergonomics: pick, squash, fixup, reword, edit, drop, reorder and range operations.

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

1. Add direct actions for pick, squash, fixup, reword, edit, drop.
2. Add move up/down and selected-range move up/down preserving relative order.
3. Add multi-select action application where semantically valid; reject transformations that make the plan invalid.
4. Display squash/fixup grouping/target clearly before execution.
5. Require explicit confirmation for drop and for any rewrite of commits detected as published/reachable from configured remotes.
6. Revalidate after every plan edit; Start remains disabled until valid.
7. Add context-sensitive help and command-palette actions without stealing ambiguous global bindings.

## Verification

- Boundary moves, range moves, invalid first squash/fixup, mixed action plans.
- Published/unpublished confirmation behavior.

## Acceptance criteria

- [ ] All required rebase actions are available in TUI.
- [ ] Invalid plans cannot execute.
- [ ] Range editing is deterministic and keyboard/mouse accessible.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
