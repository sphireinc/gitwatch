# Task 13 — Mouse interaction model

**Priority:** P0

Enable Bubble Tea mouse events. Single click selects row/pane; wheel scrolls focused pane; explicit clickable stage indicator toggles stage state; header/footer affordances may be clickable. Never bind double click to discard/delete/reset. Compute hit regions after final layout so resizing cannot invoke the wrong row.

**Acceptance:** Mouse actions have keyboard equivalents; hit testing tested for scroll offsets and resized layouts.
