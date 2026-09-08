# Task 156: Implement tag create sign verify push and delete

**Phase:** Tags and remotes
**Depends on:** 155

## Goal

Match and exceed LZ tag management with signing visibility and explicit remote deletion.

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

1. Create lightweight, annotated and signed tag flows for HEAD or selected historical commit.
2. Reuse existing signing capability/config; never guess a signing key.
3. Verify selected signed tag on demand and show signer/status/error.
4. Push selected tag to selected remote; push-all-tags requires separate explicit preview.
5. Local deletion requires exact-name confirmation; remote deletion is a separate action naming remote and tag.
6. Refresh tags/history/remotes after mutations.

## Git/process boundary

- `git tag <name> <sha>`
- `git tag -a <name> <sha> -m <message>`
- `git tag -s <name> <sha> -m <message>`
- `git verify-tag <name>`
- `git push <remote> refs/tags/<name>`
- `git tag -d <name>`
- `git push <remote> :refs/tags/<name>`

## Verification

- Signing success/failure/no key, remote push/delete with local bare remote.

## Acceptance criteria

- [x] Signed tags are first-class.
- [x] Remote deletion cannot be triggered accidentally.

## Completion record

- [x] Implementation commit recorded.
- [x] Exact tested revision recorded.
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [x] Native/manual evidence recorded where this task changes terminal interaction.
- [x] Known limitations/deferred work documented.

## Progress evidence

- Added typed local tag creation for lightweight, annotated, and signed tags;
  signed creation delegates to Git's configured signing capability and never
  guesses a key.
- Added exact-name-confirmed local tag deletion with unsafe target/message
  validation and focused argv tests. Existing explicit tag push support remains
  in `internal/remotes`.
- Added a separate typed remote-tag deletion operation using the destructive
  `:refs/tags/<name>` refspec, with explicit remote/tag validation and a local
  bare-remote push/delete integration test.
- Added Tags workspace controls for lightweight, annotated, and signed local
  creation, plus exact-name local deletion confirmation. Successful local tag
  mutations request authoritative status, tags, history-tag, and remote
  refreshes.
- Added a Remotes workspace action for remote-tag deletion. It requires a
  selected remote, an explicit tag name, and a separate destructive `y/n`
  confirmation before issuing the delete refspec.
- Added an isolated empty-`GNUPGHOME` integration fixture proving signed-tag
  creation fails safely when no signing key is available. Signed creation
  dispatch and explicit `git tag -s` argv are also covered.

## Completion evidence

- Implementation commits: `eaa0447`, `847f272`, `c5ba9f6`.
- Exact tested revision: `c5ba9f6`.
- Focused coverage: `go test ./internal/tags ./internal/remotes`, Tags and
  Remotes workspace mutation tests, bare-remote tag push/delete integration,
  and no-signing-key signed-tag integration.
- Full gate at `c5ba9f6`: `make check` passed, including lint, `go test ./...`,
  `go test -race ./...`, `go vet ./...`, formatting, diff, security, and
  performance checks.
- Native/manual exception: automated Bubble Tea routing tests cover the new
  controls and refresh commands, but a human 80x24 terminal pass with a real
  signing key and remote credentials remains release QA follow-up.
- Known limitation: signed-tag cryptographic success and signer identity are
  delegated to the user's configured Git/GPG signing capability; gitwatch
  never selects or invents a signing key.
