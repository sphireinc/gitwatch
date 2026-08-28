# Status context panes

The lower-left portion of the Status workspace can show one read-only context
pane beneath the modified-file list. The right-side details/diff pane remains
full height and unchanged.

The available panes are:

- `T` — the optional bounded commit tree. The pane is visible at startup when
  `show_commit_tree` or `--with-commit-tree` is enabled; pressing `T` also
  opens it on demand for the current session.
- `P` — commits reachable from the current branch but not its configured
  upstream. It shows the ahead count and a bounded graph. A branch without an
  upstream reports that state explicitly.
- `B` — a read-only branch summary with current-branch and ahead/behind
  information. Use lowercase `b` for the full branch-management workspace.

The context shortcuts are built into gitwatch and work without a configuration
file. They can be overridden through `keymap` or `keymap_profiles`; collision
and reserved-control validation still applies. The active pane is independently
scrollable with `j`/`k`, Page Up/Page Down, Home/End, and mouse wheel.

All context data is loaded through bounded argument-vector Git commands. Pane
refreshes are asynchronous, cancelable, and generation-scoped. Commits, pulls,
pushes, fetches, branch switches, external ref changes, and authoritative
reconciliation refresh the relevant data. Selecting a context pane never
performs a Git mutation.
