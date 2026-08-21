# macOS operator evidence

Recorded on 2026-08-21 from the current checkout in a disposable repository
created by `scripts/demo-repo.sh`, using a 120×35 terminal session on macOS.

Observed results:

- `gitwatch` launched inside the repository and displayed `main`, ready state,
  staged/modified/untracked counts, and three status rows.
- The selected `docs/notes.md` row accepted `d` and rendered its unified diff
  in the selected-diff pane without a mutation.
- The selected row accepted Space to stage and showed `STAGED 1`/`S`; a second
  Space restored `STAGED 0`/`M`, confirming the reversible stage/unstage path.
- `q` exited the session cleanly.

This is partial operator evidence only. Mouse selection, stage/unstage, resize,
filesystem-watch versus polling, clean-machine installation, Git-missing
behavior, Linux, and Windows remain pending in the [beta validation matrix](beta-validation-matrix.md).
