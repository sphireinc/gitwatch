# Codex execution rules

This file is written directly to the implementation agent.

## Your job

Work the task files in numeric/dependency order. Do not reinterpret them as suggestions. If the repository has evolved since this pack was written, preserve the intent and current architecture rather than mechanically forcing obsolete paths.

## Before touching code for each task

1. Read `AGENTS.md`, `ARCHITECTURE.md`, `UX_SPEC.md`, `KEYMAP.md`, `ROADMAP.md`, `docs/advanced-workflows.md`, and the task file.
2. Inspect the existing package that owns the behavior. Extend it instead of creating a duplicate subsystem.
3. Identify the authoritative Git command/data source.
4. Identify repository scope, cancellation scope and refresh generation.
5. Identify whether the operation mutates worktree, index, refs, provider state, or only presentation.
6. Identify the exact post-operation authoritative refresh path.
7. Identify safety/confirmation and weird-path/terminal-injection cases before implementation.

## Implementation rules

### Domain before UI

For complex workflows use this order:

```text
Git/process adapter
→ parser/domain model
→ pure transition/validation logic
→ operation-engine integration
→ Bubble Tea messages/update
→ presentation/view
→ keymap/mouse/help
→ integration tests
```

Do not put Git command construction in UI packages.

### Reuse current infrastructure

Prefer these existing areas:

- `internal/git` — Git process boundary/parsers.
- `internal/repo` — immutable repository snapshots.
- `internal/watch` — filesystem events/debounce/reconciliation/polling.
- `internal/operations` — background execution/cancellation/per-repo serialization.
- `internal/workspace` — workspace lifecycles/navigation.
- `internal/history`, `internal/patch`, `internal/hunks` — history/diff/patch primitives.
- `internal/remotes`, `internal/provider`, `internal/plugins`, `internal/registry` — existing advanced systems.
- `internal/ui/*` — pure view/presentation.

Create new packages only when the domain is genuinely new (for example `internal/sequencer`, `internal/rebase`, `internal/conflicts`, `internal/reflog`, `internal/bisect`, `internal/submodules`, `internal/tags`, `internal/blame`, `internal/health`).

### Mutations

Every mutation must follow:

```text
validate current snapshot
→ confirm if required
→ enqueue typed operation
→ execute argv-based Git/provider/process work
→ capture result
→ request authoritative refresh
→ reconcile sequencer/registry/provider state
→ notify/update timeline
```

Do not mutate repository state and then directly patch the UI as if that were authoritative.

### Long-running Git operations

Rebase, merge, cherry-pick, revert and bisect are durable Git states. They may outlive one process invocation or one gitwatch session.

Therefore:

- derive their state from Git after every refresh;
- detect operations started outside gitwatch;
- support restart/resume;
- never store “the truth” only in Bubble Tea model fields;
- use `internal/sequencer` for common lifecycle representation.

### Multi-repository behavior

For each feature ask:

- What badge/summary appears in the repositories dashboard?
- What happens if the user switches repositories before the operation returns?
- Can another repository refresh or fetch concurrently?
- Is work bounded under 50–100 registered repos?
- What happens if one repo disappears/fails?

If these questions are unanswered, the task is not finished.

## Testing requirements

Use unit tests for pure parsers/models and **real disposable Git repositories** for Git semantics.

Mocks are not sufficient for:

- rebase;
- cherry-pick;
- merge conflicts;
- reflog/undo;
- bisect;
- submodules;
- worktree/ref behavior;
- weird path handling.

At minimum after each feature lane:

```text
gofmt / formatting gate
go test ./...
go test -race ./... (or project-scoped race gate where full suite is impractical)
go vet ./...
project lint/check target
```

Also record native/manual terminal evidence when interaction, mouse, terminal sizing or external-process handoff changes.

## What not to do

Do not:

- delete the watcher architecture because command completion already triggers refresh;
- replace multi-repo with a “later” TODO;
- add a global mutable Git state singleton;
- parse localized human Git output if machine-readable output exists;
- introduce generic hard reset/force/clean shortcuts to match another tool’s feature checkbox;
- use shell strings for convenience;
- eagerly load every diff/blob/tag signature/provider record for every repository;
- mark a task complete with TODOs in required acceptance behavior;
- advertise a feature before its acceptance evidence exists.
