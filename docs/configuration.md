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
| `repositories` | Roots, groups, per-group refresh intervals, discovery depth, repository limit, and symlink/ignored-directory policy. |
| `remote` | Pull strategy, stale threshold, and worker limit. |
| `github` | Optional provider enablement, token environment name, and cache TTL. |
| `plugins` | Optional plugin directories, enablement, and output limit. |
| `notifications` | Notification preferences; set `quiet` to suppress attention badges while retaining history. |
| `layout` | Wide status split; `files_percent` controls the left file panel and `details_percent` controls the right details/diff panel. They must be positive and sum to `100`; defaults are `60` and `40`. |
| `diff` | Diff inspection budgets; `max_bytes` defaults to `4194304` and `max_lines` defaults to `20000`. Truncated diffs show an explicit notice. |
| `show_commit_tree`, `commit_tree` | Optional status-pane commit graph; disabled by default, with `max_commits` defaulting to `100` and capped at `1000`. |
| `profile`, `keymap_profiles` | Optional named keymap profile and profile definitions. |
| `keymap` | Direct action-to-key overrides; these take precedence over the selected profile. Duplicate keys, unknown actions, reserved terminal controls, and destructive-action remaps are rejected before startup. |

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

Keymap precedence is defaults, selected `keymap_profiles.<profile>`, then the
direct `keymap` object. `GITWATCH_PROFILE` and `--profile` select a profile;
the CLI flag wins. Only the documented non-dangerous navigation/workspace
actions are configurable. This keeps restore, delete, discard, and force
operations behind their deliberate built-in confirmations.

The commit tree can be enabled without changing other layout settings:

```json
{
  "show_commit_tree": true,
  "commit_tree": { "max_commits": 100 }
}
```

The CLI flag `--with-commit-tree` also enables it and takes precedence over the
file value. The graph is loaded with a bounded Git log request and refreshes
when HEAD/ref state changes.

The machine-readable schema is available at
[`docs/configuration.schema.json`](configuration.schema.json). Duration values
use the same JSON nanosecond representation as the Go configuration type.
