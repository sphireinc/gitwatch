# Opinionated architecture extension

This pack assumes the current architecture remains intact. New advanced Git behavior should extend it as follows.

## Existing core remains primary

```text
filesystem events / polling
        ↓
refresh coordinator
        ↓
typed Git status command
        ↓
immutable repository snapshot
        ↓
Bubble Tea messages/state
        ↓
render
```

Nothing in this pack replaces that flow.

## New domains

### `internal/sequencer`

Common durable operation state for rebase/cherry-pick/revert/merge/bisect. It does not execute commands. It describes what Git says is happening.

### `internal/rebase`

Interactive rebase plan parsing, validation and controlled sequence-editor bridge. Rebase lifecycle state still flows through `internal/sequencer`.

### `internal/conflicts`

Unmerged-index stage model, conflict content loading and guarded resolution editing. One conflict subsystem serves merge/rebase/cherry-pick/revert.

### `internal/reflog`

Bounded recovery-point loader. Used by Reflog workspace and semantic undo safety analysis.

### `internal/bisect`

Bisect-specific state/actions layered on sequencer/operation infrastructure.

### `internal/submodules`

Submodule configuration/status/lifecycle with explicit parent-child repository relationships. It must not recursively spawn unbounded watchers.

### `internal/tags`

Tag loading/signature/lifecycle.

### `internal/compare`

Read-only arbitrary revision comparison. Reused by remote divergence, reflog recovery and provider/navigation flows.

### `internal/blame`

Bounded blame parser/model.

### `internal/customcmd`

Safe argv-based lightweight user commands/forms. This is intentionally simpler than plugins.

### `internal/health`

Derived repository-health/attention model. Local live metrics come from snapshots; external metrics carry freshness timestamps.

## Execution vs durable state

Do not merge these concepts:

```text
internal/operations = “what process/job is running?”
internal/sequencer = “what multi-step Git state is this repository in?”
internal/repo snapshot = “what is the authoritative worktree/index/branch state?”
operation journal = “what did gitwatch attempt/complete?”
```

They are related but not interchangeable.

## Refresh priority

Use scheduling priority conceptually as:

1. active repository local status/sequencer refresh;
2. user-requested local mutation completion;
3. active repository history/diff detail;
4. visible multi-repo summaries;
5. user-requested network/provider work;
6. background auto-fetch/provider refresh;
7. plugins/custom read-only enrichment.

Do not let background GitHub checks or 50-repo auto-fetch make an external file edit feel delayed.

## Suggested UI workspaces

Keep current workspace model and add only as needed:

```text
Status
Commit
Stashes
Branches
History / Graph
Rebase
Conflicts
Reflog / Recovery
Bisect
Tags
Remote Branches
Remotes
Submodules
Worktrees
GitHub
Plugins
Repositories
Operations
```

Some may be context panes rather than permanent top-level keys. Avoid exhausting single-letter global bindings; rely on command palette and context actions.

## Safety boundary for historical rewrites

All history rewrite features should share a common preflight:

- current branch/HEAD;
- active sequencer check;
- worktree/index state;
- range/base resolution;
- remote reachability/published-history warning;
- exact operation preview;
- confirmation if destructive/rewrite;
- original HEAD captured for recovery;
- operation journal entry;
- post-command authoritative refresh.
