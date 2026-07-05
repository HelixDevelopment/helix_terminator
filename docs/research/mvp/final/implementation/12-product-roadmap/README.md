# 12 — Product Roadmap

**Status:** `Draft`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-1, CD-4)

---

## Overview

HelixTerminator development is organized into 5 phases from Jan 2025 to Feb 2026 GA. Each phase delivers incremental value with defined use cases, edge cases, and performance benchmarks.

---

## 5 Development Phases

### Phase 1 — Foundation (Jan 2025 – Mar 2025)

**Theme:** Core platform, auth, vault, SSH

| Deliverable | Status |
|-------------|--------|
| Auth Service (register, login, MFA, SSO) | Complete |
| User Service (profile, preferences) | Complete |
| Vault Service (E2E encryption, sync) | Complete |
| Host Service (CRUD, groups, import) | Complete |
| SSH Proxy Service (password, key, certificate) | Complete |
| Terminal Session Service (WebSocket, scrollback) | Complete |
| Flutter client (macOS, Windows, Linux) | Complete |
| API Gateway (routing, rate limiting) | Complete |
| PostgreSQL schemas (auth, user, vault, host, session) | Complete |
| Kubernetes deployment (dev, staging) | Complete |

**Use Cases:** UC-001 through UC-010 (individual SSH workflow)

### Phase 2 — Team Collaboration (Apr 2025 – May 2025)

**Theme:** Real-time collaboration, team features

| Deliverable | Status |
|-------------|--------|
| Collaboration Service (observer, co-pilot, owner) | In Progress |
| Organization/Team Service (multi-tenant) | In Progress |
| Workspace Service (layouts, templates) | In Progress |
| SFTP Service (file manager, sync) | In Progress |
| Session Recording Service (asciinema, signing) | In Progress |
| Flutter client (iOS, Android) | In Progress |

**Use Cases:** UC-011 through UC-020 (team SSH workflow)

### Phase 3 — Enterprise & AI (Jun 2025 – Aug 2025)

**Theme:** Enterprise features, AI augmentation, compliance

| Deliverable | Status |
|-------------|--------|
| AI/Autocomplete Service (command completion, explanation) | Planned |
| Audit Service (Merkle chain, compliance exports) | Planned |
| PKI Service (short-lived certificates, CA rotation) | Planned |
| RBAC (custom roles, resource-level permissions) | Planned |
| SCIM/SAML enterprise SSO | Planned |
| Compliance dashboard (SOC 2, ISO 27001) | Planned |

**Use Cases:** UC-021 through UC-035 (enterprise workflow)

### Phase 4 — Scale & Operations (Sep 2025 – Nov 2025)

**Theme:** Performance, reliability, multi-region

| Deliverable | Status |
|-------------|--------|
| Multi-region deployment (us-east-1 + eu-west-1) | Planned |
| Disaster recovery runbook automation | Planned |
| Auto-scaling (HPA, KEDA) | Planned |
| Cost optimization (FinOps) | Planned |
| Advanced analytics (SLO tracking, anomaly detection) | Planned |
| Container Bridge (K8s pod exec) | Planned |

**Use Cases:** UC-036 through UC-045 (operations workflow)

### Phase 5 — GA & Ecosystem (Dec 2025 – Feb 2026)

**Theme:** General availability, ecosystem, marketplace

| Deliverable | Status |
|-------------|--------|
| Self-hosted deployment option | Planned |
| API marketplace (third-party integrations) | Planned |
| HelixTrack integration (issue linking, sprint sync) | Planned |
| Plugin system (terminal extensions) | Planned |
| Mobile background execution | Planned |
| Client auto-update mechanism | Planned |

**Use Cases:** UC-046 through UC-050 (ecosystem workflow)

---

## Use Cases (50 Total)

| ID | Use Case | Phase |
|----|----------|-------|
| UC-001 | Register new account | 1 |
| UC-002 | Log in with password + MFA | 1 |
| UC-003 | Add SSH host with key auth | 1 |
| UC-004 | Open terminal session | 1 |
| UC-005 | Execute command on remote host | 1 |
| UC-006 | Create vault and add SSH key | 1 |
| UC-007 | Sync vault across devices | 1 |
| UC-008 | Import hosts from SSH config | 1 |
| UC-009 | Export session to snippet | 1 |
| UC-010 | Configure terminal preferences | 1 |
| UC-011 | Create organization | 2 |
| UC-012 | Invite team member | 2 |
| UC-013 | Share terminal session (observer) | 2 |
| UC-014 | Share terminal session (co-pilot) | 2 |
| UC-015 | Create workspace with multiple panes | 2 |
| UC-016 | Transfer file via SFTP | 2 |
| UC-017 | Record session for compliance | 2 |
| UC-018 | Search command history | 2 |
| UC-019 | Execute snippet on multiple hosts | 2 |
| UC-020 | Configure team host access | 2 |
| UC-021 | Enable enterprise SSO (SAML) | 3 |
| UC-022 | Enforce MFA for all org members | 3 |
| UC-023 | Issue short-lived SSH certificate | 3 |
| UC-024 | AI command completion | 3 |
| UC-025 | AI command explanation | 3 |
| UC-026 | Export audit log for compliance | 3 |
| UC-027 | Create custom RBAC role | 3 |
| UC-028 | SCIM provisioning from IdP | 3 |
| UC-029 | Anomaly detection alert | 3 |
| UC-030 | Runbook generation | 3 |
| UC-031 | Vault key rotation | 3 |
| UC-032 | Session recording playback | 3 |
| UC-033 | Compliance dashboard review | 3 |
| UC-034 | Bulk host import from CSV | 3 |
| UC-035 | Configure IP allowlist | 3 |
| UC-036 | Multi-region failover | 4 |
| UC-037 | Auto-scale SSH Proxy under load | 4 |
| UC-038 | Cost report by service | 4 |
| UC-039 | SLO breach alert | 4 |
| UC-040 | Kubernetes pod exec | 4 |
| UC-041 | Database PITR recovery | 4 |
| UC-042 | Certificate CA rotation | 4 |
| UC-043 | GDPR data export | 4 |
| UC-044 | GDPR data erasure | 4 |
| UC-045 | Performance regression detection | 4 |
| UC-046 | Self-hosted deployment | 5 |
| UC-047 | Third-party plugin installation | 5 |
| UC-048 | HelixTrack issue linking | 5 |
| UC-049 | Mobile background SSH session | 5 |
| UC-050 | Client auto-update | 5 |

---

## Edge Cases (41 Total)

Selected critical edge cases:

| ID | Edge Case | Mitigation |
|----|-----------|------------|
| EC-001 | Network partition during SSH handshake | Retry with exponential backoff, timeout at 30s |
| EC-002 | Vault password forgotten | Recovery via backup codes + email verification |
| EC-003 | MFA device lost | Admin override + identity verification |
| EC-004 | Certificate expiry mid-session | Pre-emptive renewal at 80% TTL |
| EC-005 | Kafka consumer lag during audit spike | Parallel consumers, backpressure |
| EC-006 | Redis node failure | Cluster failover, stale reads acceptable |
| EC-007 | Pod eviction during active session | Graceful handoff, reconnect WebSocket |
| EC-008 | Malicious SSH command injection | Input sanitization, RBAC command whitelist |
| EC-009 | Large file transfer (>10GB) | Chunked upload, resume, checksum per chunk |
| EC-010 | Collaborative session with 100+ observers | Broadcast mode, read-only fan-out |

---

## Performance Benchmarks (60+)

| Phase | Benchmark | Target |
|-------|-----------|--------|
| 1 | SSH connection establish | < 500ms |
| 1 | Terminal keystroke latency | < 16ms |
| 2 | Session share propagation | < 100ms |
| 2 | SFTP file list (1000 files) | < 200ms |
| 3 | AI suggestion latency | < 500ms |
| 3 | Audit event ingestion | < 50ms |
| 4 | Multi-region failover | < 15 minutes RTO |
| 4 | Auto-scale response | < 2 minutes |
| 5 | Mobile background reconnect | < 5s |
| 5 | Client auto-update download | < 30s |

> **DEFERRED:** Phases 2–5 acceptance criteria, test requirements, and definition-of-done are thin or missing in source doc 08. Risk register, task owners, and effort estimates do not exist below whole-doc level.

---

## Cross-References

- [01 — Executive Summary](../01-executive-summary/) — Pricing tiers, target audience
- [03 — Service Catalog](../03-service-catalog/) — 25 services mapped to phases
- [11 — Performance Analysis](../11-performance-analysis/) — SLOs, load tests, benchmarks
- [15 — Gap Analysis](../15-gap-analysis-remediation/) — Roadmap gaps (orphaned features, thin AC)

---

*Section 12 — Product Roadmap*  
*Consolidated from: 08_product_roadmap_features.md, CANONICAL_FACTS.md (CD-1, CD-4)*
