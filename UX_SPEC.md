# UX Specification

## Visual goal
Think `htop` for a Git repository: dense, legible, interactive, colorful where supported, animated but not distracting, and useful even when the repository is clean.

## Default desktop layout
```text
┌ gitwatch · repo-name ─ main ↑2 ↓1 ─ origin/main ─ clean/dirty indicator ─ 12ms ┐
│ STAGED 3 │ MODIFIED 5 │ UNTRACKED 2 │ CONFLICTS 0 │ +142 -37 │ watcher ●      │
├ Files ─────────────────────────────────────┬ Selected file ────────────────────┤
│ S  M  path                       +   -     │ status / old path / mode           │
│ ●  M  internal/app/app.go       +42  -8   │ staged: yes · unstaged: yes        │
│    ?  notes.md                             │ compact diff/stat/metadata          │
│                                           │                                    │
├ Activity ──────────────────────────────────┴────────────────────────────────────┤
│ 09:41:03 staged internal/app/app.go · 18ms                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│ [space] stage/unstage  [d] diff  [a] stage all  [u] undo  [/] filter  [?] help │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Terminal-cell spacing contract

The dashboard uses terminal cells, not CSS pixels, as its portable spacing unit.
Status panels reserve one cell of left inset and one cell of top breathing room;
wide layouts reserve one divider cell between files and details. Wrapped rows,
scroll offsets, and mouse hit regions must use the same inset calculations. The
header, metric bar, activity strip, and footer remain single-cell rows so the
minimum-size contract stays deterministic.

At narrow sizes, preserving readable content and input affordances takes
precedence over maintaining the wide split. A blank or padded cell is not
allowed to hide a required heading, status message, error, or footer binding.

When enabled, the left status panel reserves approximately its lower quarter
for a `Commit tree` region and keeps the status-file list above it. `T` toggles
focus between the file list and tree; the focused tree supports `j/k`,
Page Up/Page Down, Home/End, and mouse-wheel scrolling. The tree is context
only: selecting it never performs a Git mutation. Narrow layouts may collapse
the tree to the lower switchable region when space is constrained.

## Interaction
- Arrow keys / j,k move selection.
- Space toggles staged state for selected path according to current status.
- Enter opens/expands selected-file details.
- `a` stages all visible or all repository changes only after the action is clearly described in status/help UI.
- `U` or a deliberate binding unstages all.
- `d` opens diff view; allow switching staged/unstaged diff.
- `/` activates fuzzy filtering.
- `s` cycles sort modes.
- `g` then `g` jumps top; `G` jumps bottom.
- `r` forces refresh.
- `?` opens full help overlay.
- `q` quits when no modal is open; Esc closes the active modal/pane first.
- Mouse click on a file row selects it and opens its non-destructive diff/details pane on the right in wide layouts, or the equivalent tab/overlay in narrow layouts. Double-click must not perform destructive actions. A click on an explicit stage control may stage/unstage and must remain distinct from the row hit target.
- Mouse wheel scrolls active list/pane.

## Animation
Use tasteful animation for state changes: spinner during Git work, pulse/flash when status changes, smooth-ish progress indication, transient success/error toast, and optional subtle row transition highlighting. Do not animate every frame unnecessarily. Support `--motion=full|reduced|off`.

## Empty/clean state
Do not show a dead blank screen. Show branch/upstream information, last refresh, recent activity, repository statistics available cheaply, and a playful clean-worktree message selected from a small tasteful set. Do not let jokes interfere with professional use.

## Accessibility
- Fully keyboard operable.
- Do not communicate status by color alone; use symbols/text.
- Honor `NO_COLOR`.
- Provide high-contrast-safe semantic theme choices.
- Reduced motion/off modes.
- Avoid requiring mouse interaction.
