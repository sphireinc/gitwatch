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

## Supersede — terminal-native Git workspace

The next product direction expands gitwatch into a terminal-native Git
workspace capable of replacing the advanced workflow surface users expect from
LZ. This is a planned capability lane, not a claim that those workflows are
already shipped. Interactive rebase, fixup/autosquash, cherry-pick, merge and
conflict resolution, reflog recovery, bisect, submodules, full tag and remote
management, deep GitHub workflows, and repository-scoped multi-repository
operations are planned in Tasks 121–186.

Parity is necessary but insufficient. The Supersede milestone also requires
gitwatch to remain stronger at live filesystem-driven status, bounded
multi-repository operation, operation observability, safety and recovery, and
repository health. See [PARITY_MATRIX.md](PARITY_MATRIX.md) for the current
support baseline, closing tasks, and executable acceptance evidence.

The live watcher/status pipeline remains the product core: filesystem events
are refresh hints, and porcelain-v2 Git status remains authoritative. Advanced
workflows must preserve the existing multi-repository, typed-argv, refresh,
sanitization, bounded-work, keyboard/mouse, `NO_COLOR`, and reduced-motion
contracts.

## v2 — explicit compatibility boundary

v2 is reserved for changes that require a documented configuration, plugin API, command, or behavior compatibility break. A v2 release requires migration notes, frozen compatibility fixtures, clean-install and upgrade evidence, and the same cross-platform acceptance standards as v1.

The following remain prohibited product directions: arbitrary in-process
third-party UI, an embedded Git implementation, telemetry, and silent
destructive shortcuts. In particular, gitwatch will not add generic
`reset --hard`, raw `--force` push, or `clean -fd` shortcuts.
