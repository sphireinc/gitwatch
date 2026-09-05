# Task 149: Build bisect workspace

**Phase:** Bisect
**Depends on:** 148

## Goal

Make manual bisect visually obvious while live worktree status remains available for running tests.

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

1. Show known-good boundary, known-bad boundary, current candidate SHA/subject, progress/remaining estimate and bisect log.
2. Provide Good, Bad, Skip and Reset actions with deliberate reset confirmation.
3. Allow opening candidate commit details/diff and current live Status.
4. Keep watcher-driven file changes visible while user runs/tests candidate.
5. Surface externally changed checkout/candidate automatically.

## Verification

- 80x24/wide layouts, keyboard/mouse parity, NO_COLOR.

## Acceptance criteria

- [ ] User can execute a manual bisect loop entirely inside gitwatch.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Progress evidence

- Added a repository-scoped Bisect workspace with good/bad/candidate fields
  and bounded bisect-log rendering, available from the palette and Status `C`
  recovery routing.
- Added keyboard actions for Good, Bad, Skip, refresh, Status navigation, and
  a deliberate Reset confirmation. Mutations run through the operation engine
  and apply authoritative post-command snapshots.
- Added a deliberate `S` start flow that collects an explicit bad ref and good
  ref in sequence, confirms the pair, and starts bisect through the typed
  engine and operation engine. App coverage verifies the full prompt flow.
- The workspace remains usable while Git status/watch refreshes continue to
  update the active repository model.
- Starting a new bisect from explicit history refs, candidate inspection/diff,
  mouse parity, and native/manual 80x24/NO_COLOR evidence remain outstanding.
- The workspace-control slice is committed at `01fef5e`; the full `make check`
  gate passed at that revision. Explicit workspace start support is committed
  at `c7b7527`; the full `make check` gate passed there as well. Task 149
  remains active until inspection/parity and terminal acceptance work is
  complete.
