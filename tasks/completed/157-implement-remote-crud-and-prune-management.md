# Task 157: Implement remote CRUD and prune management

**Phase:** Tags and remotes
**Depends on:** 125

## Goal

Complete remote object management in addition to existing fetch/pull/push.

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

1. Add remote add, rename, set-url, remove, prune and detail flows.
2. Validate remote names and pass URL as opaque argv; never log URL credentials.
3. Before remove/rename show affected upstream/tracking branches.
4. Set-url UI shows redacted existing URL and never echoes submitted credentials afterward.
5. Prune describes scope and optionally previews stale refs where feasible.
6. After changes refresh remotes, branch upstreams, registry summaries and provider detection.

## Git/process boundary

- `git remote add`
- `git remote rename`
- `git remote set-url`
- `git remote remove`
- `git remote prune`
- `git remote get-url`

## Verification

- Credential-like URL redaction, rename upstream display, remove warning.

## Acceptance criteria

- [x] Remote lifecycle is safely manageable inside gitwatch.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [x] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added typed remote add, rename, set-url, remove, prune, dry-run prune, and
  redacted get-url operations at the Git boundary.
- Remote names reject option-like, path-like, traversal, and control-character
  inputs; URLs remain opaque argv values while control characters are rejected.
- Added focused argv, validation, and credential-redaction tests. Remotes
  workspace controls, upstream-impact warnings, and full lifecycle integration
  remain pending.
- Added a NUL-delimited tracking-branch query that identifies local branches
  whose upstream belongs to a selected remote, enabling scoped rename/remove
  warnings without parsing human-formatted Git output.
- Added staged Remotes workspace controls for add, rename, set-url, remove,
  and prune preview/confirmation. Rename and remove show affected tracking
  branches; URL input is hidden from status text, and successful remote jobs
  refresh status, remotes, branches, and provider detection.

## Completion evidence

- Implementation commits: `c695cd7`, `042498c`, `4b169eb`.
- Exact tested revision: `4bade03`.
- Focused coverage: remote lifecycle argv and validation tests, credential
  redaction tests, tracking-branch tests, Remotes workspace control tests, and
  a local bare-remote integration covering add, upstream push, rename,
  tracking preservation, prune preview, set-url redaction, and remove.
- Full gate at the tested tree: `make check` passed, including lint,
  `go test ./...`, `go test -race ./...`, `go vet ./...`, formatting, diff,
  security, and performance checks.
- Native/manual exception: automated Bubble Tea tests cover the new prompts,
  impact warnings, confirmations, hidden URL input, and refresh commands, but
  a human 80x24 terminal pass remains release QA follow-up.
- Known limitation: `git remote prune --dry-run` may contact the configured
  remote; the UI labels it as a preview and keeps it behind explicit action.
