# Task 13 — Mouse interaction model

**Priority:** P0

Enable Bubble Tea mouse events. A single click on a file row selects it and opens the selected-file diff/details pane on the right when the terminal is wide enough; on narrow terminals it opens the equivalent diff/details overlay or tab. Wheel scrolls the focused pane. An explicit clickable stage indicator toggles stage state. Header/footer affordances may be clickable. Never bind double click to discard/delete/reset. Compute hit regions after final layout so resizing cannot invoke the wrong row.

**Acceptance:** Mouse actions have keyboard equivalents; clicking a status-file row selects it and reveals its per-file diff/details view without performing a mutation; hit testing is tested for scroll offsets and resized layouts; explicit stage controls remain distinct from the row hit region.

**Status:** Planned — mouse row-to-diff-pane behavior is specified for implementation with the file table and diff viewer.
