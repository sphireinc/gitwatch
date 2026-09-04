# Task 151: Add submodule discovery and status domain

**Phase:** Submodules
**Depends on:** 125

## Goal

Treat submodules as nested repositories with explicit parent-child health instead of opaque dirty entries.

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

1. Create `internal/submodules` and parse `.gitmodules` through Git/config machine output where possible.
2. Load name, path, redacted URL, recorded superproject commit, checked-out commit, initialized/dirty/detached state and bounded divergence.
3. Integrate summary into parent repository health without recursively loading unlimited depth.
4. Use depth/repository limits and cycle guards.
5. Represent missing/uninitialized submodules explicitly.
6. Never allow submodule URL credentials into logs/UI.

## Git/process boundary

- `git config -z -f .gitmodules --get-regexp ...`
- `git submodule status --recursive with bounded/validated parsing`

## Verification

- Initialized/uninitialized/dirty/detached/nested submodules, paths with spaces, URL redaction.

## Acceptance criteria

- [ ] Parent repo displays submodule health without blocking core status.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
