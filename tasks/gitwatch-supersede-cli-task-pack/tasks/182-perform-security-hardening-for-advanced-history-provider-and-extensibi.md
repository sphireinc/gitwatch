# Task 182: Perform security hardening for advanced history provider and extensibility

**Phase:** Hardening
**Depends on:** 132, 143, 169, 173, 181

## Goal

Threat-model and harden the larger attack surface before release.

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

1. Update threat model for malicious refs/paths/messages, crafted diffs/conflict markers, `.gitmodules`, remote URLs, provider text, plugin/custom-command output, sequence-editor temp files and external-tool templates.
2. Add architecture/static tests preventing shell command-string execution outside an explicitly reviewed unsafe-shell capability if one ever exists.
3. Fuzz rebase todo, conflict index, reflog, blame, tag, submodule and custom-command parsers.
4. Validate temp-file permissions/cleanup and operation-ID binding for rebase/historical materialization.
5. Ensure provider/plugin/custom-command secrets never enter operation journal/crash diagnostics.
6. Review TOCTOU protection for conflict edits and historical patch editing.
7. Run dependency/vulnerability tooling required by project policy.

## Verification

- Fuzz corpora, terminal-escape regressions, AST/grep execution-boundary checks, race detector.

## Acceptance criteria

- [ ] No feature weakens argv-only default execution or terminal sanitization.
- [ ] Security evidence is recorded with exact tested revision.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
