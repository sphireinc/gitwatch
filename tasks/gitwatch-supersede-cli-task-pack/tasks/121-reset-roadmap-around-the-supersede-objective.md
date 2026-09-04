# Task 121: Reset roadmap around the supersede objective

**Phase:** Foundation
**Depends on:** 120

## Goal

Change the public product direction from an advanced day-to-day Git dashboard to a terminal-native Git workspace that can replace LZ while preserving gitwatch’s watcher-first identity.

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

1. Update `ROADMAP.md` so interactive rebase, cherry-pick, merge/conflict resolution, reflog recovery, bisect, submodules, full tag/remote management, deep GitHub workflows, and multi-repo operations are explicit planned capabilities.
2. Remove the existing sentence declaring interactive rebase unplanned. Do not remove the prohibitions on telemetry, embedded Git implementations, arbitrary in-process third-party UI, generic hard reset, raw force push, or clean-fd shortcuts.
3. Add a named `Supersede` milestone and state that parity is necessary but insufficient: gitwatch must remain stronger at live status, multi-repo operation, observability, safety, and repository health.
4. Add `PARITY_MATRIX.md` mapping LZ workflow → current support → task closing the gap → executable acceptance evidence.
5. Update `tasks/README.md` with Tasks 121+ as a new execution lane and state that Task 120 remains the prerequisite release/context gate.
6. Update `README.md` positioning only after the implementation exists; do not advertise future capabilities as shipped.

## Verification

- Documentation consistency search across ROADMAP/README/ARCHITECTURE/KEYMAP/RELEASE_CRITERIA/tasks README.
- Repository search for claims that rebase/cherry-pick/submodules/etc. are unsupported or not planned.

## Acceptance criteria

- [ ] ROADMAP no longer says interactive rebase is not planned.
- [ ] Parity and differentiation goals are separately documented.
- [ ] Existing watcher/multi-repo/safety principles are unchanged.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
