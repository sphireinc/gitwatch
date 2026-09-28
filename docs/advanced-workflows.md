# Advanced workflows

This document describes the advanced workbench surfaces included in the first
stable release. Commands are exposed only where the corresponding validation,
confirmation, cancellation, and authoritative refresh workflow is complete.

The release is still gated by native operator acceptance. These workflow notes
describe implemented behavior; they do not replace the platform evidence in
the [release checklist](release-checklist.md).

## History and patch work

History loading uses bounded pages and machine-readable fields for SHA, parents,
refs, signature state, author, timestamp, and subject. Commit inspection can
load parent-relative patches, changed-file statistics, and a path-filtered
view. Partial staging builds a patch from stable hunk/line identities and runs
`git apply --check` before changing the index.

Path history and blame are bounded investigation surfaces. Blame uses
`git blame --line-porcelain -L <start>,<end> -- <path>`, keeps raw origin data
separate from sanitized terminal rendering, and opens the selected origin
commit against its first parent without changing the worktree.

Status presentation remains backed by the same complete flat Git snapshot.
Press `O` to switch between flat rows and a collapsible directory tree; `[` /
`]` collapse or expand all directories and `Enter` toggles the selected
directory. Filtering and sorting happen on file entries before the tree index
is rebuilt, so directory counts describe only the currently visible children.

From a selected history commit inspector, `H` opens the same stable hunk/line
selection surface used for partial staging. Pressing `Enter` previews an
inverse patch and an explicit rebase plan that pauses at the selected commit;
the plan reports later commits that will be replayed. The patch is checked
against both the worktree and index before it is applied, then the existing
amend composer is used. Patch or replay conflicts remain in standard recovery,
and `Ctrl-X` aborts through normal rebase abort semantics. This is a published
history rewrite and may require coordination with downstream users.

When Git reports an active rebase—including one started in another terminal—
Status offers `C` to open Rebase recovery. The recovery pane shows Git-derived
phase, current commit, completed/remaining counts, and conflict count; an
edit stop is labeled explicitly. Use `c` to continue when the index or edit
stop permits it, `s` to skip the current commit, or `x` to abort. Lifecycle
actions refresh the authoritative repository state after Git finishes.
When a rebase finishes, the completion message and operation journal show the
original and resulting HEAD. If Git's original and onto commits are available,
the journal also counts newly created commits reachable from the result but
not from either earlier boundary; a missing count is left unavailable rather
than guessed.

The cherry-pick progress workspace uses the current branch as its target and
shows Git's ordered commit IDs. Git does not retain a unique source branch for
an externally started pick, so that field is labeled unavailable instead of
inferred. At narrow widths the recovery footer lists only currently valid
Continue, Skip, and Abort actions before navigation shortcuts.
When Git has already applied an earlier commit but no longer retains its
source ID in the sequencer todo, the completed row shows the resulting commit
ID derived from the sequencer's original HEAD; conflicted and pending rows
continue to show Git's source IDs.

Multi-commit reverts use the same durable recovery controls. After a restart,
the revert pane reads Git's sequencer state and original HEAD; when Git has
discarded completed source IDs, it shows the resulting revert commit IDs
without guessing which source commits they replaced. Continue, Skip, and
Abort act on the selected repository and refresh its authoritative status.

## Bisect and recovery

An active bisect can be reopened from Status with `C` or from the command
palette. In the Bisect workspace, `S` starts a session by asking for the
known-bad and known-good refs and then confirming them. Use `g`, `b`, or `s` to
mark the current candidate good, bad, or untestable; `i` inspects its commit,
and `x` asks before resetting to the branch where the bisect began. `1` returns
to live Status while the bisect remains active, so you can run tests and keep
watching the checked-out candidate. External bisect changes are reconstructed
from Git when the repository refreshes.

For an automated test, press `A`, enter the executable path, then enter each
argument separately. Press Enter after each value; an empty argument finishes
input, and `y` confirms the run. The command is passed to
[`git bisect run`](https://git-scm.com/docs/git-bisect#_bisect_run) as an
executable plus argv values; gitwatch does not build a shell command string.
Only run programs you trust: Git executes the selected program against each
candidate revision, and that program can read or change local files or access
the network. Output is streamed to the workspace after terminal-text
sanitization and is capped at 64 KiB for display. The operation has a
30-minute timeout; `Ctrl-C` shuts down gitwatch and cancels the running Git
process tree. Git's resulting bisect state and refreshed repository snapshot,
not the displayed output, determine the outcome.

## External tools and custom commands

From Status, `Ctrl-E` opens the selected path in the configured editor, `Ctrl-O`
opens it with the configured file opener, and `Ctrl-T` invokes the configured
difftool. Tool definitions are typed executable-plus-argument templates; they
never run through a shell. If no difftool is configured, `Ctrl-T` uses Git's
typed `difftool --no-prompt HEAD -- <path>` operation. gitwatch requests a
normal authoritative refresh after the external process exits.

Custom commands use the same process boundary. Each command declares an
executable, an argument array, an allowed context, an optional timeout, and
whether it mutates repository state. Supported placeholders are `{repo}`,
`{path}`, `{sha}`, `{branch}`, `{remote}`, `{tag}`, `{url}`, and
`{prompt:<id>}`. Typed prompts support validated text, masked secrets,
confirmation, and static or already-loaded branch/remote/tag/commit/path
selections. The form must be fully submitted before any process starts; `Esc`
cancels it. Commands do not accept shell strings or unknown placeholders.
Text and secret prompts can set Unicode character length limits in addition
to pattern validation. Validation failures keep the current field open and
show an error. Placeholders are expanded once, preserving braces and other
literal characters in selected paths and submitted prompt values.
Mutating commands request a status refresh when they finish; commands requiring
confirmation use the prompt/form workflow before execution.

Secret prompt input is masked and redacted from command output, diagnostics,
and operation records. When a secret is substituted into an argv value, the
operating system may still expose that argument to process-inspection tools
available to the same user. Do not use this transport for credentials that
must be hidden from other same-user processes; use a tool that accepts secrets
through a protected channel such as standard input.

See [configuration](configuration.md) for the JSON shape and
[the keymap](../KEYMAP.md) for the built-in bindings.

## Stashes, branches, and worktrees

Stash and branch mutations validate refs/names before invoking Git. Deletion
requires exact confirmation and never permits deleting the checked-out branch.
Worktree discovery uses `git worktree list --porcelain`; lifecycle commands are
typed argv operations and branch occupancy is explicit.

In Branches, select a local or remote-tracking branch and press `M` to merge it
into the currently checked-out branch. The prompt requires an explicit
strategy: `merge` (normal Git behavior), `ff-only`, `no-ff`, or `squash`.
Remote-tracking refs retain their qualified name, such as `origin/topic`.
Merging requires a clean worktree; gitwatch does not auto-stash, reset, or
force a merge, and it refuses a merge while another Git operation is active
or when the source branch is checked out in another linked worktree.

If Git reports a conflict, gitwatch opens the common conflict-recovery view
with the authoritative conflict snapshot. Resolve and stage the paths before
using `c` to continue; use `x` to abort. Merge does not offer Skip. With
`squash`, Git stages the combined changes but creates no merge commit; review
the staged result and commit it separately.

## Remotes and GitHub

Remote URLs are redacted before display or diagnostics. Pull strategy must be
explicit (`merge`, `rebase`, or `ff-only`), and force pushing is represented
only by the opt-in `--force-with-lease` operation. GitHub support is optional:
remote detection, environment/GitHub CLI token sources, bounded cached PR and
repository data, and check-run parsing run separately from local Git refresh.
A branch with no open pull request is a normal state. Independent provider
failures appear as sanitized resource warnings, and unavailable/authentication/
rate-limit states never mark the local repository unavailable. Creating a pull
request is an explicit provider action and never pushes the local branch.

## Plugins and multi-repository work

Plugins use the dependency-free `pkg/plugin` wire contract and execute out of
process with bounded output. The registry discovers explicit roots with depth
and repository limits, skips symlink traversal and VCS/vendor directories,
persists private versioned JSON metadata with atomic replacement, and refreshes
status via a bounded worker pool. A missing or failing repository becomes an
independent error row; it does not block healthy repositories. Repository rows
are filterable/sortable; favorites and groups are stored as registry metadata.
Health severity is semantic rather than a single numeric score. Local state is
derived from the authoritative Git snapshot and remains useful offline; the
dashboard shows its source and observation time. Provider-derived CI attention
is optional cached data marked `fresh` or `stale`. Remote-fetch outcome,
completion time, and measured duration are shown separately, so remote
freshness is not inferred from local status and remotes are not probed on every
status refresh. See [provider behavior](provider.md) and
[configuration](configuration.md) for cache and auto-fetch controls.

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

Configuration schema version 3 can be validated with:

```sh
gitwatch --config-check
gitwatch --config-inspect
```

Git commands always use argument vectors. Terminal text is sanitized and
diagnostics redact token/password/authorization fields and URL userinfo. No
telemetry is collected by these workflows.
