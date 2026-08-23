# Task 106 — Separate the commit tree from modified files

## Status

Complete.

## Objective

Make the optional commit-tree region visually distinct from the modified-file list by rendering a horizontal border across the top of the lower left pane.

## Acceptance criteria

- When the commit tree is enabled, a horizontal border is rendered at the first row of the commit-tree rectangle.
- The border spans the available left-panel width and remains correctly aligned in wide, medium, and narrow layouts.
- The right-side details panel is not shifted, resized, or visually separated by this new border.
- The separator does not become a selectable file row or a commit-tree data row.
- Commit-tree heading, status messages, wrapping, scrolling, and mouse coordinates remain correct after the separator is added.
- Add regression coverage for separator placement/content.
- Run formatting, linting, tests, race tests, vet, diff, security, and performance checks.
- Move this task to `tasks/completed/` after implementation.

## Implementation summary

Added a left-panel-width horizontal border as the first row of the commit-tree pane. The separator is styled with the semantic border role and the commit-tree viewport remains bounded beneath it.
