# 03 — Service Catalog

**Status:** `Complete`  
**Module:** A + B (all 25 services)  
**Authority:** `SERVICE_REGISTRY.md` (single source of truth per CD-3)  

---

> **This section is the canonical service registry for helix_terminator.**  
> Any document that enumerates a divergent service set, count, or naming scheme is a **defect** to be reconciled against this file, not an alternate source of truth.

---

## Canonical Service Table

| # | Name | Domain / Responsibility | Module Path | Primary Datastore | Key Upstream Deps |
|---|------|------------------------|-------------|-------------------|-------------------|
| 1 | API Gateway | Single ingress; JWT validation; rate limiting; upstream routing; circuit breaking; WebSocket proxy | `helixterminator.io/services/gateway` | none (stateless; Redis for rate-limit + JWKS cache) | all 25 downstream services (ingress) |
| 2 | Auth Service | Authentication (password, FIDO2/WebAuthn, TOTP, OIDC, SAML); JWT/refresh/device token issuance; SCIM inbound sync | `helixterminator.io/services/auth` | PostgreSQL `helixterm_auth` | user, vault, pki, notification, audit |
| 3 | User Service | User CRUD, profile, preferences, onboarding state machine, SCIM provisioning endpoint | `helixterminator.io/services/user` | PostgreSQL `helixterm_users` | org, notification, audit |
| 4 | Vault Service | Zero-knowledge encrypted storage for SSH keys, passwords, API tokens, TLS certs, secret notes; sharing + versioning | `helixterminator.io/services/vault` | PostgreSQL `helixterm_vault` | keychain, audit, pki |
| 5 | Host Service | SSH host/group CRUD, health ping, import/export, bastion/jump-host chains, host templates | `helixterminator.io/services/host` | PostgreSQL `helixterm_hosts` | vault, org, audit |
| 6 | SSH Proxy Service | Brokers SSH connections (password/pubkey/certificate auth), container-native sessions, proxy-jump chains, agent forwarding | `helixterminator.io/services/ssh-proxy` | PostgreSQL `helixterm_ssh_proxy` (connection state) | auth, vault, host, terminal, audit, recording, pki, container-bridge |
| 7 | Terminal Session Service | WebSocket terminal I/O proxy, scrollback buffer (Redis), command-boundary detection, collaborative fan-out | `helixterminator.io/services/terminal` | PostgreSQL `helixterm_terminal` | ssh-proxy, collab, recording, ai, audit |
| 8 | SFTP Service | SFTP session/file operations, transfer queue + resume, checksum verification, bidirectional directory sync | `helixterminator.io/services/sftp` | PostgreSQL `helixterm_sftp` | ssh-proxy, vault, audit |
| 9 | Port Forwarding Service | Port-forward rule catalog + lifecycle (local/remote/dynamic/reverse), auto-reconnect, tunnel metrics | `helixterminator.io/services/port-forward` | PostgreSQL `helixterm_port_forward` | ssh-proxy, vault, audit |
| 10 | Snippet Service | Command/script/SQL snippet CRUD, folders/namespaces, parameterization, execution history, versioning | `helixterminator.io/services/snippet` | PostgreSQL `helixterm_snippets` | audit |
| 11 | Keychain Service | Hardware-backed key storage (Secure Enclave / Android Keystore / DPAPI / kernel keyring / HSM), vault key wrap/unwrap, key rotation; gRPC-only | `helixterminator.io/services/keychain` | PostgreSQL `helixterm_keychain` | audit |
| 12 | Workspace Service | Named workspace CRUD (hosts + snippets + vault items + settings), templates, sharing, quick-launch | `helixterminator.io/services/workspace` | PostgreSQL `helixterm_workspaces` | user, org, audit |
| 13 | Collaboration Service | Real-time session sharing (observer/co-pilot/owner roles), CRDT buffer sync, broadcast mode, chat sidebar | `helixterminator.io/services/collab` | PostgreSQL `helixterm_collab` (+ Redis pub/sub) | terminal, user, org, notification |
| 14 | Notification Service | Multi-channel delivery (in-app, email, push, Slack, webhooks), templates, dedup, digests | `helixterminator.io/services/notification` | PostgreSQL `helixterm_notifications` | user, audit |
| 15 | Audit Service | Append-only Merkle-chained audit log, compliance query/export API, SOC 2/ISO 27001/FedRAMP evidence, retention enforcement | `helixterminator.io/services/audit` | PostgreSQL `helixterm_audit` (partitioned by org + month) | — (Kafka-only leaf) |
| 16 | Analytics Service | Session/command/transfer usage aggregation, dashboard data, SLO tracking, Grafana/Prometheus export | `helixterminator.io/services/analytics` | PostgreSQL `helixterm_analytics` (time-series, partitioned by week) | Kafka consumer only |
| 17 | AI/Autocomplete Service | Command autocomplete, output explanation, anomaly detection, runbook generation, incident assist | `helixterminator.io/services/ai` | PostgreSQL `helixterm_ai` (+ Redis suggestion cache) | terminal, user, audit |
| 18 | Session Recording Service | Assembles asciinema-format recordings from Kafka segments, Ed25519 signing, playback API, full-text search, MP4 export | `helixterminator.io/services/recording` | PostgreSQL `helixterm_recordings` (metadata) + S3-compatible object storage | terminal, audit |
| 19 | Certificate Authority / PKI | Issues short-lived SSH certificates (user + host), CA rotation, revocation checked by SSH Proxy | `helixterminator.io/services/pki` | PostgreSQL `helixterm_pki` | vault, audit |
| 20 | Organization/Team Service | Multi-tenant Org → Team → Member hierarchy, RBAC role/permission model, SCIM provisioning, invitations, seat licensing | `helixterminator.io/services/org` | PostgreSQL `helixterm_org` | user, auth, notification, audit |
| 21 | Billing Service | Subscription/seat management, Stripe integration, invoicing, usage metering, trials, dunning | `helixterminator.io/services/billing` | PostgreSQL `helixterm_billing` | org, user, notification, audit |
| 22 | Configuration Service | Centralized feature flags + operational parameters, per-org overrides, runtime config propagation via Kafka, config audit | `helixterminator.io/services/config` | PostgreSQL `helixterm_config` | audit |
| 23 | Health/Monitoring Service | Aggregates `/health/live` + `/health/ready` from all services, unified health dashboard, SLO error-budget calc, alert routing | `helixterminator.io/services/health` | none (aggregates other services) | all services (health-check fan-out) |
| 24 | Container Registry Bridge | Kubernetes cluster registration, pod exec/shell sessions, container log streaming, Docker/Podman host registration | `helixterminator.io/services/container-bridge` | PostgreSQL `helixterm_container_bridge` | vault, org, audit |
| 25 | HelixTrack Integration | OAuth2 link to `helixtrack.ru/core`; ties terminal sessions to HelixTrack issues/sprints; deployment-event sync | `helixterminator.io/services/helixtrack-bridge` | PostgreSQL `helixterm_helixtrack_bridge` | user, org, audit |

---

## Notes on the Table

- Two services (API Gateway, Health/Monitoring) have no dedicated PostgreSQL database — Gateway is explicitly stateless; Health/Monitoring aggregates other services' health rather than owning domain data.
- "Key upstream deps" lists synchronous call-graph dependencies only; Kafka-topic-only (asynchronous) relationships are called out inline where the matrix marks a service as Class C.
- Keychain Service is **gRPC-only** — no REST surface.

---

## Alias Reconciliation

Other documents use different names for some of the same 25 services. Where a name maps confidently to a canonical service above:

| Canonical Name | Alias Seen In | Confidence |
|---------------|---------------|------------|
| API Gateway (`gateway`) | `api-gateway` (doc 10) | High |
| Host Service (`host`) | `host-manager` (doc 10) | High |
| SFTP Service (`sftp`) | `sftp-proxy` (doc 10) | High |
| Workspace Service (`workspace`) | `workspace-svc` (doc 10) | High |
| Configuration Service (`config`) | `config-svc` (doc 10) | High |
| SSH Proxy Service (`ssh-proxy`) | `ssh-proxy` (doc 04) | High — same name |
| Session Recording Service (`recording`) | `session-recorder` (doc 04) | Medium |
| Certificate Authority / PKI (`pki`) | `certificate-service` (doc 04) | Medium |
| SFTP Service (`sftp`) | `file-transfer-service` (doc 04) | Medium |
| Port Forwarding Service (`port-forward`) | `tunnel-service` (doc 04) | Medium |

**Doc 04 names with NO canonical counterpart** (to be reconciled or dropped): `rbac-service`, `scheduler-service`, `inventory-service`, `key-rotation-service`, `compliance-service`, `reporting-service`, `search-service`, `webhook-service`, `approval-workflow-service`, `policy-engine`, `metrics-aggregator`.

---

## Cross-References

- [02 — System Architecture](../02-system-architecture/) — C4 diagrams, resilience matrix, deployment topology
- [04 — API Specification](../04-api-specification/) — REST endpoints and gRPC definitions per service
- [05 — Database Schema](../05-database-schema/) — CREATE TABLE and CREATE INDEX per service database
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — K8s manifests, service ports, Helm charts
- [16 — References](../16-references/) — Full `SERVICE_REGISTRY.md` canonical source

---

*Section 03 — Service Catalog*  
*Consolidated from: SERVICE_REGISTRY.md, 01_core_architecture.md §3*
