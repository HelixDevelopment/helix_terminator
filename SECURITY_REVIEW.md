# HelixTerminator Security Review

> **Review Date:** 2026-07-05  
> **Scope:** Final implementation docs (`09-security-zero-trust/`), security runbooks, infrastructure scaffold, service stubs, and compliance coverage.  
> **Reviewer:** Security Review Agent  
> **Classification:** Internal — Engineering Confidential

---

## 1. Executive Summary

| Domain | Status | Score | Notes |
|--------|--------|-------|-------|
| Security Architecture Docs | **Partial** | 55/100 | Canonical spec exists (4,871 lines) but is **not** the live `README.md`; live doc is a 148-line draft. |
| Zero-Trust Design (SPIFFE/SPIRE, mTLS, Istio) | **Partial** | 60/100 | Comprehensive spec; **zero scaffold code** for SPIRE/Istio in `infrastructure/`. |
| Cryptographic Design (AES-256-GCM, Argon2id, Ed25519) | **Pass** | 85/100 | Well-specified; Argon2id params OK; AES-256-GCM with per-item nonce; Ed25519 for JWT. |
| Authentication (MFA, FIDO2, TOTP, OIDC, SAML) | **Pass** | 80/100 | Complete spec with flows; **scaffold services have zero auth middleware**. |
| RBAC / ABAC | **Partial** | 65/100 | 6-role model + SQL schema + policy engine spec; **no implementation in scaffold**. |
| PKI / CA Design | **Partial** | 70/100 | CA hierarchy, CRL, KRL, CT log specified; **pki-service stub is empty**. |
| Audit Log (Merkle-chained) | **Partial** | 70/100 | Hash chain + WORM anchor spec excellent; **audit-service stub is empty**. |
| Infrastructure Security (NetPol, Falco, container hardening) | **Fail** | 25/100 | Default-deny NetPol present but **no per-service allow policies**; Falco/Trivy/Cosign/Sealed-Secrets are **all TODO stubs**. |
| Compliance (SOC 2, GDPR, HIPAA, FedRAMP) | **Partial** | 60/100 | Mapping tables present; **no evidence artifacts or automated compliance gates**. |
| Security Runbooks | **Fail** | 10/100 | **All 5 runbooks are empty TODO stubs** (certificate-rotation, incident-response, key-rotation, failover, postgres-pitr). |
| Hardcoded Secrets Scan | **Pass** | 95/100 | No hardcoded passwords/tokens found in tracked files. One env-var placeholder (`${SPIRE_DB_PASSWORD}`) in spec only. |
| Scaffold Anti-Patterns | **Fail** | 30/100 | **Every service stub** uses `log.Fatalf`, `TODO` comments, `latest` image tag, `IfNotPresent`, `replicaCount: 1`, missing health/live+ready split. |

### Overall Security Readiness Score: **52 / 100** (Marginal — Not Production-Ready)

> **Verdict:** The **security specification is strong** (enterprise-grade, threat-modeled, cryptographically sound). The **implementation scaffold is dangerously immature** — empty services, empty runbooks, empty security tooling configs, and pervasive anti-patterns that would become production vulnerabilities if deployed as-is.

---

## 2. Pass / Fail per Security Domain

| # | Domain | Result | Rationale |
|---|--------|--------|-----------|
| 1 | Security Architecture Documentation | **PARTIAL** | Canonical 4,871-line spec (`05_security_zero_trust.md`) is comprehensive. The live `09-security-zero-trust/README.md` is a 148-line draft that omits 90 % of the detail. Risk: engineers reference the short draft, miss critical controls. |
| 2 | Zero-Trust (SPIFFE/SPIRE, mTLS, Istio) | **PARTIAL** | Spec defines SPIRE HA topology, SVID rotation, Istio STRICT mTLS, AuthorizationPolicy. **No Helm charts, no SPIRE configs, no Istio operator YAML in `infrastructure/`**. The `infrastructure/helm/helixterm/` chart is an empty umbrella with no subcharts. |
| 3 | Cryptography (AES-256-GCM, Argon2id, Ed25519) | **PASS** | Argon2id parameters (t=3, m=64MB, p=4) meet OWASP. AES-256-GCM with 12-byte random nonce per item. Ed25519 for JWT signing. HKDF-SHA256 for ECDH key derivation. No deprecated algorithms. |
| 4 | Authentication (MFA/FIDO2/TOTP/OIDC/SAML) | **PASS** | All flows documented with sequence diagrams. FIDO2 uses `direct` attestation, `userVerification: required`. TOTP uses 160-bit secrets, ±1 window. OIDC uses `go-oidc` verifier with 5-min skew. SAML rejects IDP-initiated. **Scaffold has no auth middleware wired.** |
| 5 | RBAC / ABAC | **PARTIAL** | 6-role vocabulary, permission matrix, SQL schema, policy evaluation engine, break-glass, JIT elevation, two-person control all specified. **No code in any service stub implements RBAC checks.** |
| 6 | PKI / CA | **PARTIAL** | Root CA offline/HSM, intermediate rotation, SSH cert TTL=8h, KRL, CT log specified. **pki-service stub is empty** (only `main.go` with `TODO`). |
| 7 | Audit Log (Merkle-chained) | **PARTIAL** | Hash chain with SHA-256, PostgreSQL append-only rules, RLS, WORM anchor to S3 Object Lock Compliance mode, SIEM integration, retention classes all specified. **audit-service stub is empty.** |
| 8 | Infrastructure Security | **FAIL** | `network-policy.yaml` has default-deny but **no allow policies**. Falco, Trivy, Cosign, Sealed-Secrets are **4-line TODO files**. No PodSecurityPolicies / Pod Security Standards. No Falco rules. No image-signing policy. |
| 9 | Compliance (SOC 2 / GDPR / HIPAA / FedRAMP) | **PARTIAL** | Control mapping tables present. GDPR crypto-shredding design reconciles immutability with erasure. HIPAA BAA note present. **No automated compliance evidence collection, no DPO contact in runbooks, no FedRAMP control implementation details.** |
| 10 | Security Runbooks | **FAIL** | All 5 runbooks (`certificate-rotation`, `incident-response`, `key-rotation`, `failover-procedure`, `postgres-pitr-restore`) are **empty TODO lists** with no procedures. |
| 11 | Secrets in Code | **PASS** | No hardcoded passwords, API keys, or private keys found in tracked files. `git-secrets` / `.env` patterns are referenced in compliance docs. One env-var placeholder `${SPIRE_DB_PASSWORD}` in the SPIRE spec — acceptable as documentation. |
| 12 | Scaffold Anti-Patterns | **FAIL** | See Section 5. |

---

## 3. Critical Findings (Blockers — Do Not Deploy)

### CRIT-1: All Security Runbooks Are Empty Stubs
- **Location:** `docs/runbooks/*` (5 files)
- **Finding:** Every runbook is a 5–7 line markdown file containing only `## TODO` bullet lists with no actual procedures.
- **Impact:** In a production incident (e.g., CA compromise, credential leak, DB corruption), operators have **no documented response steps**. Incident response time will be measured in hours instead of minutes.
- **Remediation:** Populate each runbook with the step-by-step procedures already defined in `05_security_zero_trust.md` §10 (Incident Response, CA Compromise, Account Takeover). Cross-reference the P0/P1/P2 timelines and add communication templates.

### CRIT-2: Security Infrastructure Tooling Is 100 % Unimplemented
- **Location:** `infrastructure/security/{falco,cosign,sealed-secrets,trivy}/README.md`
- **Finding:** Each file is a 4–6 line stub with `## TODO` and 2–3 unchecked items.
- **Impact:** Without Falco rules, there is **no runtime threat detection**. Without Cosign, images are **not signed**. Without Sealed Secrets, Kubernetes secrets are **not encrypted at rest**. Without Trivy CI integration, vulnerable images ship undetected.
- **Remediation:**
  - Falco: Add rules for unexpected shell in containers, privilege escalation, sensitive file access.
  - Cosign: Add `cosign sign` / `cosign verify` policy and key-ref (KMS/AWS KMS).
  - Sealed Secrets: Add `SealedSecret` templates per service and document key rotation.
  - Trivy: Add `.github/workflows/trivy-scan.yml` (or local hook per Constitution §11.4.75) and fail on `HIGH`/`CRITICAL`.

### CRIT-3: Helm Values Use `latest` Tag and `IfNotPresent` — Forbidden by Constitution
- **Location:** `infrastructure/helm/helixterm/values.yaml:7-8`
- **Finding:** `tag: latest`, `pullPolicy: IfNotPresent`.
- **Impact:** Non-reproducible deployments; stale images may be cached; rollback is undefined. This is explicitly listed as a forbidden anti-pattern in `14-constitution-compliance/README.md` (Anti-Pattern #2).
- **Remediation:** Pin to semver tag + SHA-256 digest (e.g., `tag: "1.0.0@sha256:abc123…"`). Set `pullPolicy: Always` or `IfNotPresent` only when digest is pinned.

### CRIT-4: Default-Deny NetworkPolicy Has No Allow Policies
- **Location:** `infrastructure/kubernetes/base/network-policy.yaml`
- **Finding:** Only `default-deny-all` is defined; comment says `# TODO: add per-service allow policies`.
- **Impact:** If deployed, **all inter-service traffic is blocked** (functional outage), or operators will delete the policy to restore service (security regression).
- **Remediation:** Add explicit `NetworkPolicy` allow rules for each service pair (auth→db, vault→db, gateway→auth, etc.) matching the spec in `05_security_zero_trust.md` §1.7.

---

## 4. High Findings (Must Fix Before Production)

### HIGH-1: Every Service Scaffold Uses `log.Fatalf` on Startup Error
- **Location:** `services/*/cmd/*/main.go` (all 25 services)
- **Finding:** `log.Fatalf("server failed: %v", err)` — panics on listen failure.
- **Impact:** In Kubernetes, `Fatalf` prevents graceful termination hooks from running; no structured logging; no metric emission. Violates Constitution §11.4.10 (structured logging) and HT-010.
- **Remediation:** Replace with `digital.vasic.observability` structured logger, return error to `main`, and exit with `os.Exit(1)` after cleanup.

### HIGH-2: Health Endpoints Do Not Follow Constitution-Mandated Split
- **Location:** `services/*/internal/server/server.go`
- **Finding:** Services expose `/health` and `/ready` but the Constitution (HT-008) mandates `/health/live` and `/health/ready`. The readiness stub returns `{"ready": true}` without checking DB/cache/upstream.
- **Impact:** Kubernetes probes will pass while dependencies are down; rolling updates may break traffic.
- **Remediation:** Rename paths to `/health/live` and `/health/ready`; implement real dependency checks in readiness.

### HIGH-3: Go Module Version Violates Constitution (HT-002)
- **Location:** `services/*/go.mod` (e.g., `auth-service/go.mod:3`)
- **Finding:** `go 1.22` — Constitution requires `go 1.25` or higher.
- **Impact:** Compliance gate failure; potential security issues in older stdlib.
- **Remediation:** Upgrade all `go.mod` files to `go 1.25` (or `1.25.0`) and verify build.

### HIGH-4: No Auth Middleware Wired in Any Service
- **Location:** `services/*/internal/server/server.go`
- **Finding:** `// TODO: configure middleware (logging, recovery, auth, tracing)` — auth is commented out.
- **Impact:** Every service endpoint is **unauthenticated** in the current scaffold. If deployed, anyone with network access can hit internal APIs.
- **Remediation:** Wire JWT validation (Ed25519 JWKS), SPIFFE/SVID extraction, and RBAC enforcement into the server middleware chain per `05_security_zero_trust.md` §2.6 / §6.4.

### HIGH-5: Helm Chart Is Empty Umbrella with No Subcharts
- **Location:** `infrastructure/helm/helixterm/Chart.yaml`, `values.yaml`
- **Finding:** No dependencies, no subcharts, no security contexts, no resource limits, no security probes.
- **Impact:** No reproducible, versioned, security-hardened deployment path.
- **Remediation:** Add per-service subcharts with `securityContext` (runAsNonRoot, readOnlyRootFilesystem, drop ALL capabilities), resource limits, liveness/readiness probes, and network policies.

### HIGH-6: `replicaCount: 1` and `resources: {}` in Production Values
- **Location:** `infrastructure/helm/helixterm/values.yaml:3,17`
- **Impact:** Single point of failure; no resource limits means a single runaway pod can exhaust node resources (DoS vector).
- **Remediation:** Production overlay should set `replicaCount: ≥2` (3 for HA), and concrete `resources.requests` / `resources.limits`.

### HIGH-7: Missing Container Hardening (Security Contexts, PSS)
- **Location:** All Kubernetes manifests
- **Finding:** No `securityContext`, no `PodSecurityPolicy` / `Pod Security Standards` (Restricted), no `seccomp` profiles, no `AppArmor` annotations.
- **Impact:** Containers run as root by default; writable root FS; full capability set.
- **Remediation:** Apply `securityContext` to every deployment: `runAsNonRoot: true`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, `capabilities: { drop: ["ALL"] }`. Enforce PSS `restricted` at the namespace level.

### HIGH-8: No ABAC / RBAC Enforcement in Service Code
- **Location:** All service handlers
- **Finding:** Handlers return static JSON with no authorization checks.
- **Impact:** Any authenticated user (or unauthenticated user, given HIGH-4) can perform any action.
- **Remediation:** Inject the `rbac.Engine` into every handler; call `Evaluate()` before executing business logic.

---

## 5. Medium Findings (Should Fix)

### MED-1: `09-security-zero-trust/README.md` Is a Draft
- **Location:** `docs/research/mvp/final/implementation/09-security-zero-trust/README.md`
- **Finding:** Status is `Draft`; only 148 lines; omits STRIDE table, continuous verification, network micro-segmentation, SSH hardening, incident response.
- **Remediation:** Replace with a condensed but complete reference that links to the canonical spec, or promote the canonical spec into the live directory.

### MED-2: `helix-deps.yaml` Not Verified for Security Dependencies
- **Location:** Root `helix-deps.yaml`
- **Finding:** No evidence that security-critical dependencies (`go-spiffe/v2`, `go-webauthn/webauthn`, `coreos/go-oidc`, `hashicorp/vault/shamir`) are pinned, scanned for CVEs, or have SBOM entries.
- **Remediation:** Add dependency vulnerability scanning (Trivy / Snyk / OSV) to the pre-build gate and pin all security packages to verified hashes.

### MED-3: No Evidence of WORM Anchor Implementation
- **Location:** `05_security_zero_trust.md` §7.4
- **Finding:** WORM anchor design is excellent but **no code or Terraform for S3 Object Lock Compliance mode** exists in `infrastructure/`.
- **Remediation:** Add Terraform for the audit-anchor S3 bucket with `Object Lock = Compliance`, 10-year retention, and IAM policy that denies `s3:PutObjectRetention` / `s3:DeleteObject`.

### MED-4: No Secret Rotation Automation
- **Location:** `docs/runbooks/key-rotation.md`
- **Finding:** Empty stub; no automated rotation for DB credentials, JWT signing keys, or SPIRE intermediate CAs.
- **Remediation:** Implement automated rotation pipelines with `scripts/rotate-secrets.sh` and document the procedure.

### MED-5: Duo Security Integration Code References Hardcoded Timeout
- **Location:** `05_security_zero_trust.md` §2.4 (Duo client)
- **Finding:** `duoapi.SetTimeout(30*time.Second)` — no jitter, no retry, no circuit breaker.
- **Remediation:** Wrap with `digital.vasic.recovery` circuit breaker and add exponential backoff.

### MED-6: No Evidence of Certificate Transparency Log Deployment
- **Location:** `05_security_zero_trust.md` §3.7
- **Finding:** CT log schema defined but no deployment artifact (DB table, consumer, or API) exists.
- **Remediation:** Add CT log table migration and a consumer that writes to it on every `cert.issued` event.

### MED-7: GDPR Erasure Function Is Spec-Only
- **Location:** `05_security_zero_trust.md` §9.2
- **Finding:** `EraseUserData` and `CryptoShredUserPII` are documented but not implemented in any service stub.
- **Remediation:** Implement in `user-service` or `audit-service` with a gRPC endpoint and add integration tests.

### MED-8: Missing `helixterm.io/services/<name>` Module Path in Scaffold
- **Location:** `services/auth-service/go.mod:1`
- **Finding:** Uses `github.com/helixdevelopment/auth-service` instead of canonical `helixterm.io/services/auth-service` per HT-NAME-001.
- **Remediation:** Update all `go.mod` files to canonical module paths and adjust import paths.

---

## 6. Recommendations for Hardening

### Immediate (Pre-Alpha)
1. **Populate all 5 security runbooks** with procedures from the canonical spec.
2. **Implement Falco, Trivy, Cosign, and Sealed Secrets** configs; add them to CI gates.
3. **Fix Helm values:** remove `latest`, pin digests, set `pullPolicy: Always`, add `securityContext`, set `replicaCount ≥ 2`.
4. **Add per-service NetworkPolicy allow rules** so default-deny is functional, not breaking.
5. **Wire auth middleware** (JWT + SPIFFE) into the server scaffold for at least `gateway-service`, `auth-service`, `vault-service`, and `pki-service`.

### Short-Term (Alpha → Beta)
6. **Implement RBAC policy engine** as a shared library (`helixterm.io/pkg/rbac`) and inject into every handler.
7. **Add audit-service consumer** that reads Kafka `audit-events` topic and writes to PostgreSQL with hash-chain verification.
8. **Deploy SPIRE server/agent Helm charts** and validate SVID rotation.
9. **Add WORM anchor Terraform** and a daily `VerifyAgainstAnchors` cron job.
10. **Add container image signing** with Cosign + KMS key; verify in admission controller.

### Long-Term (Production Hardening)
11. **Implement break-glass and JIT elevation** UIs/APIs with two-person control enforcement.
12. **Add automated secret rotation** (DB creds, JWT keys, CA intermediates) with zero-downtime handoff.
13. **Deploy Pod Security Standards (Restricted)** across all namespaces; enforce with Kyverno or OPA Gatekeeper.
14. **Add DDoS simulation tests** (k6 load tests already exist in `test/performance/k6/` — extend with auth-gated endpoints).
15. **Conduct a third-party penetration test** against the running platform before GA.

---

## 7. Appendix: Files Reviewed

| Path | Type | Status |
|------|------|--------|
| `docs/research/mvp/final/implementation/09-security-zero-trust/README.md` | Live security doc | Draft, incomplete |
| `docs/research/mvp/output/docs/markdown/05_security_zero_trust.md` | Canonical security spec | Comprehensive, 4,871 lines |
| `docs/research/mvp/final/implementation/12-guides/ADRs/ADR-005-spiffe-over-vault-identity.md` | ADR | Accepted, well-reasoned |
| `docs/research/mvp/final/implementation/14-constitution-compliance/README.md` | Compliance doc | Complete |
| `docs/runbooks/{certificate-rotation,incident-response,key-rotation,failover-procedure,postgres-pitr-restore}.md` | Runbooks | **All empty** |
| `infrastructure/security/{falco,cosign,sealed-secrets,trivy}/README.md` | Security tooling | **All empty** |
| `infrastructure/kubernetes/base/network-policy.yaml` | NetPol | Default-deny only |
| `infrastructure/helm/helixterm/{Chart.yaml,values.yaml}` | Helm chart | Empty umbrella |
| `services/*/cmd/*/main.go` (all 25) | Service stubs | `log.Fatalf`, `TODO` |
| `services/*/internal/server/server.go` (sampled) | Server stubs | Missing auth, health split |
| `services/*/go.mod` (sampled) | Modules | Go 1.22 (violates HT-002) |
| `docs/research/mvp/output/docs/markdown/11_constitution_compliance.md` | Constitution compliance | Complete |

---

*End of Security Review*
