# Demo and recording guide

Create deterministic mixed status data with:

```sh
./scripts/demo-repo.sh /tmp/gitwatch-demo
cd /tmp/gitwatch-demo
gitwatch
```

Maintainers can capture the complete deterministic interaction with
`asciinema`, `tmux`, and a freshly built binary:

```sh
asciinema record --overwrite --capture-input --output-format asciicast-v2 \
  --window-size 160x30 \
  --headless --return \
  --command "./scripts/record-demo.sh ./gitwatch /tmp/gitwatch-demo" \
  docs/demo.cast
```

The capture driver creates an isolated tmux socket, makes the external edit,
sends the SGR mouse event and keyboard actions, resizes the active tmux window
to 80x24, and tears the temporary session down after a clean quit.

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

The cast records an external watcher-driven edit, an SGR mouse click opening the
selected-file diff, fuzzy filter entry and clearing, reversible stage/unstage,
keyboard diff parity, and a wide-to-narrow resize that preserves the selected
path in the overlay. The checked-in driver performs the click; its resulting
diff pane is visible even though terminal recorders do not draw a pointer.

## Task 33 completion capture

Task 33 evidence must continue to demonstrate all of the following in a genuine
session whenever the media is replaced:

1. An external edit appears without pressing `r`, proving live refresh.
2. A mouse click selects a file and opens its non-mutating diff/details pane.
3. `/` filters the status rows and clearing the query restores them.
4. Space stages and unstages the selected path with the refreshed state visible.
5. Keyboard diff opening shows the same selected path as the mouse workflow.
6. Resizing from wide to narrow preserves the workflow and shows the equivalent
   diff/details overlay.

The checked-in capture was recorded from code commit `524e92917b1a` on macOS
26.6.1 arm64 with asciinema 3.2.0, tmux 3.6a, Git 2.33.0, filesystem watch mode,
and Apple Terminal-compatible SGR mouse input. It starts at 160x30 and resizes
to 80x24. The capture script is checked in for reproducibility; the cast, rather
than a static mockup, is the acceptance evidence.
