# Final Whole-Branch Review Report

**Branch:** main (655d586 → b64a8fd)
**Commits:** 40
**Reviewer:** Controller (manual review — opus agents stalled)
**Date:** 2026-07-09

## Verdict: CONDITIONAL PASS

All security fixes are correct and complete. Stress+chaos tests follow the established pattern. Minor issues identified — none blocking.

## Security Fixes Reviewed

### T20 — vault-service CallerUserID/CallerOrgID ✅
- `CallerUserID` reads from gin context first (JWT claims), falls back to X-User-ID header for backward compat
- `CallerOrgID` reads from context only (no header fallback) — correct, org comes from JWT
- Both handle string and uuid.UUID context types
- Nil-repo guards on all handlers
- **PASS**

### T21 — keychain-service ListItems ✅
- `callerUserIDFromContext` reads from gin context (JWT claims)
- ListItems uses context identity, not client query params
- Nil-repo guard present
- **PASS**

### T22 — nil-repo guards (both services) ✅
- All CRUD handlers check `h.repo == nil` before any repo call
- Returns 503 Service Unavailable
- **PASS**

### T23 — keychain :id ownership check ✅
- GetItem fetches item, then compares `item.UserID` vs `callerUserIDFromContext`
- Returns 404 for mismatch (not 403 — doesn't confirm existence)
- Same pattern for UpdateItem and DeleteItem
- **PASS**

### T24 — notification Types default ✅
- UpdatePreference defaults `Types` to `["all"]` when omitted
- Prevents NOT NULL violation on DB column
- **PASS**

### T14 — billing write-side IDOR tests ✅
- Tests confirm CreateSubscription/UpdateSubscription/CancelSubscription require caller identity from JWT context
- Existing fix was already in place — tests provide proof
- **PASS**

### T11 Minor — stale X-API-Key cors ✅
- Removed X-API-Key from notification+vault corsMiddleware Allow-Headers
- Fixed stale "service-to-service API key" comment
- **PASS**

## Stress+chaos Tests (§11.4.85) — 25/25 ✅

All 25 services have stress+chaos test coverage following the established pattern:
- Stress tests: sustained load (100 iterations), concurrent contention (15 goroutines), boundary conditions
- Chaos tests: input corruption, resource exhaustion, boundary conditions
- All tests use real PostgreSQL via podman container (no mocks)
- All tests pass with `-race` flag
- **PASS**

## Coverage Ledger ✅

- 194 test files, 834 test functions across fleet
- Honest gap analysis (43 stub files, 15 services without integration tests)
- **PASS**

## QA Transcripts ✅

- helixtrack-bridge: real Core POST /do JWT auth verified
- container-bridge: real ContainerRuntime (podman) + flag-injection fix
- ai-service: real LLM (llmprovider→llama.cpp) + timeout + audit
- **PASS**

## Minor Issues (Non-blocking)

1. **Duplicate commits** — some services have duplicate stress+chaos commits (agent wrote to main repo + controller extracted from worktree). Harmless — same files, same content. Can squash before tag if desired.

2. **Gateway TODOs** — legitimate future-work items (metrics, websocket proxy, SSO). Not bugs.

3. **T15 operator-blocked** — auth-service JWT key persistence needs KMS vs mounted-secret decision. Cannot fix without operator input.

## Recommendation

Proceed to §11.4.40 full retest, then release tag. The security fixes are solid, the stress+chaos coverage is comprehensive, and the documentation is honest about gaps.
