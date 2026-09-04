# Task 133: Add repository-scoped commit basket and range selection

**Phase:** Cherry-pick and history selection
**Depends on:** 125

## Goal

Create reusable commit selection for cherry-pick, compare, batch history actions, and future workflows.

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

1. Create a history selection model with ordered SHA list, source repository ID/ref, selection generation, and range semantics.
2. Support single commit, contiguous range, and explicit multi-select while preserving intended application order.
3. Never silently carry cherry-pick selection across repositories. Cross-repo patch transfer is a separate future capability.
4. Make selection survive history pagination and filtering by SHA identity.
5. Show basket count/status in History and command palette and provide an explicit clear action.
6. Normalize display order vs Git application order and show the final order before mutation.

## Verification

- Range across paginated history.
- Multi-select with filtering.
- Repository switch clears/isolates selection.

## Acceptance criteria

- [x] Selection is repository-scoped and reusable.
- [x] No selection can accidentally target the wrong repository.

## Completion record

**Status:** Complete

- Implementation commit: `781aa47` (`feat: add scoped history commit basket`). The reusable selection model is in `internal/history/selection.go`, with History integration in `internal/ui/historyview` and `internal/app`.
- Exact tested revision: `31ada9a` plus the working-tree Task 133 changes.
- Focused tests: selection and history-view tests passed for repository/ref isolation, pagination, filtering, range normalization, explicit multi-select, and clear behavior.
- Repository tests: `go test ./...` passed.
- Race/vet/format: `go test -race ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `git diff --check` passed.
- Lint: the pinned `golangci-lint@v2.12.0` remains unavailable because the environment denies Go build-cache access; no analyzer failure was reported.
- Native/manual evidence: keyboard basket/clear paths and safe History rendering are covered by automated tests. Native terminal snapshots remain an explicit manual-QA exception.
- Known limitations/deferred work: cherry-pick execution and range mutation consumers are implemented by later tasks. The basket preserves explicit application order and is cleared whenever repository or ref scope changes; generation guards reject stale history use.
