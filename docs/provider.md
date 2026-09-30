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

The optional PR workspace preserves an explicitly selected open PR while its
details load and rejects older in-flight responses for the same repository.
Workflow runs are a separate, TTL-cached Actions resource keyed by the selected
PR head SHA (or the local branch's authoritative HEAD when no PR is selected);
workflow-run failures are reported independently and never change local Git
status. Creating a PR never pushes implicitly. If the branch has no upstream or
has unpushed commits, the app offers the existing Remotes flow; users choose a
remote, invoke the guarded upstream push, and confirm it there before returning
to PR creation.

The workflow list loads at most 100 runs for the selected commit. `j`/`k` or a
click selects a visible workflow, `W` opens its log/run URL, `!` requests a rerun
of failed jobs, and `K` requests cancellation. Mutations require confirmation
of the workflow ID and attempt; a Checks API ID is never substituted for an
Actions workflow ID. If Actions data is unavailable, check summaries remain
readable but do not authorize workflow mutations. Successful actions invalidate
the cached workflow/check data before scheduling a provider reload.

Workflow status, conclusion, attempt, URL, and elapsed time are shown separately
from check summaries. Completed elapsed time uses the provider's last update
observation; it is not the sum of job durations. Queued runs without a start
timestamp do not fabricate a runtime. The same `github.cache_ttl` configuration
controls these optional reads; they do not fetch or modify local Git refs.
See [GitHub's Actions workflow-run API](https://docs.github.com/en/rest/actions/workflow-runs)
for the distinct workflow-run endpoints and exact-commit query.
