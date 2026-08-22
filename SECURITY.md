# Security policy

## Supported versions

| Version | Supported |
| --- | --- |
| `main` and the current prerelease | Yes |
| Older prereleases and unmaintained branches | No |

After the first stable release, the latest stable minor release and `main` will receive security fixes unless a release note states otherwise.

## Report a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/sphireinc/git-watch/security/advisories/new) for suspected command injection, terminal escape injection, credential exposure, plugin-boundary bypass, arbitrary file access, or data-loss behavior.

Do not open a public issue containing exploit details. Include:

- affected gitwatch version or commit;
- operating system, architecture, terminal, and Git version;
- sanitized reproduction steps and expected/observed behavior;
- impact and whether user interaction is required; and
- sanitized logs or a minimal disposable repository when safe.

Never include credentials, private repository contents, or secret-bearing remote URLs. If private reporting is unavailable, email `juan.sanchez@juanleonardosanchez.com` with the subject `gitwatch security report` and ask for a secure transfer method before sending sensitive material.

For ordinary support requests, run `gitwatch --diagnostics` or create a
`--support-bundle` file, inspect it, and remove any personal paths or unrelated
details before sharing. Do not attach raw logs, provider responses, plugin
payloads, or repository archives.

Maintainers aim to acknowledge reports within three business days, establish severity and next steps within seven business days, and coordinate disclosure after a fix or mitigation is available. Complex or cross-platform issues may require more time; the reporter will receive status updates when practical.

## Security design

gitwatch invokes Git with typed argument vectors rather than shell command strings, treats watcher events as refresh hints, sanitizes untrusted terminal text, bounds provider/plugin output, redacts likely credentials, and requires explicit confirmation for destructive or history-changing actions. See [docs/security.md](docs/security.md) for the detailed threat model.
