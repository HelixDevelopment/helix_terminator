# HelixTerminator — Architecture Overview

**Version:** 1.0.0  
**Status:** Complete  
**Date:** 2026-07-05  
**Authority:** `CANONICAL_FACTS.md` (CD-1, CD-4, CD-5, CD-6) + `SERVICE_REGISTRY.md`

---

## System Context

HelixTerminator is a **25-service Go microservices architecture** with a **Flutter cross-platform client**, designed for enterprise-grade remote infrastructure management. The platform operates across two modules:

- **Module A — Secure Terminal Platform**: SSH / SFTP / vault / terminal / real-time collaboration
- **Module B — Zero-Trust Connection Broker**: WireGuard / VPN / connection-broker

---

## High-Level Architecture

### Service Mesh (25 Microservices)

| Stream | Services | Purpose |
|--------|----------|---------|
| **A — Core** | Gateway, Auth, User, Vault, Host, SSH-Proxy, Terminal | Authentication, host management, terminal access |
| **B — Workspace** | Keychain, Workspace, Config, PKI, Billing | SSH keys, workspaces, configuration, billing |
| **C — Organization** | Org, Notification, Audit, Health, AI | Teams, notifications, audit logs, health checks |
| **D — Integration** | Collaboration, Container-Bridge, HelixTrack-Bridge, Port-Forward | Real-time collaboration, container management |
| **E — Operations** | SFTP, Recording, Snippet, Analytics | File transfers, session recording, snippets, analytics |

### Three-Channel Communication Model

Services communicate via three channels with clear semantic separation:

1. **Synchronous REST/gRPC** (via API Gateway): request/response patterns where the caller needs an immediate result.
2. **Apache Kafka 3.9** (event streaming): durable, ordered, replayable event propagation — audit events, analytics, session telemetry, state change notifications.
3. **RabbitMQ** (command bus): work-queue patterns where a producer dispatches a command and expects exactly-once execution — SSH connection commands, SFTP transfer commands, notification delivery.

| Channel | Technology | Purpose | Semantics |
|---------|-----------|---------|-----------|
| REST | Gin Gonic / HTTP 2 | External clients → API Gateway | Request/response |
| gRPC | Protocol Buffers | Internal service-to-service | Low-latency, strongly-typed |
| Events | Apache Kafka 3.9 | Audit, analytics, telemetry | Fire-and-forget, durable, ordered |
| Commands | RabbitMQ | SSH connect, SFTP transfer, notifications | Exactly-once, work-queue |
| Cache | Redis 8 Cluster | Sessions, rate-limit, JWKS | Key-value, TTL |

---

## C4 Architecture

### Level 1 — System Context

Depicts the platform's outer trust boundary: human personas and automated actors that call it, and the external systems it depends on.

**Actors:**
- Individual Developer (`member` role)
- Engineering Team Member (`member` role)
- Org / Team Admin (`org_admin` / `team_admin`)
- Compliance Auditor (`auditor` role)
- Platform Super Admin (`super_admin`)
- CI/CD Pipeline (`api_user` service account)

**External Systems:**
- Identity Provider (customer SAML 2.0 / OIDC SSO + SCIM 2.0)
- SSH/SFTP Target Hosts (customer-owned Linux/Unix servers, cloud VMs, K8s pods)
- Object Storage (S3-compatible: session recordings, vault backups, WAL archives)
- Notification Channels (SMTP relay, Slack, PagerDuty, generic webhook)
- Payments Processor (subscription billing, invoicing, usage metering)
- HelixTrack (external work-item tracking system)
- Container Registries (Docker Hub / Harbor / ECR)

### Level 2 — Container Diagram

All 25 microservices in 5 domain swimlanes, with internal and external dependencies.

> **Diagram reference:** `docs/research/mvp/output/diagrams/drawio/01_system_architecture.drawio`

### Level 3 — Component Diagrams

Per-service component breakdowns are documented in the Service Catalog (`backend/README.md`).

---

## Resilience Matrix (Per-Service)

| Service | Failure Class | Upstream Dependencies | Recovery Strategy |
|---------|--------------|----------------------|-------------------|
| API Gateway | A (critical) | all 25 downstream | Circuit breaker + fallback; health-based routing |
| Auth Service | A | user, vault, pki, notification, audit | Retry with exponential backoff; JWKS cache in Redis |
| User Service | B | org, notification, audit | Degraded mode: read-only cache; queue writes |
| Vault Service | A | keychain, audit, pki | HSM fallback; offline key unwrap capability |
| Host Service | B | vault, org, audit | Cached host list; stale reads acceptable |
| SSH Proxy | A | auth, vault, host, terminal, audit, recording, pki, container-bridge | Connection pool draining; graceful session handoff |
| Terminal | B | ssh-proxy, collab, recording, ai, audit | Buffer flush to Kafka; reconnect WebSocket |
| SFTP | B | ssh-proxy, vault, audit | Transfer queue resume; checksum re-verify |
| Port Forward | C | ssh-proxy, vault, audit | Auto-reconnect with exponential backoff |
| Snippet | C | audit | Read-only fallback; local cache |
| Keychain | A | audit | HSM-backed; offline capability |
| Workspace | C | user, org, audit | Cached templates; deferred sync |
| Collaboration | B | terminal, user, org, notification | CRDT sync; eventual consistency |
| Notification | C | user, audit | Queue-based; retry with backoff |
| Audit | A | — (Kafka-only leaf) | WAL fsync every 100ms; Merkle chain checkpoint |
| Analytics | C | Kafka consumer only | Time-series buffer; batch flush |
| AI | C | terminal, user, audit | Model cache; degraded suggestion mode |
| Recording | B | terminal, audit | Segment assembly from Kafka; S3 multipart |
| PKI | A | vault, audit | CA rotation; revocation list cache |
| Org/Team | B | user, auth, notification, audit | Read replica; stale reads acceptable |
| Billing | C | org, user, notification, audit | Stripe webhook replay; idempotency keys |
| Config | C | audit | Local config file fallback |
| Health | C | all services (fan-out) | Cached health; stale aggregation OK |
| Container Bridge | B | vault, org, audit | K8s API retry; pod state cache |
| HelixTrack Bridge | C | user, org, audit | OAuth2 token refresh; queue sync |

**Failure Class Definitions:**
- **Class A:** Service failure is platform-wide outage. Immediate paging, 99.99% SLO.
- **Class B:** Service failure degrades functionality but platform remains usable. 99.9% SLO.
- **Class C:** Service failure is isolated to one feature. 99.5% SLO.

---

## Deployment Topology

| Region | Role | Services |
|--------|------|----------|
| `us-east-1` | Primary | All 25 services + databases + Kafka + Redis |
| `eu-west-1` | DR | Hot standby; async replication from primary |

**API Gateway ports:** `443` (edge, TLS) and `8080` (internal).

---

## Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend | Go | 1.25 |
| Web Framework | Gin | 1.10.0 |
| Database | PostgreSQL | 17.2 |
| DB Driver | pgx/v5 | 5.6.0 |
| Cache | Redis | 8.0.0 |
| Messaging | Kafka | 3.9 |
| Frontend | Flutter | 3.24.0 |
| State Management | flutter_bloc | 8.1.6 |
| Container Runtime | Docker + distroless | latest |
| Orchestration | Kubernetes (EKS) | 1.31 |
| IaC | Terraform | 1.9.0 |
| Package Manager | Helm | 3.15.0 |
| Service Mesh | Istio | 1.22 |
| Security | SPIFFE/SPIRE, mTLS | — |

---

## Cross-References

- `backend/README.md` — Canonical 25 services with full responsibilities
- `api/README.md` — REST + gRPC endpoint definitions
- `infrastructure/README.md` — K8s manifests, Helm, Terraform, DR
- `security/README.md` — SPIFFE/SPIRE, mTLS, RBAC, PKI
- `architecture/` — C4 diagrams, 3-channel model, resilience matrix

---

*HelixTerminator Architecture Overview*  
*Consolidated from: 02-system-architecture/README.md, CANONICAL_FACTS.md (CD-4, CD-5, CD-6)*
