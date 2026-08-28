# Optional commit tree

Enable the status-workspace graph at startup with `--with-commit-tree` or
`"show_commit_tree": true`. Pressing `T` opens it on demand for the current
session even when startup visibility is disabled. It occupies approximately the lower quarter of
the left panel and leaves the right details/diff pane unchanged. The default
request is the newest 100 commits; `commit_tree.max_commits` may increase that
bound up to 1000.

Press `T` in Status view to focus the tree. The lower-left context region can
also be switched to unpushed commits with `P` or the read-only branch summary
with `B`; lowercase `b` still opens branch management. Focused trees support `j/k`,
Page Up/Page Down, Home/End, and mouse-wheel scrolling. The tree is read-only;
it never checks out or mutates a commit. It refreshes after authoritative HEAD
or ref changes, successful commits, repository switching, and reconciliation.
Old repository or request generations are discarded.

Graph output is sanitized and bounded. Empty/unborn histories show `No commits
yet`; malformed, unavailable, or canceled history requests leave local status
usable and show a bounded error message. Native release evidence should cover
the feature disabled/enabled, wide/narrow layouts, resize, scrolling, external
commits, and keyboard/mouse parity.
