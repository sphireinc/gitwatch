# Post-v1 issue triage

Create these labels and milestones in the hosting project before or immediately
after the first public release. The issue templates provide the initial labels
automatically.

## Labels

- `release-blocker`: data loss, crash, security issue, or broken install;
  cannot ship in the target release.
- `needs-triage`: new report has not yet been reproduced or assigned.
- `platform/macos`, `platform/linux`, `platform/windows`: environment scope.
- `performance`: measurable latency, allocation, or refresh regression.
- `security`: secret, terminal, process, path, or plugin boundary concern.
- `enhancement`: feature request outside the current release commitment.

## Milestones

- `v1.0.0 stabilization`: only release-blocking fixes and acceptance evidence.
- `v1.1`: post-launch fixes and deliberately scoped improvements.
- `v2.0`: schema/API changes, larger workflows, and incompatible changes.

Every beta report should include the environment and reproduction fields from
the templates. Close a release-blocker only with a reproducer, a fix commit,
and a regression test or operator evidence.
