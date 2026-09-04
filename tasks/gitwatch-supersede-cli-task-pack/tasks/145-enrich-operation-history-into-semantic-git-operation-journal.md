# Task 145: Enrich operation history into semantic Git operation journal

**Phase:** Recovery and undo
**Depends on:** 122, 144

## Goal

Upgrade the existing bounded operation completion history into a user-facing record of what Git state changed.

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

1. Extend operation records with repository ID, operation kind, redacted display argv, old/new HEAD where relevant, refs changed, targets, duration and outcome.
2. Join operation completion with reflog observations where possible to identify exact recovery points.
3. Never persist credentials, URL userinfo, provider tokens, full environment or unsanitized repository text.
4. Keep the journal bounded. Persistent journal is optional and must be private/versioned if introduced.
5. Give every replayable operation stable typed intent sufficient for retry/undo checks.
6. Represent network-only operations accurately even when HEAD does not move.

## Verification

- Redaction tests, commit/rebase/merge/cherry-pick/fetch records, two-repo interleaving.

## Acceptance criteria

- [ ] Timeline explains what happened without leaking secrets.
- [ ] Journal is repository-scoped and bounded.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
