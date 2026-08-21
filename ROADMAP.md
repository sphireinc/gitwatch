# Roadmap

gitwatch is preparing its first public stable release. The implementation task history remains available under `tasks/`; this roadmap summarizes user-facing release direction without replacing those acceptance records.

## v1.0.0 — first public release

- Complete native macOS, Linux, and Windows terminal acceptance for keyboard, mouse, resize, filesystem-watch, polling, stage/unstage, diff, and clean shutdown behavior.
- Validate clean installation and upgrade paths, Git-missing/non-repository diagnostics, and supported archive formats.
- Close release-blocking correctness, security, data-loss, and performance defects.
- Publish signed release metadata, checksummed archives, dependency notices, provenance, package-manager metadata, and demo assets.

## v1.x — compatibility-preserving improvements

- Address field feedback and terminal/platform compatibility gaps.
- Improve discoverability, diagnostics, performance, and accessibility without breaking configuration or plugin API contracts.
- Add deliberately scoped day-to-day workflows that preserve the safety and refresh model.

## v2 — explicit compatibility boundary

v2 is reserved for changes that require a documented configuration, plugin API, command, or behavior compatibility break. A v2 release requires migration notes, frozen compatibility fixtures, clean-install and upgrade evidence, and the same cross-platform acceptance standards as v1.

Interactive rebase, arbitrary in-process third-party UI, an embedded Git implementation, telemetry, and silent destructive shortcuts are not planned.
