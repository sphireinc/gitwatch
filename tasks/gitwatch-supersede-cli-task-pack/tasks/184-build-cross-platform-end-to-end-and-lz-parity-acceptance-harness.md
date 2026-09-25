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

## Progress evidence

- `internal/integration/TestAdvancedHistoryAndComparisonParityScenario` now
  creates a disposable repository with real Git commits, lightweight and
  annotated tags, and verifies the bounded tags, reflog, blame, path-history,
  and arbitrary-revision comparison boundaries against that repository.
- `PARITY_MATRIX.md` names this executable scenario for the corresponding
  reflog, tags, revision-comparison, and file-history/blame claims.
- `internal/integration/TestBisectParityScenarioSurvivesFreshLoaderAndReset`
  now verifies real good/bad/skip/reset transitions and reconstructs active
  state through a fresh Git runner.
- `internal/integration/TestCherryPickConflictResumeParityScenario` now creates
  a real conflicting branch, verifies the typed engine exposes a paused
  sequencer state, resolves the file, continues through Git, and checks the
  authoritative result.
- `scripts/parity-check.sh` provides a repeatable local gate for the real
  integration scenarios, multi-repository refresh, fsnotify watcher events,
  polling fallback, bisect, cherry-pick, and submodule package coverage.
- `internal/integration/TestRebaseConflictResumeParityScenario` now creates a
  real conflicting topic rebase, verifies Git's durable rebase operation and
  authoritative conflict snapshot, resolves the path, continues, and verifies
  the completed clean state.
- `internal/integration/TestRevertConflictResumeParityScenario` now exercises
  a real typed multi-step revert conflict, shared operation-state detection,
  lifecycle continuation, and the final authoritative clean snapshot.
- The parity gate also runs the existing custom-command argv/output/cancellation
  tests and plugin manifest, handshake, capability-negotiation, and compatibility
  fixtures from `internal/customcmd`, `internal/plugins`, and `pkg/plugin`.
- The complete parity harness remains open: sequencer conflict/resume lanes,
  submodules, remotes, custom commands, provider stubs, watcher and
  polling lanes, Windows path/CRLF behavior, PTY flows, and native evidence
  still need executable coverage and release-gate results.

## Progress evidence (2026-09-22)

- Added `TestMergeConflictResumeParityScenario`, which creates a real
  conflicting branch merge, verifies the typed merge engine returns a paused
  repository-scoped operation with authoritative conflicts, resolves and stages
  the path, continues through the typed lifecycle boundary, and verifies the
  clean post-merge snapshot.
- Added the merge scenario to `scripts/parity-check.sh`; the parity harness now
  covers executable conflict/resume lanes for cherry-pick, merge, rebase, and
  revert plus fresh-loader bisect recovery.
- Task 184 remains open for the remaining provider-stub depth, watcher event
  storm, Windows path/CRLF, PTY, native, and full parity-matrix evidence.

- Extended `scripts/parity-check.sh` to execute the existing real remote
  lifecycle, provider-stub, and multi-repository package suites alongside the
  disposable-repository and watcher lanes. This makes those package-level
  claims part of the repeatable parity gate rather than relying on the broader
  default test command.

- On revision `638f217`, the disposable integration and watcher lanes passed,
  followed by the bisect/cherry-pick/submodule, remotes/provider/multirepo, and
  custom-command/plugin protocol lanes from `scripts/parity-check.sh`. The
  latest pushed CI workflow also has green quality, Ubuntu, macOS, and
  full-history secret-scan jobs; the Windows job remains the final in-flight
  cross-platform result. Native terminal, PTY, Windows path/CRLF, and full
  parity-matrix evidence remain open.

- On revision `2597122`, the full local `go test ./...` gate passed. GitHub CI
  run `35759468178` then passed the quality/policy, full-history secret scan,
  Ubuntu 24.04, macOS 15, and Windows 2025 matrix jobs, including the Windows
  runtime smoke check. This supersedes the earlier in-flight Windows result;
  native PTY/operator evidence and the remaining parity lanes are still open.

- At revision `1e1edc9`, `GOCACHE=/tmp/gitwatch-parity-cache
  ./scripts/parity-check.sh` passed the disposable integration, watcher,
  bisect, cherry-pick, submodule, remote, provider, multi-repository,
  custom-command, plugin, and app lanes. The parity task remains open for
  Windows path/CRLF, PTY, and native operator evidence.

- Revision `50a36b2` adds the real 50-repository batch-fetch lane to
  `scripts/parity-check.sh`; it creates local bare remotes, exercises mixed
  clean/dirty/local-only repositories, injects missing-remotes, and verifies
  bounded cancellation/failure isolation. CI run `35777279686` passed the
  cross-platform matrix. Windows path/CRLF, PTY, and native operator evidence
  remain open.

- Added `internal/watch/TestWatcherCoalescesFilesystemEventStorm`, which writes
  256 files through the real fsnotify watcher, drains the event stream, rejects
  errors/non-filesystem events, and bounds the settled burst to 32 notifications.
  It passed 20 normal repetitions, 5 race repetitions, and vet on macOS.
- The remaining parity gaps are Windows-specific path/process acceptance, hosted
  PTY coverage, and native operator evidence.

- At revision `142ec7d`, `GOCACHE=/tmp/gitwatch-parity-current-cache
  ./scripts/parity-check.sh` passed the full disposable integration, watcher
  event-storm, bisect, cherry-pick, submodule, remote, provider,
  multi-repository, custom-command, plugin, and app lanes.

- Revision `28d34b5` fixes metadata watcher reattachment after an external Git
  metadata directory replacement by removing stale native registrations before
  rebuilding the watch tree. The load-tolerant event deadline and the
  re-created-directory scenario pass 20 normal and 5 race repetitions.
  `make check` then passed formatting, pinned lint, full tests, race tests, vet,
  security, performance, and diff checks on macOS arm64. Hosted Windows/Linux
  and native operator evidence remain open.

- Revision `29eb743` adds an opt-in large-status PTY lane to the parity harness;
  it runs the real binary against a disposable 14,953-file repository and
  asserts the authoritative status count before the existing 80x24 help and
  clean-shutdown checks. The macOS arm64 lane passed; hosted Windows/Linux
  PTY and broader native operator evidence remain open.

- Revision `64e3788` wires the ordinary and large-status PTY lanes into the
  hosted Unix CI matrix after the existing runtime smoke. Ubuntu and macOS
  jobs now require tmux startup, authoritative 14,953-file status rendering
  at 80x24, help rendering, and clean shutdown; Windows continues through its
  dedicated path parity lane.

- Added `TestPathAndCRLFStatusScenarioPreservesGitBytesAndNames` to the parity
  gate. It uses real Git with `core.autocrlf=false`, preserves CRLF file bytes,
  and verifies status identity for a unicode/space path and a leading-hyphen
  path after modification and untracked creation. The scenario passes in
  normal and race modes; native terminal/PTy evidence and broader Windows
  process/path acceptance remain open.

- Added `scripts/pty-smoke.sh`, a bounded tmux-backed smoke flow that starts the
  real binary, verifies workspace rendering, resizes to 80x24, opens help,
  captures the pane, and requires a clean `q` shutdown. `scripts/parity-check.sh`
  exposes the same lane with `GITWATCH_PTY=1`, `GITWATCH_PTY_BINARY`, and
  `GITWATCH_PTY_REPOSITORY`. Local macOS evidence passed with an isolated
  disposable fixture; hosted native Windows and broader operator evidence
  remain open.
- At revision `6237c4a`, the real binary passed the macOS arm64 PTY smoke with
  an isolated disposable repository: startup rendered, an 80x24 resize opened
  help successfully, the pane capture was written, and `q` exited cleanly.
  Hosted Windows/Linux PTY and broader native operator evidence remain open.

- At revision `1423b31`, the local parity harness passed with the new history
  graph PTY lane enabled: disposable integration, watcher, bisect,
  cherry-pick, submodule, provider, multi-repository, plugin, app, and
  history-graph terminal lanes all passed. Hosted execution remains pending
  because the workflow-bearing commits have not yet been accepted by GitHub.

- At revision `d9f2fe2`, the elevated Darwin arm64 PTY acceptance run passed
  against the real built binary: standard startup, 80x24 resize, help view,
  clean `q` shutdown, and the 14,953-file large-status lane. A bounded,
  redacted native fixture capture was also produced. Linux, Windows, and
  broader operator evidence remain open.

- At revision `df1472b`, the repeatable parity harness passed the real
  50-repository batch lane, watcher event/polling lanes, all sequencer
  conflict/resume scenarios, bisect, submodules, remotes, provider,
  multi-repository, custom-command, plugin, and app lanes. Hosted
  cross-platform PTY and native operator evidence remain open.
- At revision `44307bf`, the real macOS arm64 binary passed the bounded
  `scripts/pty-smoke.sh` run against a disposable fixture: startup rendered,
  the terminal resized to 80x24, help opened, and `q` exited cleanly. This is
  baseline native PTY evidence only; feature-specific custom-command,
  conflict-workspace, Linux, and Windows operator evidence remain open.
- At revision `02735da`, isolated-cache cross-builds passed for both
  `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=linux GOARCH=amd64 go build ./...`. These are compile-only checks and
  do not replace hosted Windows/Linux tests or native PTY acceptance.

## Progress evidence (2026-09-25)

- Hosted Windows-2025 run `36092884046` timed out in
  `TestRepositoryBatchCancellationIsReportedAsCancelled`: after the batch
  result, `httptest.Server.Close` still observed an active TCP connection and
  its handler remained blocked on `request.Context().Done()`. Linux/macOS and
  policy jobs passed on that run. This is a cross-platform cancellation defect,
  not a failure in the newly added operation-journal test.
- Windows command cancellation now attempts a bounded
  `SystemRoot\System32\taskkill.exe /PID <pid> /T /F` to terminate Git's
  process tree (including remote helpers), then falls back to killing the
  direct process. The Windows app test binary cross-compiles with Go
  `go1.27.0 darwin/arm64`; the focused cancellation integration test passes on
  Darwin and `make check` passes on that host. A hosted Windows rerun is still
  required to confirm the fix; native Windows and remaining parity evidence
  remain open.
