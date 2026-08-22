# Configuration

gitwatch reads JSON from `GITWATCH_CONFIG`, then
`$XDG_CONFIG_HOME/gitwatch/config.json`, or the platform configuration
fallback. Files without a version and version 1 files are migrated in memory
to the current version 2 defaults; they are never rewritten automatically.
Future versions are rejected so an older binary cannot silently discard new
settings. The configuration schema version is independent of the gitwatch
release version; the schema reached version 2 during prerelease development.

The schema-version-2 top-level fields are:

| Field | Purpose |
| --- | --- |
| `version` | Configuration schema version; current value is `2`. |
| `theme` | `auto`, `dark`, `light`, or `high-contrast`. |
| `motion` | `full`, `reduced`, or `off`. |
| `watch` | `auto`, `fs`, or `poll`. |
| `interval`, `reconciliation`, `debounce` | Positive/ non-negative Go duration values encoded as JSON numbers. |
| `show_untracked`, `show_ignored`, `mouse` | Status and input preferences. |
| `repositories` | Roots, groups, per-group refresh intervals, discovery depth, and repository limit. |
| `remote` | Pull strategy, stale threshold, and worker limit. |
| `github` | Optional provider enablement, token environment name, and cache TTL. |
| `plugins` | Optional plugin directories, enablement, and output limit. |
| `notifications` | Notification preferences; set `quiet` to suppress attention badges while retaining history. |
| `layout` | Wide status split; `files_percent` controls the left file panel and `details_percent` controls the right details/diff panel. They must be positive and sum to `100`; defaults are `60` and `40`. |
| `diff` | Diff inspection budgets; `max_bytes` defaults to `4194304` and `max_lines` defaults to `20000`. Truncated diffs show an explicit notice. |
| `keymap` | Action-to-key bindings; duplicate keys are rejected and configured bindings are applied at runtime. Supported navigation actions include `quit`, `help`, `status`, `branches`, `stashes`, `history`, `remotes`, `worktrees`, `repositories`, `commit`, and `refresh`. |

Validate a file without opening the TUI with:

```text
gitwatch --config-check --config /path/to/config.json
```

For example, make the two wide status panels equal width:

```json
{
  "version": 2,
  "layout": {
    "files_percent": 50,
    "details_percent": 50
  },
  "diff": {
    "max_bytes": 4194304,
    "max_lines": 20000
  }
}
```

If the two layout percentages total more than `100`, gitwatch logs one startup
error and safely uses a `50`/`50` split for that session. Totals below `100`,
zero values, and negative values remain configuration errors.

Environment variables override matching scalar settings, and CLI flags take
precedence over both file and environment values.

The machine-readable schema is available at
[`docs/configuration.schema.json`](configuration.schema.json). Duration values
use the same JSON nanosecond representation as the Go configuration type.
