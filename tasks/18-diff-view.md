# Task 18 — Embedded staged/unstaged diff viewer

**Priority:** P0

Add a viewport-based diff overlay/pane. Fetch diffs on demand, not every refresh. Support unstaged (`git diff -- path`) and staged (`git diff --cached -- path`) modes, syntax-preserving raw text, horizontal handling/truncation strategy, search optional if cheap. Detect binary files and show metadata instead of garbage.

**Acceptance:** Large diff loading is asynchronous/cancellable; UI remains responsive.
