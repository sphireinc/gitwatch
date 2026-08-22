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
