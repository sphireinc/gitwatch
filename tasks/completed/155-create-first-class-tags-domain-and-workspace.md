# Task 155: Create first-class tags domain and workspace

**Phase:** Tags and remotes
**Depends on:** 125

## Goal

Stop treating tags as incidental history refs and expose them as managed objects.

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

1. Create `internal/tags` with name, target object/commit, annotated/lightweight type, tagger/date/message, signature state and remote presence summary.
2. Load via machine-readable ref iteration with pagination/bounds.
3. Create Tags workspace with filter/sort, commit inspection, compare, worktree creation and detached checkout confirmation.
4. Do not eagerly verify every signature in repositories with thousands of tags; verify on selection/bounded queue.
5. Keep tag names sanitized at presentation and validated before mutation.

## Git/process boundary

- `git for-each-ref refs/tags ...`
- `git cat-file / git verify-tag on demand`

## Verification

- Lightweight/annotated/signed tags, thousands-of-tags fixture.

## Acceptance criteria

- [x] Tags are independently browseable/manageable.

## Completion record

- [x] Implementation commits recorded: `dbe7ad0`, `1ac0a62`, `67b9c00`,
  and the final action slice commit below.
- [x] Exact tested revision recorded below.
- [x] Focused unit/integration tests recorded below.
- [x] `go test ./...` recorded below.
- [x] Race/vet/lint/format evidence recorded below.
- [x] Native/manual evidence exception documented below.
- [x] Known limitations/deferred work documented below.

## Progress evidence

- Added `internal/tags` with bounded NUL-delimited `for-each-ref` loading,
  lightweight/annotated metadata, tagger/date/message fields, deferred
  signature state, matching remote-tag presence, bounds, malformed-record
  handling, and a 2,000-tag fixture.
- Added a first-class Tags workspace with asynchronous loading, selection,
  name/target/date sorting, sanitized filtering, palette navigation, commit
  inspection, tag-to-HEAD comparison, on-demand annotated-tag verification,
  and explicit detached-checkout confirmation.
- Added selected-tag linked-worktree creation through
  `worktrees.AddWithCommit`; successful mutations request an authoritative
  refresh. App tests cover loading, rendering, filtering, sorting, inspection,
  comparison, verification, checkout, and worktree routing.
- Final tested revision: `1b4daf1`.
- Validation: full `make check` passed, including formatting, lint, full tests,
  race tests, vet, diff checks, security, and performance benchmarks.
- Native/manual evidence exception: automated routing and rendering coverage
  passed, but a human terminal pass at 80x24 remains release-QA follow-up.
- Deferred to Task 156: tag create/sign/verify/push/delete mutation lifecycle.
