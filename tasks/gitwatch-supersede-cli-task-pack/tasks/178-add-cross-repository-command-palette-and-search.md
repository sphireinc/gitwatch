# Task 178: Add cross-repository command palette and search

**Phase:** Multi-repository differentiation
**Depends on:** 125, 162, 170, 175

## Goal

Make Ctrl-P a workspace-wide navigator across repositories and already-loaded Git/provider state.

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

1. Index registered repositories, branches, bounded recent commits, current changed files, cached PRs and commands.
2. Do not crawl entire worktrees for filename search in this task.
3. Use category filters/prefixes and fuzzy scoring; favorites/attention may boost but exact matches must remain predictable.
4. Results carry repository identity. Selection switches repository using generation-safe cancellation then opens target workspace.
5. Keep index bounded and update incrementally from existing snapshots/caches.
6. Remove stale results when a repository disappears or generation invalidates them.

## Verification

- 50 repos/thousands entries latency budget, missing repo result, provider disabled.

## Acceptance criteria

- [x] Cross-repo navigation is fast and does not spawn unbounded Git commands.

## Completion record

- [x] Implementation commit recorded (`14ed506`).
- [x] Exact tested revision recorded (`9b1cb342fffba50675ce83dfc957ac0cabae1ea9`, Darwin arm64).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [x] Owner-approved operator acceptance recorded by source equivalence; no fresh native transcript is claimed.
- [x] Known limitations/deferred work documented.

## Progress evidence (2026-09-22)

- Extended the existing bounded command palette index with one repository-target action per already-loaded repository row. Labels include sanitized repository name and path, so fuzzy search can locate registered repositories without crawling worktrees or spawning Git commands.
- Selecting a healthy repository target starts the existing generation-safe repository discovery/open command; attention targets retain their repositories-workspace jump behavior.
- Added app coverage for indexing all loaded repositories and routing a selected target through repository opening.
- Added typed action categories and pure `repo:`/`repository:`/`category:` palette prefixes, allowing repository-only search without changing the bounded action index or spawning Git commands.
- Added bounded actions for up to 200 already-loaded branches, history commits, and changed files, with selection-preserving routes into their existing workspaces and focused app coverage.
- Added bounded provider targets for loaded pull requests, issues, and releases plus loaded plugin targets, with provider/plugin categories and existing workspace routing; no provider fetch is started by palette indexing or selection.
- Added centralized palette reindexing after repository, provider, and plugin ready messages so removed loaded rows cannot remain selectable while the palette is open; app coverage verifies stale repository targets are removed.
- Added repository-generation tagging and stale-result rejection for asynchronous GitHub loads so late provider responses cannot repopulate the current repository's palette or GitHub view.
- Added a deterministic 50-repository palette benchmark and allocation regression test; the performance gate now executes the benchmark and maintainer documentation records its bounded in-memory scope and 2,500-allocation threshold.
- Remaining: recorded cross-platform latency samples and native/manual acceptance.

- Current macOS sample: `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache go test ./internal/app -run '^$' -bench '^BenchmarkCommandPalette50Repositories$' -benchtime=3s` on Darwin arm64 / Apple M1 Pro measured `34,416 ns/op`, `40,122 B/op`, and `406 allocs/op`. Linux/Windows latency samples and native/manual acceptance remain open.
- Current-main revalidation: full `make check` passed at implementation commit
  `9b1cb342fffba50675ce83dfc957ac0cabae1ea9` on Darwin arm64 / Go 1.27.0.
  Dedicated cross-platform latency samples and native/manual acceptance remain
  open; benchmark coverage is not represented as operator evidence.

## Owner-approved source-equivalence acceptance (2026-09-27)

The owner approved carrying the all-platform disposition for beta candidate
`5b2a8e9` to current main `6e05247` for Task 178. The `Command palette and
notifications` matrix row was owner-signed green on macOS and Windows, with
Linux accepted by the stated macOS-equivalence decision. The palette's loaded
repository indexing, query/search, and selection routing are unchanged from
the candidate. The later plugin-load in-flight guard may affect when loaded
plugin entries become available; the owner explicitly accepted this carry.
This is owner-approved acceptance, not fresh native transcripts or physical
Linux runs. Cross-platform latency samples remain open.
