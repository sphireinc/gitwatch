# Advanced workflows

This document describes the current domain contracts behind the post-v1
surfaces. Interactive panels are being integrated incrementally; commands
remain available only where the corresponding UI confirmation and refresh
workflow is complete.

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

Configuration is versioned at v2 and can be validated with:

```sh
gitwatch --config-check
gitwatch --config-inspect
```

Git commands always use argument vectors. Terminal text is sanitized and
diagnostics redact token/password/authorization fields and URL userinfo. No
telemetry is collected by these workflows.
