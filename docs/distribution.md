# Distribution and upgrades

## Maintained channels

The maintained v1 channels are:

- GitHub Release archives for macOS amd64/arm64, Linux amd64/arm64, and
  Windows amd64. The protected release workflow produces checksums, SBOM, and
  provenance, and verifies archive contents before publication.
- Source installation with `go install`, using the canonical module and tag.

Package-manager formulas or manifests are optional integrations, not release
dependencies. A channel may be disabled or rolled back without deleting a
GitHub Release or weakening repository controls. A maintainer must own the
manifest, update cadence, rollback procedure, and independent install test
before adding a new channel.

## Verification and upgrade safety

Every release emits `gitwatch_<version>_release.json` from the checked-out
commit and canonical version, and includes it in `SHA256SUMS`. Inspect the
metadata, verify the checksum, and confirm the archive contains only the
executable, MIT license, README, dependency notices, and third-party license
files. The executable mode is checked on Unix and the Windows executable is
checked separately.

Upgrades do not rewrite `$XDG_CONFIG_HOME/gitwatch/config.json`, the private
repository registry, or plugin trust/enable state. If an upgrade is delayed,
continue using the prior verified archive; if a channel is partially
published, remove only that channel's listing and retry after verification.
Never replace a verified release asset in place.
