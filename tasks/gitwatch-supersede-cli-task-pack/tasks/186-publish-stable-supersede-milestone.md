# Task 186: Publish stable supersede milestone

**Phase:** Release
**Depends on:** 185

## Goal

Release only when gitwatch can credibly replace LZ for targeted workflows while remaining distinctly better at live status, multi-repo operation, observability and safety.

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

1. Freeze final parity/capability matrix and release notes.
2. Update README positioning around four pillars: live authoritative filesystem-driven status, advanced Git workbench, multi-repo operations/health, and optional provider/plugins.
3. Document remaining non-goals honestly.
4. Verify signed tag, checksums, SBOM/provenance, cross-platform archives, install/upgrade migration, config compatibility and rollback guidance.
5. Produce deterministic demos showing external edit live refresh, interactive rebase, conflict resolution, multi-repo batch fetch/health and operation timeline/undo.
6. Do not use “supersedes LZ” language unless every required parity gate is satisfied.
7. After release, keep filesystem watching and multi-repo support as product invariants for all future features.

## Verification

- Fresh install and v2-config upgrade, final make/check/race/security/performance/release artifact verification.

## Acceptance criteria

- [ ] Stable artifacts meet project release policy.
- [ ] Filesystem watcher remains default/core behavior.
- [ ] Multi-repo is first-class in docs and implementation.
- [ ] All required parity and differentiation gates are green.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
