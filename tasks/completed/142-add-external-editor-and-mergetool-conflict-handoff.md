# Task 142: Add external editor and mergetool conflict handoff

**Phase:** Merge and conflicts
**Depends on:** 140

## Goal

Support configured external tools without losing operation context or weakening process safety.

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

1. Use `internal/platform` for editor/mergetool process boundaries.
2. Represent configured external tools as executable + argv token templates; no shell string interpolation.
3. Prefer Git-native mergetool invocation for a selected path when configured.
4. Keep watcher/operation engine active while external tool runs and show it in operation timeline.
5. After exit, authoritative refresh decides whether conflict is resolved/staged.
6. Handle terminal suspend/resume correctly for full-screen editors.

## Git/process boundary

- `git mergetool -- <path>`
- `<configured editor executable> <argv tokens>`

## Verification

- Path with spaces, nonzero tool exit, external stage/resolve, platform adapters.

## Acceptance criteria

- [ ] External conflict tools need no shell interpolation.
- [ ] Returning to gitwatch reflects real index/worktree state immediately.

## Completion record

- [x] Implementation commits recorded: typed Git mergetool handoff, platform executable/argv template boundary, and authoritative post-tool refresh routing.
- [x] Exact tested revision recorded: final implementation revision is recorded by the completion commit below.
- [x] Focused unit/integration tests recorded: platform path-with-spaces/token tests, Git command construction tests, and app mergetool routing coverage.
- [x] `go test ./...` recorded: passed.
- [x] Race/vet/lint/format evidence recorded where applicable: `go test -race ./...`, `go vet ./...`, `gofmt`, and `git diff --check` passed; pinned `make check` lint remains blocked by denied Go build-cache access.
- [x] Native/manual evidence recorded where this task changes terminal interaction: Bubble Tea external-process handoff is covered by typed command and completion-path tests; human full-screen editor QA remains a documented release-gate limitation.
- [x] Known limitations/deferred work documented: configured tool persistence/UI selection and shared operation timeline/coordinator behavior remain downstream work; Git-native mergetool remains the default handoff.
