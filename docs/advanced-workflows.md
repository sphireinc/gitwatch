# Advanced workflows

This document describes the advanced workbench surfaces included in the first
stable release. Commands are exposed only where the corresponding validation,
confirmation, cancellation, and authoritative refresh workflow is complete.

## History and patch work

History loading uses bounded pages and machine-readable fields for SHA, parents,
refs, signature state, author, timestamp, and subject. Commit inspection can
load parent-relative patches, changed-file statistics, and a path-filtered
view. Partial staging builds a patch from stable hunk/line identities and runs
`git apply --check` before changing the index.

## Stashes, branches, and worktrees

Stash and branch mutations validate refs/names before invoking Git. Deletion
requires exact confirmation and never permits deleting the checked-out branch.
Worktree discovery uses `git worktree list --porcelain`; lifecycle commands are
typed argv operations and branch occupancy is explicit.

## Remotes and GitHub

Remote URLs are redacted before display or diagnostics. Pull strategy must be
explicit (`merge`, `rebase`, or `ff-only`), and force pushing is represented
only by the opt-in `--force-with-lease` operation. GitHub support is optional:
remote detection, environment/GitHub CLI token sources, cached PR metadata, and
check-run parsing degrade to an unavailable provider state without blocking
core Git workflows.

## Plugins and multi-repository work

Plugins use the dependency-free `pkg/plugin` wire contract and execute out of
process with bounded output. The registry discovers explicit roots with depth
and repository limits, persists private JSON metadata, and refreshes status via
a bounded worker pool. Repository rows are filterable/sortable; favorites and
groups are stored as registry metadata.

## Configuration and safety

### Background operation lifecycle

Git, network, history, provider, and plugin work uses the shared operation
engine. Each operation has a stable ID and repository scope and transitions
through `queued`, `running`, `completed`, `failed`, `canceled`, or `timed out`.
The engine limits concurrent work, serializes conflicting work for one
repository, rejects duplicate IDs, and keeps a bounded completion history.
Cancellation propagates through the operation context to child Git/process
boundaries. Cancellation and timeout are reported separately from ordinary
failure; a successful mutation still requests an authoritative refresh.

Repository switching supplies a new context, so late results from the prior
repository cannot be applied to the active workspace.

Configuration schema version 2 can be validated with:

```sh
gitwatch --config-check
gitwatch --config-inspect
```

Git commands always use argument vectors. Terminal text is sanitized and
diagnostics redact token/password/authorization fields and URL userinfo. No
telemetry is collected by these workflows.
