# HelixTerminator — Issues / Work Tracker

**Revision:** 1
**Last modified:** 2026-07-14T00:00:00Z

The project's formal open-work tracker (Constitution §11.4.15 Status / §11.4.16 Type /
§11.4.54 stable id). Seeded 2026-07-14 by the enterprise-readiness kick-off pass — before
this there was no tracker (planning lived in a git-ignored `progress.md`). Full analysis:
[`docs/gap_analysis/GAP_ANALYSIS_20260714.md`](gap_analysis/GAP_ANALYSIS_20260714.md).

**Status vocabulary:** Queued · In progress · Ready for testing · In testing · Reopened ·
Operator-blocked · Implemented (→ done) · Completed (→ done) · Fixed (→ done).
**Type vocabulary:** Bug · Feature · Task. **Closure = positive captured evidence (§11.4).**

Follow-up: bootstrap the §11.4.93 SQLite workable-items DB as the single source of truth
(HT-INFRA-001) and migrate this Markdown into it.

---

## Phase 1 — Core-product security & functionality

### HT-SSHCA-001 — PKI short-lived SSH certificate authority (crypto core)
- **Type:** Feature
- **Status:** Implemented (→ done)
- **Priority:** Critical
- **What:** pki-service could issue x509 TLS certs but NOT the short-lived SSH user/host
  certificates mandated by `SERVICE_REGISTRY.md §19` — the enterprise cert-based-SSH auth
  model (hosts trust a CA; sessions present principal-scoped certs that expire in minutes).
- **Done:** `services/pki-service/internal/sshca` — `GenerateCA`, `GenerateKeyPair`,
  `SignUserCertificate`, `SignHostCertificate`, `VerifyCertificate`. 11 tests (`-race`,
  80.8% cov, deterministic ×3), golden-bad self-validation (§11.4.107(10)) + independent
  OpenSSH `ssh-keygen -L` oracle. Caught+fixed a real verify defect (untrusted-CA acceptance).
- **Evidence:** `docs/qa/HT-SSHCA-20260714/` (RED baseline, GREEN result, ssh-keygen oracle).
- **Branch:** `feature/pki-ssh-certificates`.

### HT-SSHCA-002 — PKI SSH-cert HTTP endpoints + persistence
- **Type:** Feature
- **Status:** Queued
- **Priority:** Critical
- **What:** Expose HT-SSHCA-001 via the API: `POST /api/v1/pki/ssh-ca` (create CA, store
  CA key encrypted-at-rest reusing `internal/crypto.EncryptPrivateKey`), list/get CA,
  `POST /api/v1/pki/ssh-ca/:id/user-certs` + `/host-certs` (sign), list/get/revoke SSH certs.
  Migration `002_ssh_ca.*.sql` (`ssh_certificate_authorities`, `ssh_certificates`), repo
  methods, handlers consuming `internal/sshca`.
- **Acceptance:** DB-gated integration test (§11.4.3, `TEST_DATABASE_URL`) against a real
  Postgres proving create-CA → sign → retrieve → verify round-trip. REQUIRES a live Postgres
  for proper TDD (§11.4.43) — deliberately NOT bundled into HT-SSHCA-001 to avoid shipping
  untested pgx/SQL (§11.4.1).

### HT-SSHCA-003 — ssh-proxy consumes SSH certificates for host auth
- **Type:** Feature
- **Status:** Queued
- **Priority:** High
- **What:** ssh-proxy-service authenticates only via long-lived `ssh.ParsePrivateKey`. Wire
  it to request a short-lived user cert from PKI (HT-SSHCA-002) and present it, and to trust
  host certs via `@cert-authority`. Closes the enterprise cert-based-SSH story end-to-end.

### HT-TERM-001 — terminal-service websocket + PTY I/O proxy
- **Type:** Bug/Feature (investigate first)
- **Status:** Queued
- **Priority:** High
- **What:** No `websocket`/`pty` usage found in terminal-service non-test code — it appears
  to be session-metadata CRUD, not the live terminal I/O proxy the SERVICE_REGISTRY §7
  requires. Investigate depth (§11.4.124 before concluding), then implement the WS/PTY
  fan-out (composes with ssh-proxy `wshandler`).

---

## Phase 2 — Test-truth & QA

### HT-TEST-INTEG-001 — integration tests → 25/25 services
- **Type:** Task · **Status:** Queued · **Priority:** High
- **What:** ~10 services lack integration tests; ~13 have shallow repo/server coverage
  (constructor + not-connected negative path only). Add real DB-gated integration tests
  (§11.4.3) proving positive-path CRUD per service.

### HT-QA-001 — docs/qa e2e transcripts → 25/25 features
- **Type:** Task · **Status:** Queued · **Priority:** High
- **What:** Only 3 features carry `docs/qa/<run-id>/` transcripts (§11.4.83 requires one per
  shipped feature). Add real e2e transcripts for the rest.

---

## Phase 3 — Client

### HT-CLIENT-001 — Flutter client real e2e user journeys
- **Type:** Task · **Status:** Queued · **Priority:** Medium
- **What:** 8/11 client test files are `expect(true, isTrue)` no-ops; the sole e2e test is
  boot-only (explicit TODO). Implement real BLoC + widget + e2e tests driving actual user
  journeys (§11.4.143), host-rendered visual proof where UI is asserted (§11.4.170).

---

## Phase 4 — Governance / infra

### HT-INFRA-001 — bootstrap §11.4.93 workable-items SQLite DB
- **Type:** Task · **Status:** Queued · **Priority:** Medium
- **What:** Stand up `docs/workable_items.db` as the single source of truth (tracked in git,
  §11.4.95) + Go tooling for bidirectional md↔db sync; migrate this Markdown into it.

### HT-INFRA-002 — Go module-path standardization
- **Type:** Task · **Status:** Queued · **Priority:** Low
- **What:** Standardize module paths (`digital.vasic.*`, 600+ refs) — deliberately deferred
  per CHANGELOG "Deferred" section; high-churn.

---

## Phase 5 — Operator-blocked backends (external creds/decisions)

### HT-BLOCKED-AI — ai-service cloud-LLM provider keys
- **Type:** Feature · **Status:** Operator-blocked · **Priority:** Medium
- **Operator-Block-Details:** WHAT = cloud-LLM API keys for ai-service's cloud tier.
  WHY = external credentials the agent cannot obtain (§11.4.10). UNBLOCK = operator provides
  keys via gitignored `.env`. Local-tier via self-hosted HelixLLM is the non-blocked path.

### HT-BLOCKED-BILLING — billing-service Stripe keys
- **Type:** Feature · **Status:** Operator-blocked · **Priority:** Medium
- **Operator-Block-Details:** WHAT = Stripe API keys. WHY = external account/credentials.
  UNBLOCK = operator provides Stripe test+live keys.

### HT-BLOCKED-PUSH — notification-service FCM/APNs
- **Type:** Feature · **Status:** Operator-blocked · **Priority:** Low
- **Operator-Block-Details:** WHAT = FCM + APNs push credentials. WHY = external accounts.
  UNBLOCK = operator provides FCM server key + APNs .p8/cert.

### HT-BLOCKED-JWTKEY — auth-service production JWT key management (T15)
- **Type:** Task · **Status:** Operator-blocked · **Priority:** High
- **Operator-Block-Details:** WHAT = production JWT signing-key persistence strategy.
  WHY = design decision (KMS vs mounted-secret) the operator must make. UNBLOCK = operator
  decision. (Ephemeral keys work for dev; production must not rotate keys on restart.)

---

## Phase 6 — Production hardening & deploy validation

### HT-PROD-001 — real infra apply + smoke + SLO/load + security scans
- **Type:** Task · **Status:** Queued · **Priority:** Medium
- **What:** Terraform/K8s/Helm exist but are not validated by a real apply; SLO/soak/chaos
  are labeled "planned"; security scans (Trivy/govulncheck/ZAP) defined but not wired to a
  gate (CI is disabled per §11.4.156 — run via the project's own gate scripts). Validate.
