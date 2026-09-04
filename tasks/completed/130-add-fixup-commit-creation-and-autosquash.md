# Task 130: Add fixup commit creation and autosquash

**Phase:** Interactive rebase
**Depends on:** 129

## Goal

Support the fast history-cleanup workflow: create `fixup!` commits from current staged changes and autosquash them into earlier commits.

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

1. From History add `Create fixup commit` for selected commit using normal commit preflight and existing staged content.
2. Do not auto-stage. Reuse current Status/hunk staging and commit validation.
3. Capability-probe the minimum Git version before exposing amend/reword fixup variants.
4. Show fixup relationships in history/rebase plan.
5. Add `Autosquash fixups` using explicit base selection and the controlled rebase bridge.
6. Require published-history warning/confirmation when autosquash rewrites remotely-reachable commits.

## Git/process boundary

- `git commit --fixup=<sha>`
- `git rebase -i --autosquash <base>`

## Verification

- Multiple fixups to one target.
- Fixup target outside selected base range.
- Commit signing configuration remains respected.

## Acceptance criteria

- [x] Create and autosquash fixup commits entirely inside gitwatch.
- [x] Staging stays explicit.
- [x] Remote rewrite warnings are enforced.

## Completion record

**Status:** Complete

- Implementation commit: `502152f` (`feat: add fixup commits and autosquash`). `internal/git/commit.go` now supports `git commit --fixup=<sha>`, while `internal/git/rebase.go` accepts controlled autosquash execution. History and rebaseview controls are wired through `internal/app`.
- Exact tested revision: `74cda9f` plus the working-tree Task 130 changes.
- Focused tests: Git fixup integration, rebase plan, rebaseview, and app tests passed; the fixup test verified a real `fixup!` subject and explicit staging.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: the pinned `golangci-lint@v2.12.0` remains unable to execute under the environment's Go build-cache permission boundary; no analyzer failure was reported.
- Native/manual evidence: keyboard and mouse workspace paths are covered by automated app/view tests. Native terminal snapshots and signing-provider-specific manual verification remain unavailable in this environment and are recorded as an explicit QA exception.
- Known limitations/deferred work: capability-specific amend/reword variants and broader sequencer recovery remain later-task work. Fixup creation never stages files automatically; autosquash requires an explicit base and confirmation for remote-reachable rewrites.
