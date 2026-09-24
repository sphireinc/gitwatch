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

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

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
- Candidate inspection now reuses the existing history inspector from the
  bisect workspace (`i`), and mouse clicks on the candidate/action row map to
  the same inspection and Good/Bad/Skip/Reset commands as keyboard input.
- Focused app coverage verifies candidate inspection dispatch and mouse Good
  action parity.
- The workspace remains usable while Git status/watch refreshes continue to
  update the active repository model.
- Native/manual 80x24/NO_COLOR evidence and a complete operator-driven bisect
  loop remain outstanding.
- The workspace-control slice is committed at `01fef5e`; the full `make check`
  gate passed at that revision. Explicit workspace start support is committed
  at `c7b7527`; the full `make check` gate passed there as well. Task 149
  remains active until inspection/parity and terminal acceptance work is
  complete.
- Candidate inspection and mouse-parity support are committed at `bc4eca2`;
  the full `make check` gate passed at that revision. Native terminal
  evidence remains outstanding.
- Revision `e2cfe34` adds `TestBisectCandidateInspectorShowsCommitPatch`, which
  creates a real two-commit repository, opens the active candidate from the
  Bisect workspace, and verifies both commit metadata and its patch appear in
  the workspace presentation. Focused app testing and
  `GOCACHE=/tmp/gitwatch-go-cache GOMODCACHE=/tmp/gitwatch-go-mod-cache make
  check` passed on Darwin arm64; lint reported 0 issues and the full/race tests,
  vet, security, performance, formatting, and diff checks passed. GitHub
  Actions run
  [35869925536](https://github.com/sphireinc/gitwatch/actions/runs/35869925536)
  passed quality/policy, full-history secret scan, and Ubuntu, macOS, and
  Windows jobs. This closes the candidate-patch rendering gap; native/manual
  80x24/NO_COLOR and full operator-loop evidence remain open.
- Revision `07e8a29` adds the current candidate subject and Git-provided
  approximate remaining estimate to the workspace, fixes latest marked
  boundary display, and verifies an app-driven start/good/bad/reset loop in a
  disposable repository. `make check` passed on Darwin arm64 at this revision
  (lint, full/race tests, vet, format, diff, security, performance). This is
  automated integration evidence, not native/manual 80x24, NO_COLOR, mouse,
  or operator-loop acceptance; the task remains active.
- Revision `12ec973` fixes quit-key routing in the Bisect workspace: `q` and
  Ctrl-C now quit instead of being swallowed by workspace-specific dispatch,
  while literal `q` remains typeable in start-ref prompts. Regression tests
  cover both quit bindings and ref input. The interactive 80x24/`NO_COLOR=1`
  PTY run documented under Task 148 exposed the defect and exercised the
  operator loop; a locally built post-fix binary also confirmed `q` exits
  directly from the reopened active-Bisect workspace. This remains scripted
  PTY evidence rather than native/manual terminal sign-off.
- At `12ec973`, `go test ./...`, `go test -race ./...`, `go vet ./...`, format,
  diff, `scripts/security-check.sh`, and `scripts/performance-check.sh` passed
  on Darwin arm64. `make check` cannot complete its pinned lint step offline:
  fetching golangci-lint v2.12.0 fails DNS resolution for proxy.golang.org;
  the installed v2.11.3 binary is built with Go 1.26 and cannot type-check Go
  1.27 export data. Hosted CI status is pending because GitHub API reads are
  currently unreachable. Native/manual evidence remains open.
