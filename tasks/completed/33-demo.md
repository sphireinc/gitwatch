# Task 33 — Demo assets and visual polish pass

**Priority:** P1

Create a deterministic demo repository/script that produces attractive mixed status states for screenshots and terminal recordings. Polish spacing, truncation, border hierarchy, status glyphs, animations, clean-state screen, and error modals across common terminal emulators. Do not rely on Nerd Fonts by default; optionally enhance when detected/configured.

**Acceptance:** README media demonstrates live refresh, mouse selection, stage/unstage, filter, diff, responsive resizing.

**Status:** Complete — the deterministic mixed-state fixture, reproducible
tmux/asciinema capture driver, README-linked cast, and static preview cover
external watcher-driven refresh, SGR mouse selection, fuzzy filtering,
authoritative stage/unstage refreshes, keyboard diff parity, wide right-pane and
narrow overlay behavior, and clean quit. Capture provenance and replay steps are
documented in `docs/demo.md`.
