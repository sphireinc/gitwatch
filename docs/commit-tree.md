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

## Color and safety contract

gitwatch requests Git's captured graph with explicit color and the equivalent
of `--pretty=format:'%Cred%h%Creset -%C(auto)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset'`.
The captured SGR stream is an intermediate format only. A pure parser converts
hashes, decorations, dates, authors, graph text, and subjects into semantic
segments; the renderer maps those segments to gitwatch's dark, light, or
high-contrast theme roles. Git's palette is never passed through directly.

`NO_COLOR` and colorless terminals retain graph glyphs and commit text without
escape sequences. Unknown or malformed SGR, OSC, CSI, and control sequences
are discarded. Repository-controlled text remains bounded and sanitized.

## Inspecting a historical commit

Focus the tree with `T`, move to a commit with `j`/`k`, and press Enter or click
the commit row. gitwatch resolves the selected abbreviated hash without
checkout, loads its changed files into the upper-left list, and keeps the
commit's per-file diff in the right pane when a file is selected. Press `Esc`
or `1` to return to the current worktree status; inspection is read-only,
cancellable, and safely discarded when the repository changes.
