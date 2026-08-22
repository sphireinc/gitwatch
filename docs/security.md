# Security threat model

gitwatch treats repository contents, Git metadata, remote URLs, GitHub
responses, terminal text, and plugin input/output as untrusted data.

## Process and protocol boundaries

- Git is executed through `exec.CommandContext` argument vectors; filenames and
  refs are never interpolated into shell command strings.
- Plugins run out of process. A manifest is validated before execution, and
  requested capabilities must be present in the host grant.
- Plugin stdout/stderr is bounded. The public newline-delimited protocol rejects
  empty messages, oversized messages, and oversized type/ID fields. Invalid or
  mismatched handshakes are refused.
- Plugin cancellation is tied to the process context; restart supervision is
  bounded by an explicit restart count and backoff.

## Data and terminal safety

- Secret-bearing URLs and diagnostic text are redacted before presentation.
- User-controlled terminal text is sanitized before rendering.
- Repository discovery skips symlinked entries and rejects symlinked `.git`
  metadata to avoid following an unexpected filesystem boundary.
- Binary, rename, and copy diffs are refused by line-oriented partial patch
  operations; whole-file Git operations remain the supported path.

Automated coverage includes hostile-process fixtures and a checked-in staged
and full-history secret scan. Native platform review is recorded in the
[release sign-off](release-signoff.md) and never inferred from local automated
tests alone.

## Optional provider privacy boundary

GitHub/provider support is opt-in and sends only the selected repository owner,
repository name, branch/ref, and requested read-only endpoint data. Tokens come
from the configured environment or an authenticated CLI source; they are never
written to the main configuration or support bundles. Provider responses,
headers, token-bearing URLs, and response bodies are bounded and redacted
before diagnostics. Disabled, unauthenticated, offline, rate-limited,
malformed, stale-cache, and canceled states leave local Git status and
mutations usable.
