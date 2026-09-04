# Task 167: Add lightweight custom commands with safe argv templates

**Phase:** Extensibility
**Depends on:** 125

## Goal

Match LZ’s low-friction custom-command usefulness without requiring a compiled plugin for every small workflow.

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

1. Create `internal/customcmd` config model: name, contexts, optional binding, palette label, executable, argv token list, working-directory mode, timeout, confirmation policy and refresh policy.
2. Define typed placeholders for repository root, selected path, selected SHA, branch, remote, tag and provider URL. Each expands to one argv element unless explicitly list-valued.
3. Reject commands when required context values are unavailable.
4. Do not invoke a shell by default. Any future shell mode must be separately named/unsafe, explicit and disabled by default.
5. Execute through operation engine with bounded stdout/stderr, cancellation and sanitized output.
6. Redact configured secret environment values from details/journal.
7. After commands marked `mutates_repository`, force authoritative refresh; read-only commands may opt out.

## Verification

- Placeholder injection stays one argv element, missing context rejection, timeout/cancel/output limits.

## Acceptance criteria

- [ ] Small custom integrations require config, not plugin compilation.
- [ ] Default custom commands cannot become shell-injection primitives.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
