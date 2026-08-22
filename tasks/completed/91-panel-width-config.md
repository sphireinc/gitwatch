# Task 91 — Configurable status panel widths

**Priority:** P1

Add validated configuration for the wide status view's left file panel and right details/diff panel. Preserve the existing responsive behavior for medium, narrow, and too-small terminals.

**Acceptance:** Users can set `layout.files_percent` and `layout.details_percent` in the versioned configuration file; defaults remain 60/40; totals above 100 are logged once at startup and normalized to 50/50; other invalid values are rejected by config validation; wide layout, mouse hit testing, and diff rendering use the configured split; long content wraps in both panels without breaking row selection; documentation and the JSON schema describe the settings; focused and repository-wide tests pass.

**Status:** Complete — added validated 60/40 defaults, one-time overflow normalization to 50/50, configurable wide-layout wiring, wrapped content in both panels, wrapped-row mouse mapping, documentation, schema coverage, and repository-wide verification.
