# helix_terminator

A Go project owned by HelixDevelopment.

## What It Is

`helix_terminator` is a Go application developed under the governance of the Helix Constitution framework, which establishes universal rules for quality, transparency, and anti-bluff verification across all related projects.

## Governance

This project's governance is inherited from the **Helix Constitution**, mounted as a git submodule at `constitution/` (commit e6504c273c8b352fdb180449c9f057704cf85671, branch main).

All universal rules, policies, and quality standards are defined in the constitution. Project-specific rules are documented in:

- `./CLAUDE.md` — Claude Code project instructions
- `./AGENTS.md` — Agent operating rules
- `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md` — Project-specific extensions and overrides (currently none)

When this repository's rules disagree with the constitution submodule, **the constitution wins**.

## How to Verify Inheritance

Constitution inheritance is verified by three gate scripts that prove the constitution is actually inherited (not "bluff" gates that claim to check but don't):

1. **Gate (invariant check):**
   ```bash
   bash tests/verify_constitution_inheritance.sh
   ```
   Verifies that:
   - The `constitution/` directory exists and is populated
   - The constitutional anchor strings are present in the submodule
   - The parent repository files reference the submodule (invariants I1..I5)

2. **§1.1 Anti-bluff mutation proof:**
   ```bash
   bash scripts/testing/meta_test_false_positive_proof.sh
   ```
   Proves the gate is not a bluff by:
   - Mutating the forensic anchor in the constitution
   - Asserting the gate fails
   - Restoring the file and asserting the gate passes
   - Confirming the constitution/ tree remains pristine throughout

Run the gate before every build or merge.

## Constitution Privacy Policy

The constitution is **treated as public-by-policy**. No secrets, credentials, or project-specific configurations are ever ported into the constitution submodule. All sensitive or project-specific material stays in the parent repository. The constitution remains a reusable, shareable governance framework for all HelixDevelopment projects.

## Further Reading (constitution submodule)

- `constitution/Constitution.md` — The universal Helix Constitution (in submodule)
- `constitution/CLAUDE.md` — Universal rules for Claude Code and agents (in submodule)
- `constitution/AGENTS.md` — Universal agent operating rules (in submodule)

See the [Documentation](#documentation) section below for every other tracked
project document (governance, guides, reviews, runbooks, services,
infrastructure, and the full MVP research/specification package).

## Documentation

Every tracked project document is reachable from this section, directly or
transitively (Constitution §11.4.212). Governance documents (`CLAUDE.md`,
`AGENTS.md`) and this `README.md` are the roots; the constitution submodule's
own files live under `constitution/` and are governed separately (§11.4.35).

### Governance & Constitution

- [CLAUDE.md](CLAUDE.md) — Claude Code project instructions (inherits `constitution/CLAUDE.md`)
- [AGENTS.md](AGENTS.md) — Agent operating rules (inherits `constitution/AGENTS.md`)
- [GEMINI.md](GEMINI.md) — Gemini project instructions (inherits `constitution/GEMINI.md`)
- [QWEN.md](QWEN.md) — Qwen project instructions (inherits `constitution/QWEN.md`)
- [docs/CONSTITUTION_INHERITANCE.md](docs/CONSTITUTION_INHERITANCE.md) — How this project inherits + verifies constitution governance
- [docs/guides/HELIX_TERMINATOR_CONSTITUTION.md](docs/guides/HELIX_TERMINATOR_CONSTITUTION.md) — Project-specific constitution extensions (currently none promoted)

### Project Status, Planning & Requests

- [docs/CONTINUATION.md](docs/CONTINUATION.md) — Standing session-resumption record (§12.10 / §11.4.131) — read this first when resuming work
- [docs/COVERAGE_LEDGER.md](docs/COVERAGE_LEDGER.md) — §11.4.25 full-automation-coverage ledger (feature × platform × invariant)
- [docs/DEVELOPMENT_KICKOFF.md](docs/DEVELOPMENT_KICKOFF.md) — Development kick-off document — scope, plan, ground rules
- [CHANGELOG.md](CHANGELOG.md) — Notable changes to helix_terminator
- [docs/requests/history.md](docs/requests/history.md) — §11.4.208 operator-request-history ledger
- [docs/requests/feature_queue.md](docs/requests/feature_queue.md) — §11.4.213 durable FEATURE-directive research/scheduling queue

### Reviews & Audits

- [CODE_SCAFFOLD_REVIEW.md](CODE_SCAFFOLD_REVIEW.md) — Code scaffold review report
- [OPENDESIGN_COMPLIANCE_REVIEW.md](OPENDESIGN_COMPLIANCE_REVIEW.md) — OpenDesign (§11.4.162) compliance review
- [SECURITY_REVIEW.md](SECURITY_REVIEW.md) — Security review
- [docs/review/FINAL_REVIEW_20260709.md](docs/review/FINAL_REVIEW_20260709.md) — Final whole-branch review report
- [docs/review/RETEST_20260709.md](docs/review/RETEST_20260709.md) — §11.4.40 full-suite retest report

### Service Integration Guides

- [docs/guides/AI_SERVICE.md](docs/guides/AI_SERVICE.md) — AI service — local HelixLLM integration
- [docs/guides/BILLING.md](docs/guides/BILLING.md) — billing-service — real Stripe payment-provider integration
- [docs/guides/HELIXTRACK_BRIDGE.md](docs/guides/HELIXTRACK_BRIDGE.md) — HelixTrack Bridge service — real HelixTrack Core wiring
- [docs/guides/JWT_KEY_PROVISIONING.md](docs/guides/JWT_KEY_PROVISIONING.md) — JWT key provisioning for auth-service

### QA Transcripts (§11.4.83)

- [docs/qa/ai-service/TRANSCRIPT.md](docs/qa/ai-service/TRANSCRIPT.md) — QA transcript — ai-service real LLM integration
- [docs/qa/container-bridge/TRANSCRIPT.md](docs/qa/container-bridge/TRANSCRIPT.md) — QA transcript — container-bridge real ContainerRuntime integration
- [docs/qa/helixtrack-bridge/TRANSCRIPT.md](docs/qa/helixtrack-bridge/TRANSCRIPT.md) — QA transcript — helixtrack-bridge real Core integration

### Operational Runbooks

- [docs/runbooks/certificate-rotation.md](docs/runbooks/certificate-rotation.md) — Certificate rotation runbook
- [docs/runbooks/failover-procedure.md](docs/runbooks/failover-procedure.md) — Failover procedure runbook
- [docs/runbooks/incident-response.md](docs/runbooks/incident-response.md) — Incident response runbook
- [docs/runbooks/key-rotation.md](docs/runbooks/key-rotation.md) — Key rotation runbook
- [docs/runbooks/postgres-pitr-restore.md](docs/runbooks/postgres-pitr-restore.md) — PostgreSQL point-in-time-recovery restore runbook

### Script Documentation (§11.4.18)

- [docs/scripts/firebase_setup.md](docs/scripts/firebase_setup.md) — `firebase_setup.sh` — user guide
- [docs/scripts/install_git_hooks.md](docs/scripts/install_git_hooks.md) — `install_git_hooks.sh` — user guide

### Test Challenges & Chaos Engineering

- [test/challenges/architecture-challenges.md](test/challenges/architecture-challenges.md) — Architecture Challenges bank
- [test/challenges/backend-challenges.md](test/challenges/backend-challenges.md) — Backend Challenges bank
- [test/challenges/devops-challenges.md](test/challenges/devops-challenges.md) — DevOps Challenges bank
- [test/challenges/frontend-challenges.md](test/challenges/frontend-challenges.md) — Frontend Challenges bank
- [test/challenges/security-challenges.md](test/challenges/security-challenges.md) — Security Challenges bank
- [test/chaos/README.md](test/chaos/README.md) — LitmusChaos CRDs — chaos-engineering scenarios

### Client Applications

- [clients/flutter/README.md](clients/flutter/README.md) — HelixTerminator Flutter client — cross-platform app
- [clients/flutter/test/README.md](clients/flutter/test/README.md) — Flutter client test suite

### Microservices

Each service ships its own README (per-service internal doc, §11.4.212).

| Service | Description |
|---|---|
| [ai-service](services/ai-service/README.md) | Command autocomplete, output explanation, anomaly detection, runbook generation, incident assist |
| [analytics-service](services/analytics-service/README.md) | Session/command/transfer usage aggregation, dashboards, SLO tracking, Grafana/Prometheus export |
| [audit-service](services/audit-service/README.md) | Append-only Merkle-chained audit log, compliance query/export API, SOC 2/ISO 27001/FedRAMP evidence |
| [auth-service](services/auth-service/README.md) | Authentication (password, FIDO2/WebAuthn, TOTP, OIDC, SAML), JWT/refresh/device tokens, SCIM |
| [billing-service](services/billing-service/README.md) | Subscription/seat management, Stripe integration, invoicing, usage metering, trials, dunning |
| [collaboration-service](services/collaboration-service/README.md) | Real-time session sharing (observer/co-pilot/owner), CRDT buffer sync, broadcast, chat |
| [config-service](services/config-service/README.md) | Centralized feature flags + operational parameters, per-org overrides, runtime config propagation |
| [container-bridge-service](services/container-bridge-service/README.md) | Kubernetes cluster registration, pod exec/shell sessions, container log streaming |
| [gateway-service](services/gateway-service/README.md) | Single ingress for all client traffic, JWT validation, rate limiting, routing, circuit breaking |
| [health-service](services/health-service/README.md) | Aggregates /health/live + /health/ready from all services, unified health dashboard, SLO calc |
| [helixtrack-bridge-service](services/helixtrack-bridge-service/README.md) | Real JWT-authenticated link to a HelixTrack Core instance |
| [host-service](services/host-service/README.md) | SSH host/group CRUD, health ping, import/export, bastion/jump-host chains, host templates |
| [keychain-service](services/keychain-service/README.md) | Hardware-backed key storage (Secure Enclave / Android Keystore / DPAPI / keyring / HSM) |
| [notification-service](services/notification-service/README.md) | Multi-channel notification delivery (in-app, email, push, Slack, webhooks), templates |
| [org-service](services/org-service/README.md) | Manages organizations, teams, and memberships |
| [pki-service](services/pki-service/README.md) | Issues short-lived SSH certificates (user + host), CA rotation, revocation |
| [port-forward-service](services/port-forward-service/README.md) | Port-forward rule catalog + lifecycle (local/remote/dynamic/reverse), auto-reconnect |
| [recording-service](services/recording-service/README.md) | Assembles asciinema-format recordings from Kafka segments, Ed25519 signing, playback API |
| [sftp-service](services/sftp-service/README.md) | SFTP session/file operations, transfer queue + resume, checksum verification, dir sync |
| [snippet-service](services/snippet-service/README.md) | Command/script/SQL snippet CRUD, folders/namespaces, parameterization, execution history |
| [ssh-proxy-service](services/ssh-proxy-service/README.md) | Brokers SSH connections (password/pubkey/certificate auth), container-native sessions |
| [terminal-service](services/terminal-service/README.md) | WebSocket terminal I/O proxy, scrollback buffer (Redis), command-boundary detection |
| [user-service](services/user-service/README.md) | User CRUD, profile, preferences, onboarding state machine, SCIM provisioning endpoint |
| [vault-service](services/vault-service/README.md) | Zero-knowledge encrypted storage for SSH keys, passwords, API tokens, TLS certs, secret notes |
| [workspace-service](services/workspace-service/README.md) | Named workspace CRUD (hosts + snippets + vault items + settings), templates, sharing |

### Infrastructure

- [infrastructure/kubernetes/base/README.md](infrastructure/kubernetes/base/README.md) — Kubernetes base Kustomize layer
- [infrastructure/observability/grafana/README.md](infrastructure/observability/grafana/README.md) — Grafana dashboards
- [infrastructure/observability/jaeger/README.md](infrastructure/observability/jaeger/README.md) — Jaeger tracing config
- [infrastructure/observability/loki/README.md](infrastructure/observability/loki/README.md) — Loki logging config
- [infrastructure/observability/otel/README.md](infrastructure/observability/otel/README.md) — OpenTelemetry Collector config
- [infrastructure/security/SECURITY_POLICY.md](infrastructure/security/SECURITY_POLICY.md) — Pod Security Standards + cluster security policy
- [infrastructure/security/cosign/README.md](infrastructure/security/cosign/README.md) — Cosign image-signing configuration
- [infrastructure/security/falco/README.md](infrastructure/security/falco/README.md) — Falco runtime-security rules
- [infrastructure/security/sealed-secrets/README.md](infrastructure/security/sealed-secrets/README.md) — Sealed Secrets configuration
- [infrastructure/security/trivy/README.md](infrastructure/security/trivy/README.md) — Trivy security-scanning configuration
- [infrastructure/terraform/environments/production/README.md](infrastructure/terraform/environments/production/README.md) — Terraform production environment
- [infrastructure/terraform/environments/staging/README.md](infrastructure/terraform/environments/staging/README.md) — Terraform staging environment

### MVP Research & Full Technical Specification Package (`docs/research/mvp/`)

The MVP research/design package. Entry points first, then every document
underneath them so no file is orphaned; content that is byte-identically
mirrored at a second tracked path is marked *(mirror)*.

#### Entry points

- [docs/research/mvp/final/implementation/README.md](docs/research/mvp/final/implementation/README.md) — Implementation-doc master index (Quick Navigation + Critical Documents)
- [docs/research/mvp/final/implementation/INDEX.md](docs/research/mvp/final/implementation/INDEX.md) — Implementation-doc 16-section index
- [docs/research/mvp/output/README.md](docs/research/mvp/output/README.md) — Full Technical Specification Package overview (12 specification documents)
- [docs/research/mvp/REMEDIATION_REGISTER.md](docs/research/mvp/REMEDIATION_REGISTER.md) — Master remediation register — synthesized from 6 independent audits

#### Canonical facts, scope & registries

- [docs/research/mvp/final/implementation/CANONICAL_FACTS.md](docs/research/mvp/final/implementation/CANONICAL_FACTS.md) — Canonical facts (CD-1..CD-12) — single source of truth
- [docs/research/mvp/final/implementation/SCOPE_AND_MODULES.md](docs/research/mvp/final/implementation/SCOPE_AND_MODULES.md) — Dual-module (Terminal Platform / Connection Broker) scope boundary
- [docs/research/mvp/final/implementation/SERVICE_REGISTRY.md](docs/research/mvp/final/implementation/SERVICE_REGISTRY.md) — Canonical 25-service registry
- [docs/research/mvp/final/implementation/01-executive-summary/CANONICAL_FACTS.md](docs/research/mvp/final/implementation/01-executive-summary/CANONICAL_FACTS.md) — *(mirror)* Canonical facts, as republished in the executive-summary section
- [docs/research/mvp/final/implementation/01-executive-summary/SCOPE_AND_MODULES.md](docs/research/mvp/final/implementation/01-executive-summary/SCOPE_AND_MODULES.md) — Scope & module boundary, as republished in the executive-summary section
- [docs/research/mvp/final/implementation/01-executive-summary/SERVICE_REGISTRY.md](docs/research/mvp/final/implementation/01-executive-summary/SERVICE_REGISTRY.md) — Service registry, as republished in the executive-summary section
- [docs/research/mvp/output/CANONICAL_FACTS.md](docs/research/mvp/output/CANONICAL_FACTS.md) — *(mirror)* Canonical facts, as republished in the output package
- [docs/research/mvp/output/SCOPE_AND_MODULES.md](docs/research/mvp/output/SCOPE_AND_MODULES.md) — Scope & module boundary, as republished in the output package
- [docs/research/mvp/output/SERVICE_REGISTRY.md](docs/research/mvp/output/SERVICE_REGISTRY.md) — Service registry, as republished in the output package

#### Implementation sections (01–16)

- [01-executive-summary/README.md](docs/research/mvp/final/implementation/01-executive-summary/README.md) — Executive Summary — mission, positioning, target audience, pricing
- [02-system-architecture/README.md](docs/research/mvp/final/implementation/02-system-architecture/README.md) — System Architecture — C4 diagrams, 3-channel model, resilience matrix *(mirror: `architecture/`)*
- [03-service-catalog/README.md](docs/research/mvp/final/implementation/03-service-catalog/README.md) — Service Catalog — canonical 25 services, module paths, ports, DBs, deps
- [04-api-specification/README.md](docs/research/mvp/final/implementation/04-api-specification/README.md) — API Specification — OpenAPI 3.1, 221 REST endpoints, gRPC .proto *(mirror: `api/`)*
- [05-database-schema/README.md](docs/research/mvp/final/implementation/05-database-schema/README.md) — Database Schema — 120 CREATE TABLE statements, 261 indexes
- [06-client-specification/README.md](docs/research/mvp/final/implementation/06-client-specification/README.md) — Client Specification — Flutter/Dart, BLoC, 6 platforms *(mirror: `frontend/`)*
- [07-testing-strategy/README.md](docs/research/mvp/final/implementation/07-testing-strategy/README.md) — Testing Strategy — 17 test types, Go tests, k6 scripts, CI gates *(mirror: `testing/`)*
- [08-devops-infrastructure/README.md](docs/research/mvp/final/implementation/08-devops-infrastructure/README.md) — DevOps Infrastructure — K8s 1.31, Helm, Terraform, CI/CD, DR *(mirror: `infrastructure/`)*
- [09-security-zero-trust/README.md](docs/research/mvp/final/implementation/09-security-zero-trust/README.md) — Security — Zero Trust — SPIFFE/SPIRE, mTLS, vault, RBAC, PKI *(mirror: `security/`)*
- [10-ux-design-system/README.md](docs/research/mvp/final/implementation/10-ux-design-system/README.md) — UX Design System — 750+ tokens, 35 components, 25 wireframes
- [11-performance-analysis/README.md](docs/research/mvp/final/implementation/11-performance-analysis/README.md) — Performance Analysis — SLOs, gap analysis, k6, benchmarks
- [12-product-roadmap/README.md](docs/research/mvp/final/implementation/12-product-roadmap/README.md) — Product Roadmap — 5 phases, 50 use cases, 41 edge cases
- [13-submodule-integration/README.md](docs/research/mvp/final/implementation/13-submodule-integration/README.md) — Submodule Integration — 17 submodules, dependency manifest
- [14-constitution-compliance/README.md](docs/research/mvp/final/implementation/14-constitution-compliance/README.md) — Constitution Compliance — AGENTS/CLAUDE, helix-deps.yaml, CI gates
- [15-gap-analysis-remediation/README.md](docs/research/mvp/final/implementation/15-gap-analysis-remediation/README.md) — Gap Analysis & Remediation — 253 findings, canonical decisions
- [16-references/README.md](docs/research/mvp/final/implementation/16-references/README.md) — References — canonical facts, service registry, remediation register, changelog
- [docs/research/mvp/final/implementation/15-gap-analysis-remediation/GAP_REGISTER.md](docs/research/mvp/final/implementation/15-gap-analysis-remediation/GAP_REGISTER.md) — Gap register (section-15 copy)
- [docs/research/mvp/final/implementation/GAP_REGISTER.md](docs/research/mvp/final/implementation/GAP_REGISTER.md) — *(mirror)* Gap register, republished at the implementation root

- [docs/research/mvp/final/implementation/architecture/README.md](docs/research/mvp/final/implementation/architecture/README.md) — *(mirror of `02-system-architecture/`)*
- [docs/research/mvp/final/implementation/api/README.md](docs/research/mvp/final/implementation/api/README.md) — *(mirror of `04-api-specification/`)*
- [docs/research/mvp/final/implementation/backend/README.md](docs/research/mvp/final/implementation/backend/README.md) — Service catalog / backend section (own content, distinct from `03-service-catalog/`)
- [docs/research/mvp/final/implementation/frontend/README.md](docs/research/mvp/final/implementation/frontend/README.md) — *(mirror of `06-client-specification/`)*
- [docs/research/mvp/final/implementation/testing/README.md](docs/research/mvp/final/implementation/testing/README.md) — *(mirror of `07-testing-strategy/`)*
- [docs/research/mvp/final/implementation/infrastructure/README.md](docs/research/mvp/final/implementation/infrastructure/README.md) — *(mirror of `08-devops-infrastructure/`)*
- [docs/research/mvp/final/implementation/security/README.md](docs/research/mvp/final/implementation/security/README.md) — *(mirror of `09-security-zero-trust/`)*

#### Implementation-root reference documents

- [ARCHITECTURE_OVERVIEW.md](docs/research/mvp/final/implementation/ARCHITECTURE_OVERVIEW.md) — High-level system architecture overview
- [CONSTITUTION_INHERITANCE.md](docs/research/mvp/final/implementation/CONSTITUTION_INHERITANCE.md) — *(mirror)* Constitution inheritance, republished under the implementation tree
- [CONTINUATION.md](docs/research/mvp/final/implementation/CONTINUATION.md) — Session-resumption record scoped to the MVP-spec subdirectory
- [CONTRIBUTING.md](docs/research/mvp/final/implementation/CONTRIBUTING.md) — Contribution guidelines, PR process, code standards
- [DEPLOYMENT_GUIDE.md](docs/research/mvp/final/implementation/DEPLOYMENT_GUIDE.md) — Deployment procedures for all environments
- [DEVELOPMENT_KICKOFF.md](docs/research/mvp/final/implementation/DEVELOPMENT_KICKOFF.md) — *(mirror)* Development kick-off document, republished under the implementation tree
- [DEVELOPMENT_PLAN.md](docs/research/mvp/final/implementation/DEVELOPMENT_PLAN.md) — Full development plan
- [MASTER_DEVELOPMENT_PLAN.md](docs/research/mvp/final/implementation/MASTER_DEVELOPMENT_PLAN.md) — Master development plan — 7 parallel work streams
- [REVIEW_REPORT.md](docs/research/mvp/final/implementation/REVIEW_REPORT.md) — Documentation completeness review report
- [SECURITY_RUNBOOK.md](docs/research/mvp/final/implementation/SECURITY_RUNBOOK.md) — Security procedures and incident response
- [TROUBLESHOOTING.md](docs/research/mvp/final/implementation/TROUBLESHOOTING.md) — Common issues and resolution steps

#### Architecture Decision Records (ADRs)

Canonical copy at `user-guides/ADRs/`; byte-identical mirror at `12-guides/ADRs/`.

| ADR | Canonical | Mirror |
|---|---|---|
| Flutter over Electron for cross-platform desktop | [user-guides/ADRs/ADR-001-flutter-over-electron.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-001-flutter-over-electron.md) | [12-guides/ADRs/ADR-001-flutter-over-electron.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-001-flutter-over-electron.md) |
| Go over Rust/Node.js for backend microservices | [user-guides/ADRs/ADR-002-go-over-rust-node.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-002-go-over-rust-node.md) | [12-guides/ADRs/ADR-002-go-over-rust-node.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-002-go-over-rust-node.md) |
| Kafka + RabbitMQ over NATS for messaging | [user-guides/ADRs/ADR-003-kafka-over-nats.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-003-kafka-over-nats.md) | [12-guides/ADRs/ADR-003-kafka-over-nats.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-003-kafka-over-nats.md) |
| PostgreSQL over CockroachDB for primary datastore | [user-guides/ADRs/ADR-004-postgres-over-cockroach.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-004-postgres-over-cockroach.md) | [12-guides/ADRs/ADR-004-postgres-over-cockroach.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-004-postgres-over-cockroach.md) |
| SPIFFE/SPIRE over HashiCorp Vault for workload identity | [user-guides/ADRs/ADR-005-spiffe-over-vault-identity.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-005-spiffe-over-vault-identity.md) | [12-guides/ADRs/ADR-005-spiffe-over-vault-identity.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-005-spiffe-over-vault-identity.md) |
| CRDTs over Operational Transformation for real-time collaboration | [user-guides/ADRs/ADR-006-crdt-over-ot.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-006-crdt-over-ot.md) | [12-guides/ADRs/ADR-006-crdt-over-ot.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-006-crdt-over-ot.md) |
| Helm over Kustomize for Kubernetes packaging | [user-guides/ADRs/ADR-007-helm-over-kustomize.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-007-helm-over-kustomize.md) | [12-guides/ADRs/ADR-007-helm-over-kustomize.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-007-helm-over-kustomize.md) |
| Terraform over Pulumi for infrastructure as code | [user-guides/ADRs/ADR-008-terraform-over-pulumi.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-008-terraform-over-pulumi.md) | [12-guides/ADRs/ADR-008-terraform-over-pulumi.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-008-terraform-over-pulumi.md) |
| EdDSA (Ed25519) over RSA for JWT signing | [user-guides/ADRs/ADR-009-eddsa-over-rsa.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-009-eddsa-over-rsa.md) | [12-guides/ADRs/ADR-009-eddsa-over-rsa.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-009-eddsa-over-rsa.md) |
| Microservices over monolith for system architecture | [user-guides/ADRs/ADR-010-microservices-over-monolith.md](docs/research/mvp/final/implementation/user-guides/ADRs/ADR-010-microservices-over-monolith.md) | [12-guides/ADRs/ADR-010-microservices-over-monolith.md](docs/research/mvp/final/implementation/12-guides/ADRs/ADR-010-microservices-over-monolith.md) |

- [docs/research/mvp/final/implementation/user-guides/HELIX_TERMINATOR_CONSTITUTION.md](docs/research/mvp/final/implementation/user-guides/HELIX_TERMINATOR_CONSTITUTION.md) — *(mirror)* Project constitution, republished under `user-guides/`

#### Runbooks (MVP package)

Canonical copy at `user-guides/runbooks/`; byte-identical mirror at `12-guides/runbooks/`.

| Runbook | Canonical | Mirror |
|---|---|---|
| Certificate rotation runbook | [user-guides/runbooks/CERTIFICATE_ROTATION.md](docs/research/mvp/final/implementation/user-guides/runbooks/CERTIFICATE_ROTATION.md) | [12-guides/runbooks/CERTIFICATE_ROTATION.md](docs/research/mvp/final/implementation/12-guides/runbooks/CERTIFICATE_ROTATION.md) |
| Failover procedure runbook | [user-guides/runbooks/FAILOVER_PROCEDURE.md](docs/research/mvp/final/implementation/user-guides/runbooks/FAILOVER_PROCEDURE.md) | [12-guides/runbooks/FAILOVER_PROCEDURE.md](docs/research/mvp/final/implementation/12-guides/runbooks/FAILOVER_PROCEDURE.md) |
| Incident response runbook | [user-guides/runbooks/INCIDENT_RESPONSE.md](docs/research/mvp/final/implementation/user-guides/runbooks/INCIDENT_RESPONSE.md) | [12-guides/runbooks/INCIDENT_RESPONSE.md](docs/research/mvp/final/implementation/12-guides/runbooks/INCIDENT_RESPONSE.md) |
| Kafka recovery runbook | [user-guides/runbooks/KAFKA_RECOVERY.md](docs/research/mvp/final/implementation/user-guides/runbooks/KAFKA_RECOVERY.md) | [12-guides/runbooks/KAFKA_RECOVERY.md](docs/research/mvp/final/implementation/12-guides/runbooks/KAFKA_RECOVERY.md) |
| Key rotation runbook | [user-guides/runbooks/KEY_ROTATION.md](docs/research/mvp/final/implementation/user-guides/runbooks/KEY_ROTATION.md) | [12-guides/runbooks/KEY_ROTATION.md](docs/research/mvp/final/implementation/12-guides/runbooks/KEY_ROTATION.md) |
| PostgreSQL PITR runbook | [user-guides/runbooks/POSTGRES_PITR.md](docs/research/mvp/final/implementation/user-guides/runbooks/POSTGRES_PITR.md) | [12-guides/runbooks/POSTGRES_PITR.md](docs/research/mvp/final/implementation/12-guides/runbooks/POSTGRES_PITR.md) |
| SSH CA incident runbook | [user-guides/runbooks/SSH_CA_INCIDENT.md](docs/research/mvp/final/implementation/user-guides/runbooks/SSH_CA_INCIDENT.md) | [12-guides/runbooks/SSH_CA_INCIDENT.md](docs/research/mvp/final/implementation/12-guides/runbooks/SSH_CA_INCIDENT.md) |
| Vault breach runbook | [user-guides/runbooks/VAULT_BREACH.md](docs/research/mvp/final/implementation/user-guides/runbooks/VAULT_BREACH.md) | [12-guides/runbooks/VAULT_BREACH.md](docs/research/mvp/final/implementation/12-guides/runbooks/VAULT_BREACH.md) |

#### OpenDesign / Design System

Canonical copy at `design/`; byte-identical mirror at `opendesign/`.

| Doc | Canonical | Mirror |
|---|---|---|
| HelixTerminator × OpenDesign integration overview | [design/README.md](docs/research/mvp/final/implementation/design/README.md) | [opendesign/README.md](docs/research/mvp/final/implementation/opendesign/README.md) |
| OpenDesign integration plan | [design/INTEGRATION_PLAN.md](docs/research/mvp/final/implementation/design/INTEGRATION_PLAN.md) | [opendesign/INTEGRATION_PLAN.md](docs/research/mvp/final/implementation/opendesign/INTEGRATION_PLAN.md) |
| Visual regression test strategy | [design/VISUAL_REGRESSION_STRATEGY.md](docs/research/mvp/final/implementation/design/VISUAL_REGRESSION_STRATEGY.md) | [opendesign/VISUAL_REGRESSION_STRATEGY.md](docs/research/mvp/final/implementation/opendesign/VISUAL_REGRESSION_STRATEGY.md) |
| Component library specification | [design/component-library-spec.md](docs/research/mvp/final/implementation/design/component-library-spec.md) | [opendesign/component-library-spec.md](docs/research/mvp/final/implementation/opendesign/component-library-spec.md) |

#### Full specification documents (`output/docs/markdown/`)

- [01_core_architecture.md](docs/research/mvp/output/docs/markdown/01_core_architecture.md) — Core Architecture Specification
- [02_client_specification.md](docs/research/mvp/output/docs/markdown/02_client_specification.md) — Client-Side Technical Specification
- [03_testing_strategy.md](docs/research/mvp/output/docs/markdown/03_testing_strategy.md) — Testing Strategy & QA Specification
- [04_devops_infrastructure.md](docs/research/mvp/output/docs/markdown/04_devops_infrastructure.md) — DevOps & Infrastructure Specification
- [05_security_zero_trust.md](docs/research/mvp/output/docs/markdown/05_security_zero_trust.md) — Security & Zero Trust Architecture Specification
- [06_ux_design_system.md](docs/research/mvp/output/docs/markdown/06_ux_design_system.md) — UX Design System Specification
- [07_api_and_database.md](docs/research/mvp/output/docs/markdown/07_api_and_database.md) — API & Database Specification
- [08_product_roadmap_features.md](docs/research/mvp/output/docs/markdown/08_product_roadmap_features.md) — Product Roadmap, Feature Specifications, Use Cases & Edge Cases
- [09_performance_analysis.md](docs/research/mvp/output/docs/markdown/09_performance_analysis.md) — Performance Analysis Specification
- [10_submodule_integration.md](docs/research/mvp/output/docs/markdown/10_submodule_integration.md) — Submodule Integration Specification
- [11_constitution_compliance.md](docs/research/mvp/output/docs/markdown/11_constitution_compliance.md) — Constitution Compliance Specification
- [12_mermaid_diagrams.md](docs/research/mvp/output/docs/markdown/12_mermaid_diagrams.md) — Mermaid Diagram Suite (source, markdown copy)
- [docs/research/mvp/output/diagrams/mermaid/mermaid_diagrams.md](docs/research/mvp/output/diagrams/mermaid/mermaid_diagrams.md) — Complete Mermaid diagram suite (30 diagrams, rendered forms alongside)
