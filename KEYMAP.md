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
| ! | Toggle conflict-only status filter (in Status view) |
| R then type `yes` | Restore the selected tracked path after exact-scope confirmation (in Status view) |
| d | Open diff |
| 1 | Status view |
| b | Branches view |
| / / s | Filter / sort branches (in Branches view) |
| c / R | Create / rename branch (in Branches view) |
| u / N | Set / unset branch upstream (in Branches view) |
| D / X | Confirm normal / force branch deletion (in Branches view) |
| s | Stashes view |
| l | History view |
| ] | Load more history (in History view) |
| / | Search history (in History view) |
| Enter | Inspect selected commit (in History view) |
| x then y | Confirm checkout of selected commit (in History view) |
| B | Create a named branch at selected commit (in History view) |
| R | Revert selected commit after typing its exact SHA (in History view) |
| t | Load tag refs (in History view) |
| M / f / g / y | Inspect next parent / filter inspected path / jump to ref / copy SHA (in History view) |
| n | Remotes view |
| w | Worktrees view |
| v / Enter | Repositories dashboard / open selected repository |
| G | GitHub workspace (when enabled) |
| E | Plugin workspace (when enabled) |
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

Destructive actions must use deliberately distinct bindings and confirmation dialogs.

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
