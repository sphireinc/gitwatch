# Default Keymap

| Key | Action |
|---|---|
| ↑/↓, j/k | Move selection |
| PgUp/PgDn | Page |
| gg / G | Top / bottom |
| Enter | Open the selected path's diff/details (in Status view) |
| Left click a file row | Select the path and open its diff/details (in Status view) |
| Space | Stage or unstage selected path |
| a | Stage all tracked, untracked, and deleted paths (in Status view) |
| U | Unstage all while preserving working-tree content (in Status view) |
| S | Cycle status-file sort mode (in Status view) |
| O | Toggle flat/tree status presentation (in Status view) |
| [ / ] | Collapse all / expand all status directories (in tree mode) |
| ! | Toggle conflict-only status filter (in Status view) |
| R then type `yes` | Restore the selected tracked path after exact-scope confirmation (in Status view) |
| d | Open diff |
| Ctrl-E / Ctrl-O / Ctrl-T | Open selected file in editor / opener / difftool (in Status view) |
| 1 | Status view |
| b | Branches view |
| / / s | Filter / sort branches (in Branches view) |
| c / R | Create / rename branch (in Branches view) |
| u / N | Set / unset branch upstream (in Branches view) |
| D / X | Confirm normal / force branch deletion (in Branches view) |
| s | Stashes view |
| l | History view |
| h | Path history for the selected Status/inspected path |
| L | Bounded blame for the selected Status/inspected path |
| ] | Load more history (in History view) |
| / | Search history (in History view) |
| Enter | Inspect selected commit (in History view) |
| x then y | Confirm checkout of selected commit (in History view) |
| B | Create a named branch at selected commit (in History view) |
| R | Revert selected commit after typing its exact SHA (in History view) |
| H | Edit selected historical commit through a guarded patch/rebase flow (in History view) |
| t | Load tag refs (in History view) |
| M / f / g / y | Inspect next parent / filter inspected path / jump to ref / copy SHA (in History view) |

| n | Remotes view |
| w | Worktrees view |
| v / Enter | Repositories dashboard / open selected repository |
| G | GitHub workspace (when enabled) |
| E | Plugin workspace (when enabled) |
| I | Open the repository-scoped `.gitignore` manager |
| o / y | Open GitHub PR / copy its URL (in GitHub view) |
| c | Open the first check URL (in GitHub view) |
| A / D / P | Add / remove / prune worktrees (in Worktrees view) |
| Enter | Open selected repository or worktree |
| H | Open hunk selection for the currently loaded diff |
| f | Fetch selected remote (in Remotes view) |
| C | Create stash (in Stashes view) |
| a / p / D | Apply / pop / drop selected stash (in Stashes view) |
| u | Toggle include-untracked while creating a stash |
| m / e / o | Pull with merge / rebase / fast-forward-only strategy |
| p | Push current branch (in Remotes view) |
| u / T | Push current branch with upstream tracking / push an explicitly entered tag (in Remotes view) |
| P then y | Confirm force-with-lease push (in Remotes view) |
| c | Commit workspace |
| Ctrl-S | Execute a valid commit |
| A / N / o / S / @ | Toggle amend / no-edit / signoff / signing / author override (in Commit view) |
| Tab | Cycle focus/panes |
| / | Filter status files |
| r | Force refresh |
| T | Focus the optional commit tree in Status view |
| P | Open unpushed commits in the lower-left Status pane |
| B | Open the read-only branch summary in the lower-left Status pane |
| j/k, PgUp/PgDn, Home/End | Scroll the focused lower-left context pane |
| ? | Help |
| Ctrl-P | Open command palette |
| Ctrl-N | Dismiss newest notification attention |
| Esc | Close overlay / cancel |
| q | Quit |

## Path-history controls

| Key | Action |
|---|---|
| j / k | Move through the loaded history entries |
| Enter | Inspect the selected commit |
| Y | Compare the selected commit with `HEAD` |
| f | Toggle explicit rename-following mode |
| ] | Load the next bounded history page |
| Esc | Return to the prior workspace |

## Blame controls

| Key | Action |
|---|---|
| j / k | Move through loaded blame lines |
| Enter | Open the selected origin commit |
| ] | Load the next bounded blame-line page |
| Esc | Return to the prior workspace |

## Historical-edit controls

| Key | Action |
|---|---|
| H | Open the parent-relative hunk selector from a selected commit inspector |
| Enter | Preview the inverse patch and guarded rebase plan; the paused commit then opens the amend composer |
| Ctrl-X | Abort the rebase and restore the original history |

## `.gitignore` manager

| Key or input | Action |
|---|---|
| j / k or arrows | Move through templates |
| Space | Select or deselect a template |
| / | Search the catalog |
| Tab | Change catalog tabs |
| c | Clear the search |
| Esc | Return to the prior workspace |
| a | Preview an append, or create a missing `.gitignore` |
| p | Preview the selected operation |
| d | Preview exact owned-block removal |
| u | Preview an owned-block update |
| m | Preview adoption of a matching unmanaged template |
| y | Confirm the displayed preview |
| n or Esc | Cancel the displayed preview |
| r | Refresh the optional upstream catalog |
| b | Return to the embedded offline catalog |
| Mouse | Select rows and the visible selection control |
| Safety | Destructive actions require deliberately distinct bindings and keyboard confirmation |

## Profiles and validation

The JSON configuration can define named profiles:

```json
{
  "profile": "writer",
  "keymap_profiles": { "writer": { "quit": "x", "help": "h" } },
  "keymap": { "refresh": "R" }
}
```

The direct `keymap` object overrides the selected profile, which overrides the
defaults. Select a profile with `GITWATCH_PROFILE` or `--profile`. Invalid
actions, duplicate keys, blank/overlong sequences, and terminal controls such
as `ctrl+c` are rejected by `--config-check`. The profile and effective
bindings are included in `--config-inspect`; secrets are not part of the
keymap model.
