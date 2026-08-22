# Terminal operator checklist

This checklist records native visual and interaction evidence separately from
automated rendering tests. Record the exact commit, OS/architecture, terminal
emulator, shell, Git version, dimensions, theme, `NO_COLOR`, motion mode, and
watch mode for each run.

## Spacing and layout

- [ ] Wide status view shows one cell of left/top panel inset and a visible
      divider without clipping either heading.
- [ ] Medium and narrow views retain the inset while keeping status, diff,
      activity, and footer content visible at the supported minimum size.
- [ ] Long paths, Unicode, tabs, diff lines, errors, binary notices, and clean
      state wrap inside the active panel rather than overwriting the divider.
- [ ] Resize from wide to narrow and back; selection, diff mode, scroll offset,
      and mouse targets remain aligned.

## Accessibility and capability

- [ ] Keyboard-only status, diff, search, stage/unstage, close, help, and quit
      workflows complete without mouse input.
- [ ] `NO_COLOR=1` preserves status meaning through symbols and text.
- [ ] High-contrast theme keeps headings, additions, deletions, warnings,
      conflicts, selection, and errors distinguishable.
- [ ] Reduced and off motion do not hide state or schedule unnecessary visual
      ticks.
- [ ] Unsupported mouse or terminal capabilities degrade to keyboard behavior
      without an error loop.

## Evidence rules

Screenshots and recordings document visual observations; they do not replace
before/after repository assertions, automated tests, or process-shutdown checks.
Do not publish repository contents, credentials, personal paths, or unredacted
provider/plugin output in evidence.
