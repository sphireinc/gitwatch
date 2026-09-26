# Task 150: Add optional automated bisect-run commands safely

**Phase:** Bisect
**Depends on:** 148, 167

## Goal

Support automated bisect testing without violating argv-only execution.

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

1. Represent test command as executable + argv array.
2. If unsafe shell mode ever exists, require explicit `unsafe_shell` capability/config and keep it disabled by default.
3. Run through operation engine with timeout, cancellation and bounded output.
4. Stream bounded sanitized output to bisect workspace.
5. After completion refresh and show first-bad result.
6. Do not leak environment secrets into journal/logs.

## Git/process boundary

- `git bisect run <cmd> <args...>`

## Verification

- Passing/failing sequence, timeout/cancel, executable path with spaces.

## Acceptance criteria

- [x] Automated bisect is bounded/cancellable.
- [x] Default execution never uses shell interpolation.

## Completion record

- [x] Implementation commits recorded (`f0b0509`, `62e2de1`, `64fe6d4`, `5f40631`, and `abebf5c`).
- [x] Exact tested revision recorded (`e3bdedb`; current-main hosted verification is recorded below).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded through `make check` at `e3bdedb`.
- [x] Race/vet/lint/format evidence recorded through `make check` at `e3bdedb`.
- [x] Owner-approved operator acceptance recorded by source equivalence; no fresh native transcript is claimed.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added `bisect.RunCommand` with an executable plus already-tokenized argv,
  bounded output, context cancellation support, and authoritative post-run
  snapshot/state refresh. No shell command string is accepted or built.
- Real repository coverage exercises `git bisect run` with an executable path
  containing spaces and an argument containing spaces; validation rejects
  newline/NUL/option-like executable tokens.
- Workspace launch controls now collect the executable and each argv token
  separately, require confirmation, run through the operation engine, and
  render bounded sanitized stdout/stderr in the bisect workspace. Automated
  app coverage exercises executable and argument paths containing spaces and
  verifies the operation enters the pending state.
- Native/manual terminal evidence remains outstanding; the output is captured
  after the bounded operation rather than streamed incrementally.
- The bounded argv-only run slice is committed at `f0b0509`; the full
  `make check` gate passed at that revision. The workspace slice is committed
  at `62e2de1`, with the same full gate passing at that revision. Task 150
  remains active until the
  dependent custom-command foundation, terminal acceptance, and remaining
  cancellation/timeout evidence are complete.

## Progress evidence (2026-09-22)

- Added `TestRunCommandPropagatesCancelledContext`, which starts a real
  disposable bisect, invokes the typed run boundary with an already-cancelled
  context, and asserts the Git cancellation sentinel is preserved. This keeps
  cancellation observable without introducing shell-based test helpers.
- Task 150 remains open for in-flight process timeout evidence, incremental
  output streaming, native terminal acceptance, and the dependent custom
  command foundation.

- Revision `64fe6d4` adds an argv-only `git.RunBoundedStreaming` boundary and
  wires `bisect.RunRequest.OnOutput` to it. Stdout and stderr are delivered as
  copied fragments while the process runs, each stream remains bounded, and
  cancellation/output-limit errors retain the existing typed Git error model.
  `TestRunnerBoundedStreamingDeliversBothStreamsAndBoundsRetention` passes
  under the race detector; the full `go test ./...` and `go vet ./...` gates
  also pass at this revision. Native terminal acceptance and in-flight
  timeout evidence remain open.

- Revision `5f40631` connects the bounded stream to the Bubble Tea bisect
  workspace through repository-generation-tagged messages. Stdout/stderr are
  sanitized, displayed incrementally, and capped at 64 KiB; dropped display
  fragments cannot affect the authoritative final `git bisect run` result.
  Focused normal and race tests cover the display sanitizer/bound and the
  existing argv collection path. Native terminal acceptance and in-flight
  timeout evidence remain open.

- Revision `abebf5c` adds `TestRunCommandCancelsInFlightProcessWithinTimeout`,
  which starts a real disposable bisect and invokes an argv-only test executable
  that remains running until the context deadline. The test passes repeatedly
  and verifies cancellation returns within a bounded time; the full bisect
  package also passes. Native terminal acceptance and dependent custom-command
  completion remain open.
- At revision `e3bdedb`, the full `make check` gate passed on macOS arm64 after
  the in-flight cancellation test: formatting, pinned lint (0 issues), normal
  and race tests, vet, security fuzz checks, diff checks, and performance
  benchmarks all completed successfully.

## Current-main documentation and gate audit (2026-09-25)

- The application accepts an executable and argument tokens separately and
  passes them as argv to `git bisect run`; it does not construct a shell
  command. The shared Git runner uses cancellable process-tree handling on
  Unix and Windows. The automated run is operation-engine scoped with a
  30-minute timeout, bounded streaming output, and sanitized 64 KiB workspace
  display.
- The later commits above close the previously noted incremental-output and
  in-flight timeout/cancellation implementation gaps; Task 167 is in
  `tasks/completed`, satisfying the custom-command foundation dependency.
- In `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache
  GOPROXY=off GOSUMDB=off go test ./internal/bisect ./internal/git
  ./internal/app -count=1`, the bisect and Git packages passed; the app package
  was blocked by the sandbox denying the test server's IPv6 localhost bind in
  unrelated `TestRepositoryBatchCancellationIsReportedAsCancelled`.
- `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache
  GOPROXY=off GOSUMDB=off go test ./internal/app -run 'Bisect' -count=1`
  passed, including automated-run argv collection and bounded output display.
- The current local `make check` retry stopped at pinned-linter module lookup
  because the sandbox cannot resolve `proxy.golang.org`; the full gate remains
  recorded as passing at `e3bdedb`, with current-main hosted CI below.
- Hosted Actions run `36179909264` for current main `eebde8f` passed quality,
  policy, secret scanning, and the Ubuntu, macOS, and Windows matrix.
- The user-approved matrix has no bisect-specific row. The later explicit
  source-equivalence approval and its no-transcript scope are recorded below.
- User documentation now describes the manual and automated bisect workflows,
  argv-only command entry, confirmation, bounded output, timeout, and cancel
  behavior in `README.md`, `KEYMAP.md`, and `docs/advanced-workflows.md`.
- The executable is intentionally user-supplied and may mutate files or access
  the network; gitwatch does not sandbox it. Output display is bounded and
  sanitized, so it is diagnostic rather than a source of operation truth.

Task 150's code and hosted gates are complete. The owner-approved operator
acceptance for this automated-run workflow and its Task 148–149 workspace
dependencies is recorded by source equivalence below.

## Owner-approved source-equivalence acceptance (2026-09-26)

The owner explicitly approved carrying the operator disposition to Tasks
148–150 by source equivalence, despite the beta matrix having no bisect-specific
row. The automated-run engine and app integration have no Go-source delta from
the signed candidate through `9b1cb34`. This records owner-approved
acceptance, not a fresh bisect transcript or physical Linux run; Linux remains
accepted by the stated macOS-equivalence decision.

Task 150 is complete under the implementation, verification, and
owner-authorized platform-acceptance evidence recorded above. Its task-specific
source is unchanged on `9b1cb34`; the separate Task 89 hosted-CI gate remains
pending for beta publication.
