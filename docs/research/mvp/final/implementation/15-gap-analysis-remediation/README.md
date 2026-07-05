# 15 — Gap Analysis & Remediation

**Status:** `Complete`  
**Module:** A + B  
**Authority:** `REMEDIATION_REGISTER.md` (253 findings, 12 canonical decisions)

---

## Overview

This section consolidates the master gap analysis from six independent audits (A1–A6) covering all 12 source documents. **253 distinct findings** were identified after cross-report de-duplication. They are organized by action class: Canonical Decisions (CD-1..CD-12), Fix-Now items, and Deep-Work items.

| Severity Bucket | Count |
|-----------------|-------|
| Critical (C) | ~55 |
| Important / High (I/H) | ~100 |
| Minor / Low (M/L) | ~98 |
| Constructive (Improvement + Diagram-Need) | ~45 |

---

## 12 Canonical Decisions (CD-1..CD-12)

These are high-blast-radius conflicts that required a single source of truth to be chosen. All are now locked in `CANONICAL_FACTS.md`.

| CD | Topic | Decision | Blast Radius |
|----|-------|----------|--------------|
| CD-1 | Product identity | **Dual-module:** Module A (SSH/terminal) + Module B (WireGuard broker) | 11, README, AGENTS.md |
| CD-2 | Org / domain | **HelixDevelopment** / **helixterminator.io** | 04, 05, 08, 10, 11, README |
| CD-3 | 25-service list | **Adopt doc 01's SSH-domain set** as canonical; publish ONE registry | 01, 04, 10, 11, 12, README |
| CD-4 | Version pins | **PostgreSQL 17.2, Go 1.25, Kafka 3.9, Redis 8, K8s 1.31, Flutter 3.24, Istio 1.22** | 01, 03, 04, 06, 08, 10, 11, README |
| CD-5 | API gateway port | **443 edge + 8080 internal**; drop 8000 | 04, 11, 12 |
| CD-6 | Primary/DR regions | **us-east-1 primary, eu-west-1 DR** | 04, 12 |
| CD-7 | JWT signing | **EdDSA (Ed25519), iss: auth.helixterminator.io** | 05, 07 |
| CD-8 | RBAC roles | **super_admin, org_admin, team_admin, member, auditor, api_user** | 05, 07 |
| CD-9 | Constitution version | **Pinned e6504c2, helixcode-v1.1.0 line** | 01, 04, 10, 11 |
| CD-10 | Zero-knowledge | **HARD for vault items** (client-side only); SSH password-auth explicitly non-ZK | 02, 05, 07 |
| CD-11 | helix-deps.yaml | **One manifest, slash-path imports** | 01, 04, 10, 11 |
| CD-12 | Test-type count | **12 mandatory test types** (reconcile 17-type claim) | 03, 11, README |

---

## Fix-Now Items (No Decision Needed)

Objectively-correct fixes requiring no product judgment.

### Document Integrity

| Doc | Fix | Evidence |
|-----|-----|----------|
| 02 | Delete duplicated ~1,340-line block (§9→Appendix C) | A5: L6884-8225 |
| 02 | Repair spliced `host_list_page_test.dart` code block | A5: L5351 |
| 02 | Remove second "End of document" marker | A5: L8224 |
| 06 | Split interleaved document (two docs under same section numbers) | A1: L197 vs L812 |
| 01 | Restore TOC order: §4.15-4.20 stranded after §9/§10 | A4: L5341/L6142 |

### Version & Domain Drift

| Doc | Fix | Evidence |
|-----|-----|----------|
| 03 | Replace fabricated image digests with real pinned digests | A3: L1905-1906 |
| 03 | Align Go/Flutter CI versions to CD-4 | A3: L5429/5929/6007 |
| 04 | Remove `:latest` tags from prod Deployments | A6: L577/809/991/1183/1372 |
| 04 | Fix container UID mismatch (65532 vs mandated 65534) | A6: L548 vs L7310 |
| 04 | De-duplicate PrometheusRule groups (ssh.alerts, vault.alerts) | A6: L5389-5436 vs L5572-5607 |
| 04 | Fix `/healthz` grep gate (literal vs `/healthz/live` + `/healthz/ready`) | A6: L8028 vs L601/609 |
| 05 | Fix retention mismatch (sshkey.exported severity vs retention_class) | A2: L3503 vs L3686 |
| 05 | Fix GDPR-erasure no-op (UPDATE silently no-ops) | A2: L3446 vs L4011 |
| 08 | Correct version drift to CD-4 values | A3/A6: L417 |
| 08 | Fix domain drift `helixterminator.io` → `helixterm.io` per CD-2 | A6: L1862/2703/2822 |
| 11 | Remove `:latest` from Appendix A.2 example table | A6: L5198-5222 |
| 11 | Implement or remove `--check <name>` bluff flag | A6: L3448 vs L3486-3512 |
| 12 | Remove orphaned nodes (P1, CONFIG_POD, HEALTH_POD) | A6: L236, L2277-2278 |
| 12 | Add 6 missing pods to K8s layout | A6: (Keychain/Snippet/Workspace/Analytics/HelixTrack/Container) |

### Code Correctness

| Doc | Fix | Evidence |
|-----|-----|----------|
| 07 | Fix migration-tool Go sample (`strings.ToUpper` without import) | A2: L7558-7576 |
| 07 | Fix SSH-config export duplicate `IdentityFile` | A2: L2311/L2318 |
| 07 | Reconcile `reason` field (required vs optional) | A2: L2520 |
| 07 | Collapse doubly-defined `api_keys` table | A2: L1257/L5402 |
| 03 | Replace empty-body stub test `TestAuthService_RateLimit_Login` | A3: L626-628 |
| 03 | Fix flaky-tolerance contradiction (<1% vs 0.05) | A3: L141 vs L5144 |

---

## Deep-Work Items (Substantial Authoring Gaps)

| Size | Item | Target Doc |
|------|------|------------|
| L | Add Row-Level Security to every multi-tenant table | 07 (+ 01 schema) |
| L | Real audit tamper-evidence: external WORM anchoring | 05, 07 |
| L | Item-level vault endpoints + key-rotation/re-wrap | 07 (+ 05 crypto) |
| L | PostgreSQL DR + HA: RPO/RTO, Patroni, PITR, backup cadence | 01, 04 |
| L | Complete Phases 2–5 with AC/Test/DoD | 08 |
| L | Full collaboration spec: latency budget, BLoC/transport/CRDT, wireframes | 09, 02, 06 |
| M | Resolve GDPR-erasure vs audit-immutability end-to-end | 05, 07 |
| M | Secret redaction for session logs/recordings/AI context | 07, 05 |
| M | Injection & blast-radius gating (snippet params, port-forward, broadcast) | 07 |
| M | Break-glass / JIT / SoD controls | 05 |
| M | Redis persistence + cluster hash-tags | 07, 01 |
| M | WebSocket reconnect/resume semantics | 07 |
| M | Cost / FinOps section | 04 |
| M | Schedule orphaned features into phases | 08 |
| M | Risk register + task owners + effort estimates | 08 |
| M | Auto-update mechanism for all 6 platforms | 02 |
| M | Mobile background-execution for SSH/SFTP | 02 |
| M | Conflict-resolution UI + connection-error taxonomy | 02 |
| M | Missing wireframes (Vault, Org/Team, Billing) | 06 |
| M | Device/topology matrix | 03 |
| M | Terminal-rendering performance test methodology | 03 |
| M | Native accessibility testing (VoiceOver/TalkBack/desktop SR) | 03 |
| S | Encrypt SSO IdP OAuth tokens | 07 |
| S | RabbitMQ production path (provision or remove) | 04, 03 |
| S | Missing Pact contracts + Flutter/Dart SBOM/vuln gate | 03 |

---

## Deferred Work (Explicitly Not Done This Increment)

Per `CANONICAL_FACTS.md`:

- Go module-path standardization (`digital.vasic.*` dot-paths, 600+ refs)
- RLS-everywhere (partially resolved in this increment; full enforcement deferred)
- Audit WORM anchoring
- PostgreSQL DR/HA + RPO/RTO authoring
- Item-level vault + key-rotation endpoints
- Full real-time-collaboration spec
- Roadmap Phases 2–5 acceptance criteria
- Client auto-update / mobile background exec
- Device/native-a11y test coverage
- Missing diagrams
- ZK server-keygen removal
- Dual-product scope section (resolved in this increment via SCOPE_AND_MODULES.md)
- Single service registry (resolved in this increment via SERVICE_REGISTRY.md)

---

## Cross-References

- [01 — Executive Summary](../01-executive-summary/) — Product scope, positioning
- [04 — API Specification](../04-api-specification/) — Endpoint gaps
- [05 — Database Schema](../05-database-schema/) — RLS, audit, PII gaps
- [09 — Security — Zero Trust](../09-security-zero-trust/) — Security deep-work items
- [12 — Product Roadmap](../12-product-roadmap/) — Roadmap gaps
- [16 — References](../16-references/) — Full REMEDIATION_REGISTER.md source

---

*Section 15 — Gap Analysis & Remediation*  
*Consolidated from: REMEDIATION_REGISTER.md, CANONICAL_FACTS.md (CD-1..CD-12)*
