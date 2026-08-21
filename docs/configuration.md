# Configuration

gitwatch reads JSON from `GITWATCH_CONFIG`, then
`$XDG_CONFIG_HOME/gitwatch/config.json`, or the platform configuration
fallback. Files without a version and version 1 files are migrated in memory
to the current version 2 defaults; they are never rewritten automatically.
Future versions are rejected so an older binary cannot silently discard new
settings.

The v2 top-level fields are:

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
| `keymap` | Action-to-key bindings; duplicate keys are rejected and configured bindings are applied at runtime. Supported navigation actions include `quit`, `help`, `status`, `branches`, `stashes`, `history`, `remotes`, `worktrees`, `repositories`, `commit`, and `refresh`. |

Validate a file without opening the TUI with:

```text
gitwatch --config-check --config /path/to/config.json
```

Environment variables override matching scalar settings, and CLI flags take
precedence over both file and environment values.

The machine-readable v2 schema is available at
[`docs/configuration.schema.json`](configuration.schema.json). Duration values
use the same JSON nanosecond representation as the Go configuration type.
