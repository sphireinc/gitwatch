# Task 148: Implement bisect engine and state loader

**Phase:** Bisect
**Depends on:** 123, 125

## Goal

Support starting, resuming, marking, skipping and resetting Git bisect as a repository-scoped operation.

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

1. Create `internal/bisect` with good refs, bad refs, current candidate, log and remaining estimate where Git exposes it.
2. Start only after explicit good/bad commit selection and ref resolution.
3. Support mark good, mark bad, skip and reset.
4. Detect externally-started bisect using Task 123 and reconstruct from Git.
5. Execute marks through operation engine and refresh status/history/registry after each checkout.
6. Show exact preflight when dirty worktree affects Git behavior; do not silently stash/reset.

## Git/process boundary

- `git bisect start <bad> <good>`
- `git bisect good`
- `git bisect bad`
- `git bisect skip`
- `git bisect reset`
- `git bisect log`

## Verification

- Happy path to first bad, skip, restart mid-bisect.

## Acceptance criteria

- [ ] Bisect can be fully controlled and resumed from gitwatch.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
