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

- [x] ROADMAP no longer says interactive rebase is not planned.
- [x] Parity and differentiation goals are separately documented.
- [x] Existing watcher/multi-repo/safety principles are unchanged.

## Status

Complete for its documentation scope — the roadmap, parity matrix, and
task-lane documentation are implemented. Task 120’s operator-owned native
release evidence remains pending as a documented dependency exception; it does
not change the planned Supersede scope or claim rule.

## Completion record

- [ ] Implementation commit recorded; repository commit writes are blocked by
  the current sandbox's inability to create `.git/index.lock`.
- [x] Exact tested baseline recorded: `main` at `71eaaff0793483ee3bdd8af6c1b8d8522362d867` when the documentation checks were run; changes remain uncommitted because the sandbox cannot create `.git/index.lock`.
- [x] Focused documentation consistency search recorded: no stale governing-doc claim that interactive rebase is unplanned or that the new parity workflows are unsupported.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go test ./...` passed.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go test -race ./...` passed.
- [x] `GIT_CONFIG_GLOBAL=/dev/null GOCACHE=/tmp/gitwatch-go-cache go vet ./...` passed.
- [x] Formatting and `git diff --check` passed.
- [ ] Lint evidence: `make check` reached lint, but the sandbox could not download `golangci-lint` from `proxy.golang.org`; the default cache path also required the temporary `GOCACHE` workaround.
- [x] Native/manual evidence not applicable: this task changes planning
  documentation only; native evidence remains required by Task 120 and later
  implementation/release tasks.
- [x] Known limitations/deferred work documented: the matrix is planning evidence only; Tasks 122–186 remain unimplemented, Task 120 native evidence remains operator-owned, and README shipping copy was intentionally unchanged.
