# Demo and recording guide

Create deterministic mixed status data with:

```sh
./scripts/demo-repo.sh /tmp/gitwatch-demo
cd /tmp/gitwatch-demo
gitwatch
```

For a recording, show the modified, untracked, and staged/unstaged states; click `docs/notes.md` to open the right-side diff pane; use `d` and Space to demonstrate keyboard parity; resize once to a narrow terminal and open the same diff overlay. Avoid relying on Nerd Fonts: the default ASCII status symbols are intentional.

The fixture is disposable and contains no credentials or network dependencies.

An asciinema-compatible captured frame from the disposable fixture is checked
in at [`docs/demo.cast`](demo.cast). Play it with `asciinema play
docs/demo.cast`; then use the steps above for the interactive mouse, diff,
stage, and resize portions of a complete recording.
