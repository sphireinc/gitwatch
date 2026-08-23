# Task 105 — Optional commit-tree pane in the status workspace

**Priority:** P2
**Lane:** v1.x user experience
**Dependencies:** Tasks 23, 24, 26, 34, 81, 93, 95, and 96

## Objective

Add an optional, bounded commit-history graph to the bottom quarter of the
left side of the status workspace. The feature should provide useful history
context while preserving the existing status-file workflow, selected-file
details/diff pane, responsive layout, accessibility behavior, and authoritative
refresh model.

The feature is disabled by default so existing layouts and startup performance
remain unchanged. When enabled, the status workspace shows the status-file list
in approximately the top three quarters of the left panel and a vertically
scrollable commit tree in the bottom quarter.

## User configuration

Support both configuration and CLI activation:

```sh
gitwatch --with-commit-tree
```

```json
{
  "show_commit_tree": true,
  "commit_tree": {
    "max_commits": 300
  }
}
```

Requirements:

- Add `show_commit_tree` as a versioned configuration field.
- Add `--with-commit-tree` as an explicit CLI flag.
- The CLI flag takes precedence over the configuration file and environment
  values, consistent with existing configuration precedence.
- The default is disabled (`false`). Existing configuration files must load
  unchanged and retain the existing status layout when the feature is off.
- Add a `commit_tree.max_commits` setting controlling the history bound.
- The default maximum is `100` commits, with the most recent commit at the top.
- Validate that `max_commits` is positive and enforce an implementation safety
  ceiling so a user cannot request unbounded history output. Document the
  selected ceiling and the behavior when a configured value exceeds it.
- Update the configuration schema, configuration inspection output, examples,
  CLI help, README configuration guidance, and migration/release notes.

## Git and data model

Use the requested presentation command in the Git boundary:

```text
git log --oneline --graph --decorate --all -n <max_commits>
```

Implementation requirements:

- Invoke Git with an argument vector through `internal/git`; never construct a
  shell command string.
- Keep this presentation-only history path isolated from the machine-readable
  status and mutation paths.
- Sanitize graph output, commit subjects, and decoration text before terminal
  rendering. Repository-controlled text must not inject terminal escapes.
- Bound stdout, line count, commit count, and rendering work before they reach
  the Bubble Tea model. A malformed or unexpectedly large response must fail
  safely with a visible, bounded diagnostic.
- Preserve graph characters, decoration markers, and useful commit subjects;
  do not silently replace the graph with a plain list.
- Include the current HEAD/ref identity in the loaded model so refreshes can
  avoid repeating Git history work when the relevant refs have not changed.
- Define behavior for empty history, unborn branches, detached HEAD, shallow
  clones, missing refs, malformed output, and Git capability failures.

## Layout and interaction

Wide status layout:

- Preserve the right details/diff panel unchanged.
- Divide the left panel into a status-files region occupying approximately the
  top 75% and a commit-tree region occupying approximately the bottom 25%.
- Keep a clear heading/divider such as `Commit tree · last 100`.
- Ensure each region retains its required heading, status message, and input
  affordances at the minimum supported terminal size.
- Use terminal cells and display width, not byte length, for all calculations.

Interaction requirements:

- Provide a clear way to focus or toggle the commit-tree pane without making
  ordinary status-file selection ambiguous.
- The focused tree scrolls vertically independently of the status-file list.
- Support keyboard scrolling, Page Up/Page Down, Home/End where consistent
  with the existing keymap, and mouse-wheel scrolling when mouse input is
  enabled.
- Preserve the tree scroll position across ordinary status refreshes when the
  referenced history remains valid.
- Clamp or reset the offset safely when the commit list changes or shrinks.
- Do not make a commit-tree click perform a checkout, revert, branch creation,
  or any other mutation. If commit selection is implemented later, it must be
  explicitly scoped to a future task.
- Keep existing file-row hit testing, stage controls, diff opening, filtering,
  and sorting behavior unchanged.

Responsive and accessibility requirements:

- At narrow widths/heights, collapse the tree into a switchable lower pane or
  equivalent overlay rather than reducing the status-file list below its
  minimum useful height.
- Never let a blank/padded tree cell hide a required status message or footer
  binding.
- Support `NO_COLOR`, high-contrast themes, reduced/off motion, keyboard-only
  use, and color-independent graph/decorations semantics.
- Wrap or safely clip long subjects and decorations without allowing terminal
  escape injection or horizontal overflow.
- Document the layout and input behavior in `UX_SPEC.md` and `KEYMAP.md`.

## Refresh and concurrency semantics

The tree must follow the authoritative refresh model:

- Load or refresh the commit tree after startup when enabled.
- Refresh it automatically after a successful commit operation.
- Refresh it when watcher/polling reconciliation observes a changed HEAD or
  relevant branch/ref state, including commits made outside gitwatch.
- Refresh it after repository switching and invalidate old tree results when
  the repository generation changes.
- Do not reload the tree merely because Bubble Tea renders another frame.
- Coalesce repeated refresh requests and keep Git history work cancelable.
- A slow or failing tree request must never block status refresh, mutations,
  confirmation dialogs, input, repository switching, or shutdown.
- Late results from an old repository, request, or generation must be ignored.
- A tree failure should leave the existing tree visible when safe and show a
  bounded, actionable status/notification; it must not put core Git status into
  an error state.

## Tests and fixtures

Add focused tests for:

- Configuration defaults, migration, schema validation, `max_commits` bounds,
  and CLI-over-config precedence.
- Exact argument vectors, including `--graph`, `--decorate`, `--all`,
  `--oneline`, and the bounded commit request.
- Parsing/presentation of graph branches, merges, decorations, long subjects,
  Unicode, control bytes, malformed output, empty history, detached HEAD,
  unborn branches, shallow repositories, and missing optional refs.
- Output and allocation budgets for repositories with more than 100 commits.
- Independent vertical scrolling, offset clamping, focus transitions, mouse
  hit testing, resize behavior, narrow-layout fallback, wrapping, `NO_COLOR`,
  high contrast, and reduced/off motion.
- Refresh after successful commits and external HEAD/ref changes.
- Cancellation, duplicate/coalesced requests, repository switching, shutdown,
  stale generations, and tree failures while status remains usable.
- No process or goroutine leaks in canceled, failed, and shutdown paths.
- Existing status-file selection, filtering, sorting, stage/unstage, and diff
  behavior when the tree is enabled and disabled.

Use disposable real Git repositories for integration coverage. Keep native
terminal rendering, mouse, resize, and terminal-restoration checks separate
from automated assertions.

## Documentation and release evidence

Update:

- `README.md` feature/configuration sections.
- `ARCHITECTURE.md` refresh and status-workspace flow.
- `UX_SPEC.md` layout, focus, scrolling, and narrow-terminal behavior.
- `KEYMAP.md` with the final toggle/focus/scroll bindings.
- `docs/configuration.md` and `docs/configuration.schema.json`.
- `docs/advanced-workflows.md` or a dedicated `docs/commit-tree.md`.
- `docs/beta-validation-matrix.md` and `docs/release-checklist.md` with the
  native evidence required for enabled/disabled, wide/narrow, resize, and
  keyboard/mouse scenarios.

## Acceptance criteria

- With no new configuration or flag, gitwatch behaves and lays out exactly as
  before.
- `--with-commit-tree` and `show_commit_tree: true` enable the pane, with CLI
  precedence and an explicit effective-config representation.
- Enabled mode shows a vertically scrollable graph with the newest 100 commits
  by default, newest first, while preserving graph/decorations presentation.
- A higher configured bound works within the documented safety ceiling; values
  outside validation rules fail before the TUI starts.
- The tree updates after successful commits, external HEAD/ref changes,
  repository switching, and authoritative reconciliation, without blocking
  core status or shutdown.
- Status-file interactions and the right details/diff panel retain their prior
  behavior in both modes.
- All repository-controlled tree text is sanitized and bounded.
- Automated tests, race tests, vet, lint, performance/security gates, and
  native operator evidence are complete or explicitly recorded as pending.

**Status:** Complete

**Completion summary:** Added the opt-in bounded commit graph to the status
workspace with `--with-commit-tree` and `show_commit_tree`, a default 100-commit
limit capped at 1000, independent focus/scrolling, responsive layout regions,
sanitized argument-vector Git history loading, generation-scoped cancellation,
and refresh integration. Updated configuration, UX, architecture, keymap,
release evidence, and commit-tree documentation. Native terminal evidence
remains an operator-owned release-matrix item.
