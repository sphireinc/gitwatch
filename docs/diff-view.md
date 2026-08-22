# Diff inspection

The status view opens a read-only diff for the selected path. In a wide
terminal it appears in the right panel; in medium and narrow terminals it
occupies the equivalent detail area. Opening a diff never stages, discards, or
otherwise mutates repository content.

## Controls

- `Enter` or `d` opens the selected path's default diff mode.
- `V` switches between the staged/index and unstaged/worktree diff for the
  selected path and reloads it asynchronously.
- `PageUp`, `PageDown`, and the mouse wheel scroll the diff.
- `/` starts a bounded, case-insensitive search in the loaded diff.
- `n` moves to the next search match after a search has been accepted.
- `Esc` closes search first, then closes the diff pane.
- `H` opens the hunk-selection workflow when the loaded text is a supported
  patch.

The heading reports the active mode, path, addition/deletion counts, and the
available mode/search controls. Search changes the viewport to the matching
line; it does not rewrite or reformat the patch.

## Budgets and truncation

Diff loading is bounded by the `diff.max_bytes` and `diff.max_lines` settings
in the JSON configuration. Defaults are 4 MiB and 20,000 lines. If either
limit is reached, gitwatch renders the received prefix and an explicit
`diff truncated at configured budget` notice. Increase the limits only for
trusted repositories with sufficient memory; a large diff still runs outside
the Bubble Tea render path and remains cancellable when the selection changes,
the mode switches, or the pane closes.

Binary, rename, copy, conflict, empty, malformed, failed, and stale-result
states remain explicit. Unsupported binary/rename/copy line-selection flows
continue to refuse partial mutation rather than guessing at patch semantics.

## Safety and limitations

Git remains the source of truth and is invoked through the typed argument-vector
runner. Repository-controlled text is sanitized before rendering. Search is
case-insensitive in this version and operates only on the bounded loaded diff;
it does not search Git history or fetch additional content.
