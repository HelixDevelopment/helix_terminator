# QA Transcript — helixtrack-bridge real Core integration

**Run ID:** qb-helixtrack-bridge-20260707
**Date:** 2026-07-07
**Feature:** helixtrack-bridge real Core `POST /do` JWT auth
**Commit:** 929b60a
**Tester:** AI agent (autonomous)

## What was tested
The helixtrack-bridge service's ability to make real, authenticated calls to the HelixTrack Core API using JWT tokens from the auth.tokenmanager.

## Evidence captured

### Test 1: Real Core API call with JWT auth
- **Action:** helixtrack-bridge sends `POST /do` to Core API with Bearer JWT
- **Expected:** 200 OK with real task data
- **Actual:** 200 OK — Core accepted the JWT and returned task data
- **Evidence:** Integration test `TestHelixTrackBridge_RealCoreIntegration` in `services/helixtrack-bridge-service/internal/coreclient/` — RED→GREEN proof: pre-fix returned fabricated `"pending"` response; post-fix returns real Core API response.

### Test 2: JWT token validation
- **Action:** helixtrack-bridge validates JWT from auth.tokenmanager
- **Expected:** Token parsed correctly, claims extracted
- **Actual:** Token validated, orgID/userID extracted from claims
- **Evidence:** Unit tests in `internal/coreclient/` prove token parsing.

### Test 3: Error handling on Core unreachable
- **Action:** helixtrack-bridge attempts call when Core is down
- **Expected:** Clean error response (not panic, not fabricated success)
- **Actual:** Returns honest error status
- **Evidence:** Test `TestHelixTrackBridge_CoreUnreachable` proves graceful degradation.

## Verdict
**PASS** — helixtrack-bridge makes real authenticated calls to Core API. No fabricated responses. JWT auth chain verified end-to-end.

## Captured evidence paths
- `services/helixtrack-bridge-service/internal/coreclient/*_test.go`
- `scratchpad/exec/session_20260708/cbs_review.diff`
