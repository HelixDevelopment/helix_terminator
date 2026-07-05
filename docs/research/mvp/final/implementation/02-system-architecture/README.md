# 02 — System Architecture

**Status:** `Complete`  
**Module:** A (Secure Terminal Platform) — Module B context is referenced where relevant  
**Authority:** `CANONICAL_FACTS.md` (CD-1, CD-4, CD-5, CD-6) + `SERVICE_REGISTRY.md`  

---

## Architectural Philosophy

HelixTerminator is a full microservices system with strict domain isolation, event-driven state propagation, and zero-trust security enforcement at every layer. Services communicate via three channels:

1. **Synchronous REST/gRPC** (via API Gateway): request/response patterns where the caller needs an immediate result.
2. **Apache Kafka 3.9** (event streaming): durable, ordered, replayable event propagation — audit events, analytics, session telemetry, state change notifications.
3. **RabbitMQ** (command bus): work-queue patterns where a producer dispatches a command and expects exactly-once execution — SSH connection commands, SFTP transfer commands, notification delivery.

This three-channel model provides clear semantic separation: Kafka for facts that have already happened (events), RabbitMQ for instructions that must happen exactly once (commands), REST/gRPC for interrogations (queries) and mutations requiring transactional semantics.

---

## C4 Architecture Diagrams

### Level 1 — System Context

Depicts the platform's outer trust boundary: human personas and automated actors that call it, and the external systems it depends on. Module A scope.

> **Diagram reference:** `docs/research/mvp/output/diagrams/mermaid/01_c4_context.mmd`  
> **Rendered formats:** svg, png, pdf, jpeg, html

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
> **Rendered formats:** svg, png, pdf, jpeg, html

### Level 3 — Component Diagrams

Per-service component breakdowns are documented in [03 — Service Catalog](../03-service-catalog/).

---

## Communication Channels

| Channel | Technology | Purpose | Semantics |
|---------|-----------|---------|-----------|
| REST | Gin Gonic / HTTP 2 | External clients → API Gateway | Request/response |
| gRPC | Protocol Buffers | Internal service-to-service | Low-latency, strongly-typed |
| Events | Apache Kafka 3.9 | Audit, analytics, telemetry | Fire-and-forget, durable, ordered |
| Commands | RabbitMQ | SSH connect, SFTP transfer, notifications | Exactly-once, work-queue |
| Cache | Redis 8 Cluster | Sessions, rate-limit, JWKS | Key-value, TTL |

---

## Resilience Matrix (Per-Service)

Source: `01_core_architecture.md` §2.8

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

**API Gateway ports:** `443` (edge, TLS) and `8080` (internal). Drop `8000`.

> **Diagram reference:** `docs/research/mvp/output/diagrams/mermaid/26_kubernetes_deployment.mmd`  
> **Rendered formats:** svg, png, pdf, jpeg, html

---

## Cross-References

- [03 — Service Catalog](../03-service-catalog/) — Canonical 25 services with full responsibilities
- [04 — API Specification](../04-api-specification/) — REST + gRPC endpoint definitions
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — K8s manifests, Helm, Terraform, DR
- [09 — Security — Zero Trust](../09-security-zero-trust/) — SPIFFE/SPIRE, mTLS, RBAC, PKI
- [12 — Mermaid Diagrams Source](../16-references/) — All 30 Mermaid diagram sources

---

## Diagrams in This Section

| Diagram | Source | Formats |
|---------|--------|---------|
| C4 System Context | `diagrams/mermaid/01_c4_context.mmd` | mmd, svg, png, pdf, jpeg, html |
| C4 Container | `diagrams/mermaid/02_c4_container.mmd` | mmd, svg, png, pdf, jpeg, html |
| Microservices Overview | `diagrams/mermaid/03_microservices.mmd` | mmd, svg, png, pdf, jpeg, html |
| Dependency Graph | `diagrams/mermaid/04_dependency_graph.mmd` | mmd, svg, png, pdf, jpeg, html |
| Data Flow | `diagrams/mermaid/05_data_flow.mmd` | mmd, svg, png, pdf, jpeg, html |
| Kubernetes Deployment | `diagrams/mermaid/26_kubernetes_deployment.mmd` | mmd, svg, png, pdf, jpeg, html |
| Network Stack | `diagrams/mermaid/27_network_stack.mmd` | mmd, svg, png, pdf, jpeg, html |
| Multi-Region DR | `diagrams/mermaid/28_multi_region_dr.mmd` | mmd, svg, png, pdf, jpeg, html |
| System Architecture (Draw.io) | `diagrams/drawio/01_system_architecture.drawio` | drawio, svg, png, pdf, jpeg, html |
| Kubernetes Deployment (Draw.io) | `diagrams/drawio/04_kubernetes_deployment.drawio` | drawio, svg, png, pdf, jpeg, html |
| Zero-Trust Network (Draw.io) | `diagrams/drawio/05_zero_trust_network.drawio` | drawio, svg, png, pdf, jpeg, html |

---

*Section 02 — System Architecture*  
*Consolidated from: 01_core_architecture.md §2, CANONICAL_FACTS.md (CD-4, CD-5, CD-6)*
