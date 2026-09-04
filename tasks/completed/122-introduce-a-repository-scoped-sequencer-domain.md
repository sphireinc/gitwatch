# Task 122: Introduce a repository-scoped sequencer domain

**Phase:** Foundation
**Depends on:** 121

## Goal

Create one domain model for multi-step Git operations so rebase, cherry-pick, revert, merge, and bisect do not each invent incompatible state machines.

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

1. Create `internal/sequencer` with immutable types for operation Kind, Phase, RepositoryID, HeadBefore, HeadCurrent, Target, CurrentCommit, Remaining, Completed, ConflictPaths, timestamps, and recovery metadata.
2. Represent at minimum rebase, cherry-pick, revert, merge, and bisect. Use typed operation-specific substructures instead of `map[string]any`.
3. Define transition rules explicitly and reject invalid transitions. UI code must never directly mutate sequencer state.
4. Keep `internal/sequencer` separate from `internal/operations`: operations schedules/cancels work; sequencer describes Git state that may outlive one child process.
5. Carry repository identity and refresh generation through every sequencer message so late results cannot land in a newly selected repository.
6. Expose read-only interfaces to UI packages. Views must not inspect `.git` paths or invoke Git directly.
7. Add package docs explaining that Git metadata/status is authoritative and sequencer objects are reconstructed projections.

## Verification

- Table tests for valid/invalid transitions.
- Race tests with interleaved messages for two repository IDs.
- No serialization becomes authoritative over Git; if persistence is added later it is cache/journal only.

## Acceptance criteria

- [x] All planned multi-step operations fit one repository-scoped state model.
- [x] `internal/operations` remains the execution coordinator.
- [x] No UI package owns a parallel Git operation state machine.

## Status

Complete — added the immutable, repository-scoped `internal/sequencer` state
projection, typed details for rebase/cherry-pick/revert/merge/bisect, explicit
lifecycle transitions, and refresh-generation message validation. Git and UI
integration remain intentionally deferred to Tasks 123 onward.

## Completion record

- [ ] Implementation commit recorded; to be filled with the focused commit
  after the task is moved.
- [x] Exact tested baseline recorded: current `main` at `ba62811` plus the
  Task 122 working-tree changes.
- [x] Focused tests: `go test ./internal/sequencer` and
  `go test -race ./internal/sequencer` passed, including interleaved
  repository-generation message coverage.
- [x] `GOCACHE=/tmp/gitwatch-go-cache go test ./...` passed.
- [x] `GOCACHE=/tmp/gitwatch-go-cache go test -race ./...` passed.
- [x] `GOCACHE=/tmp/gitwatch-go-cache go vet ./...` passed.
- [x] `gofmt` and `git diff --check` passed.
- [ ] Lint: pinned `make check` lint could not download golangci-lint v2.12.0
  because `proxy.golang.org` is unavailable; the installed v2.11.3 binary is
  not a valid substitute and failed module loading.
- [x] Native/manual evidence not applicable: this task adds a pure domain
  package and does not change terminal interaction.
- [x] Known limitations/deferred work documented: Git metadata detection,
  operation execution, UI messages, and persistence/recovery integration are
  deferred to Tasks 123 onward; state remains a reconstructed projection, not
  an authoritative store.
