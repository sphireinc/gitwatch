# Task 139: Load safe three-way conflict content and diffs

**Phase:** Merge and conflicts
**Depends on:** 138

## Goal

Provide base/ours/theirs/result detail with bounded binary/large-file handling.

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

1. Load blobs by object ID from the conflict model using `git cat-file`, not by trusting conflict-marker text.
2. Detect binary and oversized blobs before rendering full content; show metadata/external-tool options when over budget.
3. Build base→ours, base→theirs and current-result presentation data.
4. Reuse existing patch/diff truncation budgets.
5. Keep raw bytes only where necessary for patch/edit operations; sanitize terminal output at presentation.
6. Cache only generation-scoped selected detail and invalidate on status/index refresh.

## Git/process boundary

- `git cat-file blob <oid> or bounded git cat-file --batch`

## Verification

- UTF-8, invalid UTF-8, binary, huge files, deleted side.

## Acceptance criteria

- [ ] Conflict detail comes from index object IDs.
- [ ] Large/binary conflicts cannot freeze or corrupt terminal output.

## Completion record

- [x] Implementation commit recorded: `feat: add bounded conflict content loader`.
- [x] Exact tested revision recorded: implementation commit below.
- [x] Focused unit/integration tests recorded: `go test ./internal/git ./internal/conflicts`.
- [x] `go test ./...` recorded: passed.
- [x] Race/vet/lint/format evidence recorded where applicable: `go test -race ./...`, `go vet ./...`, `gofmt`, and `git diff --check` passed; `make check` lint remains blocked by the environment's denied Go build-cache path.
- [x] Native/manual evidence recorded where this task changes terminal interaction: not applicable; content loading is an asynchronous/data boundary with no new terminal interaction.
- [x] Known limitations/deferred work documented: content is loaded for selected conflicts only; blob contents are bounded and binary/invalid UTF-8 content is metadata-classified for later presentation; resolver UI owns presentation and edits.
