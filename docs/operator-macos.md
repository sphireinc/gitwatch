# macOS operator evidence

Recorded on 2026-08-21 from the current checkout in a disposable repository
created by `scripts/demo-repo.sh`, using a 120×35 tmux terminal session on
macOS 26.6.1 arm64 with Git 2.33.0 and tmux 3.6a.

Observed results:

- `gitwatch` launched inside the repository and displayed `main`, ready state,
  staged/modified/untracked counts, and three status rows.
- The selected `docs/notes.md` row accepted `d` and rendered its unified diff
  in the selected-diff pane without a mutation.
- The selected row accepted Space to stage and showed `STAGED 1`/`S`; a second
  Space restored `STAGED 0`/`M`, confirming the reversible stage/unstage path.
- `q` exited the session cleanly.
- With `GITWATCH_WATCH=poll`, the same disposable repository launched and
  rendered the ready status screen.
- From an empty directory, the binary exited with status 1 and a concise
  `not a git repository` diagnostic.

This is partial operator evidence only. Mouse selection, resize,
filesystem-watch event delivery, clean-machine installation, Git-missing
behavior, Linux, and Windows remain pending in the [beta validation matrix](beta-validation-matrix.md).
