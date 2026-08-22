# Task 100 — Provider resilience, authentication, and privacy boundaries

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 70–72, 86, 92, and 97

## Objective

Make optional GitHub/provider features dependable when disabled, unauthenticated,
offline, rate-limited, partially authorized, or serving malformed data, while
keeping core local Git workflows fully independent.

## Required work

- Define provider states: disabled, not configured, authenticating, available,
  stale-cache, rate-limited, unauthorized, unavailable, malformed, and canceled.
- Implement bounded timeouts, retry/backoff only for safe idempotent reads,
  cache TTL/stale policy, request budgets, and cancellation.
- Redact tokens, URLs with credentials, private repository identifiers where
  appropriate, response bodies, headers, and provider errors before display or
  logs.
- Make browser/open/copy integrations validate schemes and never expose tokens.
- Provide a clear opt-in authentication path and explain exactly what data is
  sent; never prompt for or persist secrets in the main config.
- Ensure provider failures never block status refresh, local mutations, quit, or
  repository switching.

## Acceptance criteria

- Core behavior is identical with provider support disabled or unavailable.
- Every provider state has a bounded UI representation and actionable recovery.
- Tests cover token source precedence, redaction, cache expiry, rate limits,
  malformed responses, HTTP cancellation, retries, browser validation, and
  provider isolation.
- Native evidence demonstrates offline startup and quit without hangs.

## Verification and documentation

Update security/threat-model/provider docs, privacy disclosure, configuration
schema, and release checklist. Use disposable test servers and non-secret
fixtures only; run full and race tests.

**Status:** Planned
