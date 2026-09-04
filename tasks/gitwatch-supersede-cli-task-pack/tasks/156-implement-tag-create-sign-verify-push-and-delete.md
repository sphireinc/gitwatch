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

- [ ] Signed tags are first-class.
- [ ] Remote deletion cannot be triggered accidentally.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
