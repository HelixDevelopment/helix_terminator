# QA Transcript — ai-service real LLM integration

**Run ID:** qb-ai-service-20260707
**Date:** 2026-07-07
**Feature:** ai-service real local LLM call (llmprovider generic → llama.cpp)
**Commit:** 8fb5a8c
**Tester:** AI agent (autonomous)

## What was tested
The ai-service's ability to make real LLM inference calls via the llmprovider abstraction (backed by local llama.cpp).

## Evidence captured

### Test 1: Real LLM completion
- **Action:** ai-service sends a prompt to local LLM via llmprovider
- **Expected:** Real inference response (not fabricated `"pending"`)
- **Actual:** Real completion returned from llama.cpp backend
- **Evidence:** Integration test `TestGenericClient_Complete_LiveHelixLLMContainer` — RED→GREEN: pre-fix returned fabricated `"pending"` string; post-fix returns real LLM completion. Test SKIPs honestly when backend absent (§11.4.3).

### Test 2: Timeout handling
- **Action:** ai-service handles LLM timeout
- **Expected:** Clean 504 response on deadline exceeded
- **Actual:** 504 returned with descriptive error, no hang
- **Evidence:** Synchronous-timeout regression fixed — pre-fix hung indefinitely; post-fix returns 504 on deadline.

### Test 3: Startup env-invariant check
- **Action:** ai-service validates `AI_LLM_TIMEOUT` < `AI_HTTP_WRITE_TIMEOUT` at startup
- **Expected:** Fatal on misconfiguration (write timeout shorter than LLM timeout)
- **Actual:** `log.Fatalf` on invalid config, prevents silent deadline mismatch
- **Evidence:** `TestValidateTimeoutInvariant` — 5 test cases prove the invariant check works.

### Test 4: Audit-persist path
- **Action:** ai-service persists inference audit records
- **Expected:** Audit record written to DB on each inference
- **Actual:** Best-effort persist (does not block response if DB slow)
- **Evidence:** `TestAuditPersistPath` — RED→GREEN: removed CreateRequest call → test fails; restored → passes.

## Verdict
**PASS** — ai-service makes real LLM calls. No fabricated responses. Timeout handling, startup validation, and audit persistence all verified.

## Captured evidence paths
- `services/ai-service/internal/llmclient/*_test.go`
- `services/ai-service/internal/handler/*_test.go`
- `scratchpad/exec/session_20260708/ai_review.diff`
