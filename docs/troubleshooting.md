# Troubleshooting

## The repository is not refreshing

Run `gitwatch --watch=poll` to bypass filesystem notifications. A manual `r`
refresh is authoritative. Check that Git is on `PATH` and that the current
directory is inside a non-bare worktree.

## A remote or GitHub panel is unavailable

Remote and provider panels degrade independently of core status. Confirm the
remote with `git remote -v`, verify network access, and check that the selected
pull strategy is explicit. GitHub requires a token from the configured
environment variable or the `gh auth token` command; missing credentials do
not block local Git workflows.

## A plugin is unhealthy

Open `E`, inspect the manifest error and declared capabilities, then press `r`
to reload. Plugins are separate processes and output is bounded. Run
`./scripts/security-check.sh` when validating a local plugin installation.

## A multi-repository row is inactive or shows warnings

Inactive repositories use the configured group refresh policy and cached data.
Open the repository to make it active, or press `v` then `r` to refresh the
dashboard. A warning count indicates that an auxiliary stash/remote summary
failed while the authoritative status snapshot was still available.

## Release validation

Run `./scripts/release-check.sh` on the release host. It covers tests, race,
vet, build, demo repository behavior, security checks, and archive checksums.
Native Windows, terminal recordings, clean-machine installation, and release
publication require operator evidence and are not simulated by the script.
