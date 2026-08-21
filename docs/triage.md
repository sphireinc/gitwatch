# Issue triage

## Labels

- `release-blocker`: data loss, crash, security issue, or broken installation that prevents the target release.
- `needs-triage`: a new report that has not been reproduced or assigned.
- `platform/macos`, `platform/linux`, `platform/windows`: environment scope.
- `performance`: measurable latency, allocation, resource, or refresh regression.
- `security`: secret, terminal, process, path, provider, or plugin-boundary concern; sensitive details belong in a private advisory.
- `enhancement`: a feature request outside the current release commitment.
- `documentation`: user, contributor, architecture, or release documentation.

## Milestones

- `v1.0.0`: release-blocking fixes and acceptance evidence for the first stable release.
- `v1.x`: compatibility-preserving fixes and deliberately scoped improvements.
- `v2.0`: explicit schema, API, command, or behavior compatibility changes.

Every report should include the environment and reproduction fields from the issue templates. Close a release blocker only with a reproducer, a fix commit, and automated regression coverage or native operator evidence.
