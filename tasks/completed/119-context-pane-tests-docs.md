# Task 119 — Test and document context panes

**Priority:** P2
**Lane:** v1.x UX and release evidence
**Dependencies:** Tasks 116–118

## Objective

Document and validate the lower-left context-pane family across supported
repositories, terminals, themes, and keymap configurations.

## Required coverage

Test linear/merge history, ahead/behind/no-upstream/detached/unborn states,
missing refs, shallow history, large bounds, external changes, pane switching,
keyboard/mouse scrolling, wrapping, resize, narrow layouts, `NO_COLOR`, high
contrast, reduced/off motion, keymap overrides, collisions, and unchanged file
and right-panel behavior.

Update README, UX/architecture/keymap/configuration guidance, commit-tree and
troubleshooting docs, beta matrix, and release checklist. Separate automated
results from native terminal evidence.

## Acceptance

Documentation is consistent, reproducible native scenarios are recorded, and
all normal quality gates pass.

**Status:** Complete

## Implementation summary

Added Git and app regression coverage, README/keymap/configuration/
architecture/UX documentation, context-pane and commit-tree guidance, and
troubleshooting/beta-matrix updates. Automated tests, race tests, vet,
format/diff checks, security checks, and performance checks pass locally.
