# helix_terminator MVP — Canonical Service Registry

**Revision:** 1 · **Date:** 2026-07-05
**Authority:** `docs/research/mvp/output/CANONICAL_FACTS.md` (CD-3: "Adopt doc 01's
SSH-domain service set as canonical. Publish ONE service registry; other docs
reference it rather than re-enumerating divergently.")
**Org / domain identity:** per CANONICAL_FACTS CD-2 — GitHub org `HelixDevelopment`,
primary domain `helixterminator.io`. Module paths below use the canonical
`helixterminator.io/services/<name>` form; the source document (`01_core_architecture.md`)
still writes them as `helixterm.io/services/<name>` — that is the pre-CD-2 domain string,
reconciled here per CD-2, not a different service set.

**This file is the single source of truth for the helix_terminator microservice set.**
`01_core_architecture.md`, `04_devops_infrastructure.md`, `10_submodule_integration.md`,
and `11_...` (or any other spec document) MUST reference this registry rather than
re-enumerating the service list themselves. Any document that enumerates a divergent
service set, count, or naming scheme is a **defect** to be reconciled against this file,
not an alternate source of truth (see "Reconciliation note" at the end of this document).

## Actual service count

The canonical set enumerated in `01_core_architecture.md` §3 ("Complete Microservices
Catalog", §3.2–§3.26) contains **25 services**. This matches the "25 downstream services"
/ "all 25 services" figure asserted elsewhere in doc 01 (§2.8 resilience matrix, §2.9
dependency graph) and in doc 10's own restatement of the CD-3 set (§1.2). The count is
reported here as a verified fact, not forced to match any prior claim — 25 is what doc 01
actually enumerates.

## Canonical service table

Source for columns 2–3: `01_core_architecture.md` §3.2–§3.26 (Responsibilities +
Module/Port/Database fields). Source for column 5 (Key upstream deps): `01_core_architecture.md`
§2.8 "Full Per-Service Failure-Mode & Resilience Matrix" (Upstream Dependencies column,
itself sourced from the §2.9 dependency graph).

| # | Name (doc 01 §) | Domain / Responsibility | Owning module (org `HelixDevelopment`) | Primary datastore | Key upstream deps |
|---|---|---|---|---|---|
| 1 | API Gateway (§3.2) | Single ingress for all client traffic; JWT validation; per-user/IP rate limiting; upstream routing; circuit breaking; WebSocket upgrade proxy | `helixterminator.io/services/gateway` | none (stateless; Redis used only for rate-limit + JWKS cache) | all 25 downstream services (ingress) |
| 2 | Auth Service (§3.3) | Authentication (password, FIDO2/WebAuthn, TOTP, OIDC, SAML); JWT/refresh/device token issuance; SCIM inbound sync | `helixterminator.io/services/auth` | PostgreSQL `helixterm_auth` | user, vault, pki, notification, audit |
| 3 | User Service (§3.4) | User CRUD, profile, preferences, onboarding state machine, SCIM provisioning endpoint | `helixterminator.io/services/user` | PostgreSQL `helixterm_users` | org, notification, audit |
| 4 | Vault Service (§3.5) | Zero-knowledge encrypted storage for SSH keys, passwords, API tokens, TLS certs, secret notes; sharing + versioning | `helixterminator.io/services/vault` | PostgreSQL `helixterm_vault` | keychain, audit, pki |
| 5 | Host Service (§3.6) | SSH host/group CRUD, health ping, import/export, bastion/jump-host chains, host templates | `helixterminator.io/services/host` | PostgreSQL `helixterm_hosts` | vault, org, audit |
| 6 | SSH Proxy Service (§3.7) | Brokers SSH connections (password/pubkey/certificate auth), container-native sessions, proxy-jump chains, agent forwarding | `helixterminator.io/services/ssh-proxy` | PostgreSQL `helixterm_ssh_proxy` (connection state only) | auth, vault, host, terminal, audit, recording, pki, container-bridge |
| 7 | Terminal Session Service (§3.8) | WebSocket terminal I/O proxy, scrollback buffer (Redis), command-boundary detection, collaborative fan-out | `helixterminator.io/services/terminal` | PostgreSQL `helixterm_terminal` | ssh-proxy, collab, recording, ai, audit |
| 8 | SFTP Service (§3.9) | SFTP session/file operations, transfer queue + resume, checksum verification, bidirectional directory sync | `helixterminator.io/services/sftp` | PostgreSQL `helixterm_sftp` | ssh-proxy, vault, audit |
| 9 | Port Forwarding Service (§3.10) | Port-forward rule catalog + lifecycle (local/remote/dynamic/reverse), auto-reconnect, tunnel metrics | `helixterminator.io/services/port-forward` | PostgreSQL `helixterm_port_forward` | ssh-proxy, vault, audit |
| 10 | Snippet Service (§3.11) | Command/script/SQL snippet CRUD, folders/namespaces, parameterization, execution history, versioning | `helixterminator.io/services/snippet` | PostgreSQL `helixterm_snippets` | audit |
| 11 | Keychain Service (§3.12) | Hardware-backed key storage (Secure Enclave / Android Keystore / DPAPI / kernel keyring / HSM), vault key wrap/unwrap, key rotation; gRPC-only, no REST | `helixterminator.io/services/keychain` | PostgreSQL `helixterm_keychain` | audit |
| 12 | Workspace Service (§3.13) | Named workspace CRUD (hosts + snippets + vault items + settings), templates, sharing, quick-launch | `helixterminator.io/services/workspace` | PostgreSQL `helixterm_workspaces` | user, org, audit |
| 13 | Collaboration Service (§3.14) | Real-time session sharing (observer/co-pilot/owner roles), CRDT buffer sync, broadcast mode, chat sidebar | `helixterminator.io/services/collab` | PostgreSQL `helixterm_collab` (+ Redis pub/sub) | terminal, user, org, notification |
| 14 | Notification Service (§3.15) | Multi-channel notification delivery (in-app, email, push, Slack, webhooks), templates, dedup, digests | `helixterminator.io/services/notification` | PostgreSQL `helixterm_notifications` | user, audit |
| 15 | Audit Service (§3.16) | Append-only Merkle-chained audit log, compliance query/export API, SOC 2/ISO 27001/FedRAMP evidence, retention enforcement | `helixterminator.io/services/audit` | PostgreSQL `helixterm_audit` (partitioned by org + month) | — (Kafka-only leaf; consumes nothing synchronously) |
| 16 | Analytics Service (§3.17) | Session/command/transfer usage aggregation, dashboard data, SLO tracking, Grafana/Prometheus export | `helixterminator.io/services/analytics` | PostgreSQL `helixterm_analytics` (time-series, partitioned by week) | Kafka consumer only (no synchronous upstream) |
| 17 | AI/Autocomplete Service (§3.18) | Command autocomplete, output explanation, anomaly detection, runbook generation, incident assist | `helixterminator.io/services/ai` | PostgreSQL `helixterm_ai` (+ Redis suggestion cache) | terminal, user, audit |
| 18 | Session Recording Service (§3.19) | Assembles asciinema-format recordings from Kafka segments, Ed25519 signing, playback API, full-text search, MP4 export | `helixterminator.io/services/recording` | PostgreSQL `helixterm_recordings` (metadata) + S3-compatible object storage | terminal, audit |
| 19 | Certificate Authority Service / PKI (§3.20) | Issues short-lived SSH certificates (user + host), CA rotation, revocation checked by SSH Proxy | `helixterminator.io/services/pki` | PostgreSQL `helixterm_pki` | vault, audit |
| 20 | Organization/Team Service (§3.21) | Multi-tenant Org → Team → Member hierarchy, RBAC role/permission model, SCIM provisioning, invitations, seat licensing | `helixterminator.io/services/org` | PostgreSQL `helixterm_org` | user, auth, notification, audit |
| 21 | Billing Service (§3.22) | Subscription/seat management, Stripe integration, invoicing, usage metering, trials, dunning | `helixterminator.io/services/billing` | PostgreSQL `helixterm_billing` | org, user, notification, audit |
| 22 | Configuration Service (§3.23) | Centralized feature flags + operational parameters, per-org overrides, runtime config propagation via Kafka, config audit | `helixterminator.io/services/config` | PostgreSQL `helixterm_config` | audit |
| 23 | Health/Monitoring Service (§3.24) | Aggregates `/health/live` + `/health/ready` from all services, unified health dashboard, SLO error-budget calc, alert routing | `helixterminator.io/services/health` | none (aggregates other services; no dedicated database documented) | all services (health-check fan-out) |
| 24 | Container Registry Bridge (§3.25) | Kubernetes cluster registration, pod exec/shell sessions, container log streaming, Docker/Podman host registration | `helixterminator.io/services/container-bridge` | PostgreSQL `helixterm_container_bridge` | vault, org, audit |
| 25 | HelixTrack Integration Service (§3.26) | OAuth2 link to `helixtrack.ru/core`; ties terminal sessions to HelixTrack issues/sprints; deployment-event sync | `helixterminator.io/services/helixtrack-bridge` | PostgreSQL `helixterm_helixtrack_bridge` | user, org, audit |

Notes on the table:
- Two services (API Gateway, Health/Monitoring) have no dedicated PostgreSQL database in
  doc 01 — Gateway is explicitly stateless, Health/Monitoring aggregates other services'
  health rather than owning domain data. This is a documented fact, not an omission.
- "Key upstream deps" lists synchronous call-graph dependencies only, per the §2.8
  resilience matrix; Kafka-topic-only (asynchronous, fire-and-forget) relationships are
  called out inline where the matrix marks a service as Class C.

## Alias reconciliation

Other documents in this spec set use different names for some of the same 25 services
(the exact divergence CD-3 exists to close). Where a name seen elsewhere maps confidently
to a canonical service above, it is listed here. Names are grouped by source document.

### `10_submodule_integration.md` — pre-convergence code-sample names

Doc 10 §1.2 explicitly restates the CD-3 canonical 25-service list (correctly matching this
registry), but flags that its own later code samples (§2–§17) still use pre-convergence
names it has not yet propagated through. Mapping those flagged names to the canonical set:

| Canonical name | Alias seen in doc 10 | Confidence |
|---|---|---|
| API Gateway (`gateway`) | `api-gateway` | High — same service, kebab-cased differently |
| Host Service (`host`) | `host-manager` | High |
| SFTP Service (`sftp`) | `sftp-proxy` | High |
| Workspace Service (`workspace`) | `workspace-svc` | High |
| Configuration Service (`config`) | `config-svc` | High |
| Organization/Team Service (`org`) | `team` | Medium — doc 01 names this service "Organization/**Team**" so `team` is plausibly a shorthand, not a separate service |
| Terminal Session Service (`terminal`) | `session` | Low — ambiguous; could also be read as a generic "session" concept rather than this specific service. Flagged for operator/author confirmation, not asserted as fact. |
| Vault Service (`vault`) | `secret-manager` | Low — ambiguous with Keychain Service (§3.12), which also manages key material. Flagged for confirmation. |
| User Service (`user`) | `challenge`/`onboarding` | Low — doc 01 §3.4 lists an "onboarding workflow state machine" under User Service, and `digital.vasic.challenges` is scoped to "AI Service, User Service" per doc 10 §1.1, but no doc text confirms `challenge`/`onboarding` names a piece of the User Service specifically rather than a separate flow. Flagged for confirmation. |

### `04_devops_infrastructure.md` — divergent `<name>-service` directory layout

Doc 04 asserts its own "25 backend microservices" under `services/` with a naming scheme
(`<name>-service`) that only partially overlaps the canonical set. Confident mappings:

| Canonical name | Alias seen in doc 04 | Confidence |
|---|---|---|
| API Gateway (`gateway`) | `gateway` | High — same name |
| Auth Service (`auth`) | `auth-service` | High |
| Vault Service (`vault`) | `vault-service` | High |
| User Service (`user`) | `user-service` | High |
| Organization/Team Service (`org`) | `organization-service` | High |
| Audit Service (`audit`) | `audit-service` | High |
| Notification Service (`notification`) | `notification-service` | High |
| Billing Service (`billing`) | `billing-service` | High |
| SSH Proxy Service (`ssh-proxy`) | `ssh-proxy` | High — same name |
| Session Recording Service (`recording`) | `session-recorder` | Medium |
| Certificate Authority Service / PKI (`pki`) | `certificate-service` | Medium |
| SFTP Service (`sftp`) | `file-transfer-service` | Medium — doc 04 describes it as "SCP/SFTP file transfer brokering" |
| Port Forwarding Service (`port-forward`) | `tunnel-service` | Medium — doc 04 describes it as "TCP tunnel management" |
| Keychain Service (`keychain`) | `credential-manager` | Low — ambiguous with Vault Service; flagged for confirmation |

**Doc 04 names with NO canonical counterpart** (per CD-3, doc 01 is canonical — these are
divergent inventions to be reconciled or dropped in doc 04, not additional aliases for an
existing canonical service; listing them as aliases would be a bluff):

`rbac-service`, `scheduler-service`, `inventory-service`, `key-rotation-service`,
`compliance-service`, `reporting-service`, `search-service`, `webhook-service`,
`approval-workflow-service`, `policy-engine`, `metrics-aggregator`.

These eleven names describe capabilities that, in the canonical set, are either folded into
an existing service's Responsibilities (e.g. RBAC is part of Organization/Team Service §3.21;
key rotation is part of Keychain Service §3.12) or are simply not present in doc 01's
25-service catalog at all. Doc 04 is the document that must be reconciled against this
registry — this registry does not invent matching canonical services for names doc 01 never
enumerated.

## Reconciliation note

Any document under `docs/research/mvp/output/docs/markdown/` (or elsewhere in this spec set)
that enumerates the helix_terminator microservice set — by name, by count, or by directory
layout — and disagrees with the 25-service table above is exhibiting the exact defect
CD-3 was raised to close. The correct remediation is to edit that document to reference this
file (`SERVICE_REGISTRY.md`) rather than to re-derive or restate a divergent service list.
Where a document's own service names are historically load-bearing (e.g. existing code
samples, Kubernetes manifests already using a legacy name), the fix is to add an explicit
"see SERVICE_REGISTRY.md for the canonical name" cross-reference and, where feasible, rename
in place — not to leave two conflicting enumerations standing.
