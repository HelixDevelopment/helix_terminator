# Coverage Ledger — §11.4.25 Full-Automation-Coverage

**Revision:** 1
**Last modified:** 2026-07-08T14:30:00Z
**Generated against:** `main` HEAD `c2718d2` (clean tree, remote==local)
**Method:** direct `find`/`grep`/`cat` against the current working tree — no build, no test execution, no guessing (§11.4.6). Every count below cites the exact command or file(s) read.

---

## 0. Fleet-wide totals

| Metric | Count |
|---|---|
| Services under `services/*/` | 25 |
| Test files (`*_test.go`) | 194 |
| Test functions (`func Test...`) | 834 |
| Services with integration tests | 10 (18 files) |
| Services with stress tests | 1 (auth-service only) |
| Services with chaos tests | 1 (auth-service only) |
| Dedicated security test files | 4 |
| Flutter client test files | 11 (8 stubs + 2 real + 1 e2e stub) |

**Change since prior draft (f0b29ff → c2718d2):** +46 test files, +315 test functions, stress+chaos landed for auth-service, terminal-service stubs replaced with real tests, 6 new integration test files across billing/container-bridge/keychain/user services.

---

## 1. Per-service coverage table

Legend: **R**=real (substantive assertions against actual logic) · **M**=moderate (real CRUD exercised, but via in-memory double or all-negative-path only) · **S**=shallow (compiles+runs, only checks `New()` returns non-nil / not-connected error) · **X**=stub-bluff (tautology, empty-body, or unconditional t.Skip placeholder) · **A**=absent (no file). Integration/Security/Stress/Chaos: **R**=present+real, **A**=absent.

| Service | Files | Funcs | Main | Model | Repo | Server | Handler | Extra | Integ. | Security | Stress | Chaos |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| ai-service | 9 | 28 | X | X | X | X | R(3) | — | A | A | A | A |
| analytics-service | 6 | 21 | X | R(2) | S | S | R(7) | — | A | A | A | A |
| audit-service | 6 | 35 | X | R(6) | M(1,all-neg) | R(4) | R(14) | — | A | A | A | A |
| auth-service | 12 | 45 | R(integ) | R(4) | R(5) | R(3) | R(3) | crypto R(4) | R(4 files) | partial | **R(4)** | **R(3)** |
| billing-service | 8 | 18 | X | X | X | X | R(3) | — | R(2 files) | A | A | A |
| collaboration-service | 6 | 31 | X | R(2) | S | R(2) | R(16) | — | A | A | A | A |
| config-service | 6 | 26 | X | R(2) | R(3) | R(3) | R(8) | — | A | A | A | A |
| container-bridge-service | 14 | 37 | X | R(2) | S | S | R(8) | red_injection R(1) | R(1 file) | R(injection) | A | A |
| gateway-service | 6 | 37 | X | X | X | X(stub) | R(23+2) | resolve_path_params R(1) | R(2 funcs) | R(SSRF) | A | A |
| health-service | 6 | 33 | R(minimal) | R(5) | S | R(5) | R(11) | checker R(10) | A | A | A | A |
| helixtrack-bridge-service | 9 | 33 | X | R(2) | S | S | R(8) | coreclient R(2) | A | A | A | A |
| host-service | 6 | 24 | R(2) | R(4) | R(2) | R(3) | R(4) | — | A | A | A | A |
| keychain-service | 9 | 41 | X | X | X | X | R(3) | crypto R(5) | R(1 file) | A | A | A |
| notification-service | 14 | 105 | X(taut) | X(taut) | R(12) | R(10) | R(14+3) | delivery R(25, 5 files) | R(4 files) | R(injection+ssrf+idor) | A | A |
| org-service | 6 | 20 | X | R(2) | S | S | R(6) | — | A | A | A | A |
| pki-service | 8 | 30 | X | R(4) | S | S | R(5) | crypto R(8) | R(1 file) | A | A | A |
| port-forward-service | 9 | 34 | X | R(2) | S | S | R(8+4) | forwarder R(5+3) | R(1 file) | A | A | A |
| recording-service | 6 | 23 | X | R(3) | S | S | R(8) | — | A | A | A | A |
| sftp-service | 6 | 23 | X | R(3) | S | S | R(8) | — | A | A | A | A |
| snippet-service | 6 | 21 | X | R(1) | S | S | R(8) | — | A | A | A | A |
| ssh-proxy-service | 8 | 32 | R(2) | R(4) | M(in-memory) | R(2) | R(3) | sshclient R(5), wshandler R(5) | A | A | A | A |
| terminal-service | 7 | 41 | X | R(3) | R(2) | R(5) | R(9) | recorder R(13) | A | A | A | A |
| user-service | 7 | 17 | X | X | X | X | R(3) | — | R(1 file) | A | A | A |
| vault-service | 8 | 60 | R(49L) | R(2) | R(2)+R(integ,3) | R(9)+R(integ,7) | R(15) | — | R(2 files) | R(IDOR) | A | A |
| workspace-service | 6 | 19 | X | R(3) | S | R(1) | R(4) | — | A | A | A | A |

Counts in parens are Test-func counts per layer, cited from per-file `grep -c "func Test"`.

---

## 2. §11.4.85 Stress/chaos coverage

| Service | Stress | Chaos | Notes |
|---|---|---|---|
| auth-service | **R** (415L, 4 funcs) | **R** (404L, 3 funcs) | Real: sustained-load + concurrent-contention + boundary-conditions (stress); input-corruption + resource-exhaustion + boundary-conditions (chaos). Gated on `//go:build stress`/`//go:build chaos` tags. Conditional t.Skip when podman unavailable (§11.4.3-compliant). |
| All other 24 services | **A** | **A** | Zero stress/chaos tests fleet-wide outside auth-service. |

**Gap:** 24/25 services have zero stress and zero chaos test coverage per §11.4.85. This is the single largest test-type gap in the repo.

---

## 3. §11.4.27 Stub/tautology inventory

### Tautology stubs (`assert.True(t, true)`, zero real assertions)

| File | Function | Notes |
|---|---|---|
| `notification-service/cmd/notification-service/main_test.go` | `TestMainCompiles` | Constructs nothing, asserts `true` |
| `notification-service/internal/model/model_test.go` | `TestNotificationTypes` | Constructs 2 structs (discarded), asserts `true` |

**Count:** 2 files, 2 functions.

### Live `t.Skip` placeholder stubs (unconditional, not DB-gated, not comment-only)

41 files across 22 services contain a live `t.Skip("...")` that fires unconditionally every run. The 15 services with the most stub files:

| Service | Stub files | Pattern |
|---|---|---|
| ai-service | 4 (cmd, model, repo, server) | t.Skip placeholder |
| billing-service | 5 (cmd, model, repo, server, server_integ) | t.Skip placeholder |
| keychain-service | 4 (cmd, model, repo, repo_integ) | t.Skip placeholder |
| user-service | 4 (cmd, model, repo, server) | t.Skip placeholder |
| gateway-service | 3 (cmd, model, repo) | t.Skip placeholder |
| All others | 1-2 each (mostly cmd/main_test.go) | t.Skip placeholder |

**DB-gated migration tests (NOT stubs, correctly implemented):** 25 files across all 25 services — each conditional on `DATABASE_URL`/`TEST_DATABASE_URL` with real executing counterparts when DB is present. These are §11.4.3/§11.4.69-compliant and NOT part of the gap.

**Comment-only false positives:** 0 remaining (auth-service `cmd/main_test.go` was previously a false positive — the `t.Skip` string appears only inside a code comment; the live file is a real 90-line PG-backed integration test).

**Total stub surface:** 43 files (41 t.Skip + 2 tautology) across 23 services contribute zero real coverage despite compiling and reporting PASS.

---

## 4. §11.4.83 docs/qa/ status

**`docs/qa/` does not exist.** No end-to-end QA transcripts have been committed. Per §11.4.83, every feature that ships MUST carry a recorded e2e communication transcript under `docs/qa/<run-id>/`. This is a release-blocking gap.

---

## 5. §11.4.25 Six-invariant coverage per service

Per §11.4.25, every feature must satisfy six invariants before deliverable. Current status:

| Invariant | Status | Notes |
|---|---|---|
| (1) Anti-bluff posture | PARTIAL | auth/vault/notification/gateway/port-forward have captured-evidence tests. 15+ services have stub-only or shallow-only coverage. |
| (2) Working capability end-to-end | PARTIAL | 10 services have integration tests. 15 services have zero integration tests. |
| (3) Working implementation matching docs | NOT ASSESSED | No docs/qa/ transcripts exist to cross-reference. |
| (4) No open issues/bugs | NOT ASSESSED | No docs/Issues.md or workable-items DB exists. |
| (5) Full documentation | NOT ASSESSED | No user manual entries verified. |
| (6) Four-layer test floor | PARTIAL | Pre-build gates exist; post-build/runtime/paired-mutation layers not verified in this pass. |

---

## 6. Flutter client (`clients/flutter`)

11 test files total:

| File | Status | Evidence |
|---|---|---|
| `test/api_client_test.dart` | STUB | `expect(true, isTrue)` |
| `test/auth_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/collaboration_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/host_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/notification_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/terminal_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/vault_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/workspace_bloc_test.dart` | STUB | `expect(true, isTrue)` |
| `test/widget_test.dart` | REAL (minimal) | 1 assertion, splash-screen only |
| `test/terminal_view_golden_test.dart` | REAL (deep) | 149L, 3 testWidgets, §11.4.170-compliant golden+content-oracle+VT-engine |
| `integration_test/app_test.dart` | STUB (e2e) | Boot-only smoke, explicit `TODO: implement real e2e tests` |

**Gap:** 8/11 files are literal no-op stubs. The sole e2e test does not drive any real user journey.

---

## 7. Release-blocking gaps (ordered by risk per §11.4.132)

1. **[CRITICAL] 24/25 services have zero stress/chaos tests.** §11.4.85 mandates stress+chaos for every fix. Only auth-service complies. This is the largest uniform gap.

2. **[CRITICAL] 43 test files are stubs (zero real coverage).** 41 t.Skip placeholders + 2 tautologies across 23 services. These compile and report PASS while exercising nothing.

3. **[HIGH] 15/25 services have zero integration tests.** analytics, audit, collaboration, config, health, helixtrack-bridge, host, org, recording, sftp, snippet, ssh-proxy, terminal, workspace, ai — no integration-tagged test file exists.

4. **[HIGH] `docs/qa/` does not exist.** §11.4.83 requires e2e QA transcripts for every shipped feature. Zero transcripts committed.

5. **[HIGH] 4 services (ai/billing/keychain/user) have model+repo+server layers 100% stub.** The real pgx-backed production repository code has zero executing test coverage beyond the handler's 3 constructor-adjacent tests.

6. **[MEDIUM] Flutter client: 8/11 test files are no-op stubs.** The sole e2e test is boot-only with an explicit TODO.

7. **[MEDIUM] notification-service has 2 tautology stubs** (main_test.go + model_test.go) — `assert.True(t, true)` with no real assertions.

8. **[MEDIUM] keychain-service stores private keys/passphrases in plaintext Postgres columns.** `private_key TEXT NOT NULL` + `passphrase TEXT` with zero encryption anywhere in the codebase. This is a security defect, not a test gap.

9. **[LOW] ~13 services have shallow-only repository/server coverage.** Constructor non-nil + "database not connected" negative-path only, no positive-path CRUD assertion.

---

## 8. Changes since prior draft (f0b29ff → c2718d2)

| Change | Detail |
|---|---|
| +46 test files | 148 → 194 (fleet-wide) |
| +315 test functions | 519 → 834 (fleet-wide) |
| auth-service stress+chaos landed | `handler_stress_test.go` (415L, 4 funcs) + `handler_chaos_test.go` (404L, 3 funcs) — real, §11.4.85-compliant |
| terminal-service stubs replaced | `repository_test.go` and `server_test.go` rewritten from tautology stubs to real tests (2+5 funcs respectively) |
| 6 new integration test files | billing (2), container-bridge (1), keychain (1), user (1), auth readiness (1) |
| config/pki/workspace main_test.go rewritten | Now have proper §11.4.3 skip documentation (honest SKIP, not bluff) |
| org-service main_test.go changed | From tautology to t.Skip placeholder (still a stub, different pattern) |

---

## 9. Notes / honest boundaries

- Test **execution** (green/red) was **not run** in this pass — static/read-only inventory per §11.4.6. All classifications are based on reading source, not `go test` output.
- `UNCONFIRMED`: whether any background fix streams landed after `c2718d2` — the controller should re-diff `git log --oneline -5` before curating this ledger into release gates.
- This ledger supersedes the prior draft at `scratchpad/exec/session_resume/coverage_ledger_draft.md` (generated at f0b29ff).
