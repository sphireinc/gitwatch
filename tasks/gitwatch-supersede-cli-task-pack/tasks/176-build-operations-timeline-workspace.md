# Task 176: Build operations timeline workspace

**Phase:** Multi-repository differentiation
**Depends on:** 145, 146, 174

## Goal

Expose Git, network, provider, plugin and custom-command work as a searchable htop-style timeline.

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

1. Create timeline over bounded journal with repository, type, target, status, duration, timestamps, old/new HEAD and retry/undo availability.
2. Filter by repository, operation type and outcome; support grouped multi-repo display.
3. Enter opens sanitized details including argv display with secrets removed.
4. Allow cancel running operation, retry replayable operation after fresh preflight, and undo only where Task 146 policy allows.
5. Distinguish background read/network work from history mutations visually/textually.
6. Timeline rendering must never synchronously query Git.

## Verification

- High-volume virtualization, secret-redaction snapshots, multi-repo interleaving.

## Acceptance criteria

- [ ] User can answer “what just happened?” without opening logs.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
