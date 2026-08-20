# Task 11 — Responsive htop-style dashboard shell

**Priority:** P0

Build header, metric bar, primary file table, detail pane, activity strip, footer/help bar. Wide layout uses split panes; medium hides lower-priority metrics; narrow switches detail/activity to tabs or overlays. Define minimum supported terminal size and a graceful small-terminal screen.

**Acceptance:** Test at 200x60, 120x40, 80x24, and below-minimum sizes without clipping/panic.

**Status:** Complete — pure responsive layout calculation covers wide split panes, medium/narrow collapse behavior, activity/footer regions, minimum-size messaging, and hit-region containment tests.
