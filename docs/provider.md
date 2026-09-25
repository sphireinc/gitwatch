# Optional provider behavior

Provider support is opt-in. The UI represents provider data as `disabled`,
`not configured`, `authenticating`, `available`, `stale-cache`,
`rate-limited`, `unauthorized`, `unavailable`, `malformed`, or `canceled`.
Each state is bounded and recoverable: configure the documented token source,
retry a safe read, or continue using local Git workflows.

Only idempotent reads retry, with at most three short attempts. HTTP and body
sizes are bounded, request contexts are cancelable, and expired cached data may
be shown explicitly as stale when the provider cannot be reached. Tokens,
authorization headers, credential-bearing URLs, and response bodies are never
included in user-facing errors or logs.

In the multi-repository dashboard, GitHub checks and other provider-derived
attention are optional cached enrichments. Their detail is labeled `fresh` or
`stale`; an expired cache is never presented as current. Provider failure or
disabled GitHub support does not make local repository health unavailable.
The local clean/dirty/conflict and branch state continues to come from the
authoritative Git snapshot. Remote-fetch outcomes include their completion time
and measured duration so users can distinguish a recent fetch from old remote
information without polling remotes during every status refresh. See
[configuration](configuration.md) for the GitHub token source, cache lifetime,
and background-fetch controls.
