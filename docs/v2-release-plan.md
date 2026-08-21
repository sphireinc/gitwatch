# v2 release plan

The v2 configuration schema is currently version `2`; unknown future
configuration versions are rejected. The plugin wire contract is currently
API version `1`, and its compatibility fixtures live under
`pkg/plugin/testdata/v1/`. These are the freeze points for the next release.

Before a v2 tag, maintainers must:

- review the schema and plugin fixtures as a compatibility change;
- update `CHANGELOG.md`, migration notes, and the version package;
- run `./scripts/release-check.sh` and `./scripts/verify-release-artifacts.sh`;
- perform clean-install and upgrade checks from the previous release;
- publish the signed tag, archives, SDK documentation, and announcement.

This plan intentionally records publication and announcement as external
release actions; passing local tests is not evidence that they occurred.
