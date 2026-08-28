# Task 110 — Render the commit tree with gitwatch theme roles

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Task 109

## Objective

Render parsed segments with gitwatch semantic theme roles rather than passing
Git's terminal palette directly to the user.

## Requirements

Define mappings for dark, light, and high-contrast themes. Style graph topology,
hashes, branch/tag/HEAD decorations, subjects, relative dates, and authors while
keeping graph meaning understandable without color. Use lipgloss/theme roles,
not hard-coded raw ANSI. Respect `NO_COLOR`, high contrast, reduced/off motion,
safe wrapping, display width, and existing overflow behavior. Malformed parser
input must fall back to bounded colorless text.

## Tests and native evidence

Test every theme, colorless mode, high contrast, long/wrapped lines, wide
Unicode, decorations, malformed input, unsafe-control absence, and visible
width bounds. Record native terminal evidence for dark, light, high-contrast,
and `NO_COLOR` runs with merge graphs and decorated refs.

## Acceptance

Colorized trees are readable and theme-consistent, colorless/high-contrast
output remains semantically useful, and wrapping, scrolling, the separator, and
pane sizing are unchanged.

**Status:** Complete

## Completion summary

Added semantic commit-tree theme roles for graph topology, hashes, decorations,
subjects, dates, and authors across dark, light, and high-contrast themes.
Colorless mode remains readable and strips captured Git controls; the renderer
preserves the existing separator, wrapping, scrolling, and pane dimensions.
App, parser, theme, and safety regression tests pass. Integration refresh
coverage is completed by Task 111.
