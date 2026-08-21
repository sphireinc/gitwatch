# Task 18 — Embedded staged/unstaged diff viewer

**Priority:** P0

Add a viewport-based diff pane for the selected status path. On wide terminals it occupies the right side of the dashboard beside the file table; on medium/narrow terminals it becomes a tab or overlay without losing the selected path. Clicking a file row opens this pane immediately; `Enter` or `d` provides the keyboard-equivalent open action. Fetch diffs on demand, not every refresh. Support unstaged (`git diff -- path`) and staged (`git diff --cached -- path`) modes, syntax-preserving raw text, horizontal handling/truncation strategy, search optional if cheap. Detect binary files and show metadata instead of garbage.

**Acceptance:** Clicking a file visible in Git status opens that file’s diff/details in the right-side pane when layout permits, with an equivalent keyboard action and an obvious close/back action; large diff loading is asynchronous/cancellable; UI remains responsive; changing selection updates the pane without running a mutation; binary files show metadata rather than rendered binary bytes.

**Status:** Complete — cancellable asynchronous staged/unstaged diff loading, stale-result rejection, binary detection, path-safe Git arguments, and viewport tests are implemented. Wide terminals render a true right-side pane; medium/narrow terminals render a reversible overlay; mouse, `Enter`, and `d` open the same non-mutating view, and selection changes reload it.
