# Task 184: Build cross-platform end-to-end and LZ-parity acceptance harness

**Phase:** Hardening
**Depends on:** 132, 143, 150, 154, 160, 164, 166, 169, 173, 180, 183

## Goal

Create reproducible evidence for advanced Git semantics and every parity claim.

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

1. Extend `internal/integration` with real disposable-repository scenarios for rebase, autosquash, cherry-pick, merge conflicts, reflog undo, bisect, submodules, tags, remote branches, compare, blame, custom commands, provider stubs and multi-repo batch operations.
2. Use real Git subprocesses for Git semantics; do not mock behavior that depends on Git state transitions.
3. Add Windows-specific path/CRLF/process tests and macOS/Linux terminal acceptance where CI/native evidence allows.
4. Maintain `PARITY_MATRIX.md`: LZ workflow → gitwatch task → automated test → native/manual evidence. Never mark parity without evidence.
5. Extend Expect/PTY demo/smoke flows for representative interactions, while deterministic model/integration tests remain primary CI.
6. Include fsnotify and polling fallback lanes.
7. Run race/vet/lint/format/security/performance gates.

## Verification

- Full matrix in CI, one real conflict/resume per sequencer kind, multi-repo concurrent scenario.

## Acceptance criteria

- [ ] Every parity claim maps to executable evidence.
- [ ] Watcher-first and multi-repo behavior are explicit release gates.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
