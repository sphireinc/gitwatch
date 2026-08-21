# Demo and recording guide

Create deterministic mixed status data with:

```sh
./scripts/demo-repo.sh /tmp/gitwatch-demo
cd /tmp/gitwatch-demo
gitwatch
```

The target path must either not exist or be an empty directory. The script
refuses to overwrite files or non-empty directories; omit the argument to let
it create a uniquely named temporary directory and print that path.

For a recording, show the modified, untracked, and staged/unstaged states; click `docs/notes.md` to open the right-side diff pane; use `d` and Space to demonstrate keyboard parity; resize once to a narrow terminal and open the same diff overlay. Avoid relying on Nerd Fonts: the default ASCII status symbols are intentional.

The fixture is disposable and contains no credentials or network dependencies.

## Current evidence

A real asciinema recording from the disposable fixture is checked in at
[`docs/demo.cast`](demo.cast). It shows startup, selected-file diff, reversible
keyboard stage/unstage, a wide-to-narrow resize, and clean quit behavior. Play
it with `asciinema play docs/demo.cast`. The README image is a static rendered
snapshot of the recorded wide-terminal diff state.

This is partial demo evidence. Neither the cast nor the static README image
shows a mouse click, fuzzy filtering, or an external edit causing live refresh.
The cast also does not show the selected-file overlay after the terminal becomes
narrow.

## Task 33 completion capture

Before Task 33 can be marked complete, README-linked media from the current
release candidate must visibly demonstrate all of the following in a genuine
session:

1. An external edit appears without pressing `r`, proving live refresh.
2. A mouse click selects a file and opens its non-mutating diff/details pane.
3. `/` filters the status rows and clearing the query restores them.
4. Space stages and unstages the selected path with the refreshed state visible.
5. Keyboard diff opening shows the same selected path as the mouse workflow.
6. Resizing from wide to narrow preserves the workflow and shows the equivalent
   diff/details overlay.

Record the exact commit, OS, terminal, dimensions, Git version, and watch mode
with the capture. Scripted keystroke instructions or a static mockup do not
replace the recording.
