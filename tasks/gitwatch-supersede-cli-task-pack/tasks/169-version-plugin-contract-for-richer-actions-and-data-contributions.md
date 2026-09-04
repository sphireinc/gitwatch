# Task 169: Version plugin contract for richer actions and data contributions

**Phase:** Extensibility
**Depends on:** 167, 168

## Goal

Extend out-of-process plugins without allowing arbitrary in-process UI code.

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

1. Design plugin protocol vNext with explicit version negotiation and backwards compatibility.
2. Allow plugins to register palette actions, contextual actions, bounded table/detail data, notifications and read-only repository metadata extensions.
3. Host renders schema-defined UI; plugin never injects Go/UI code into process.
4. Declare capabilities for process/network/Git mutation and require config/user approval as appropriate.
5. Keep output/time limits, cancellation and sanitization.
6. Provide dependency-free SDK updates, examples and compatibility fixtures.

## Verification

- Old plugin still works, unknown vNext capability degrades, oversized/control-sequence output bounded.

## Acceptance criteria

- [ ] Plugins are richer while isolation remains a core differentiator.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
