# 16 — References

**Status:** `Complete`  
**Module:** A + B  
**Authority:** All canonical documents

---

## Canonical Documents

| Document | Path | Purpose |
|----------|------|---------|
| Canonical Facts | `../CANONICAL_FACTS.md` | CD-1..CD-12: versions, identity, networking, auth, scope |
| Service Registry | `../SERVICE_REGISTRY.md` | 25 canonical services with module paths, ports, DBs, deps |
| Scope & Modules | `../SCOPE_AND_MODULES.md` | Dual-module boundary reconciliation (Module A + Module B) |
| Remediation Register | `../REMEDIATION_REGISTER.md` | 253 findings, 12 canonical decisions, fix-now/deep-work |

---

## Source Documents (12 Spec Documents)

| # | Document | Module | Lines | Status in Consolidation |
|---|----------|--------|-------|------------------------|
| 01 | Core Architecture | A | 6,377 | Consolidated into 02, 03, 05 |
| 02 | Client Specification | A | 8,226 | Consolidated into 06 |
| 03 | Testing Strategy | A | 6,577 | Consolidated into 07 |
| 04 | DevOps Infrastructure | A | 8,220 | Consolidated into 08 |
| 05 | Security — Zero Trust | A | 4,330 | Consolidated into 09 |
| 06 | UX Design System | A | 10,737 | Consolidated into 10 |
| 07 | API & Database | A | 8,744 | Consolidated into 04, 05 |
| 08 | Product Roadmap | A | 3,417 | Consolidated into 12 |
| 09 | Performance Analysis | A | 6,260 | Consolidated into 11 |
| 10 | Submodule Integration | A | 6,781 | Consolidated into 13 |
| 11 | Constitution Compliance | B | 5,460 | Consolidated into 14 |
| 12 | Mermaid Diagrams | A | 2,745 | Referenced throughout all sections |

**Total: ~77,874 lines** of original specification content.

---

## Canonical Facts Summary (CD-1..CD-12)

| CD | Topic | Canonical Value |
|----|-------|-----------------|
| CD-1 | Product scope | Dual: Module A (SSH/terminal) + Module B (WireGuard broker) |
| CD-2 | Org / domain | `HelixDevelopment` / `helixterminator.io` |
| CD-3 | Service count | 25 microservices (doc 01 set) |
| CD-4 | Versions | PostgreSQL 17.2, Go 1.25, Kafka 3.9, Redis 8, K8s 1.31, Flutter 3.24, Istio 1.22 |
| CD-5 | API gateway port | 443 (edge) + 8080 (internal); drop 8000 |
| CD-6 | Regions | us-east-1 primary, eu-west-1 DR |
| CD-7 | JWT signing | EdDSA (Ed25519), iss: auth.helixterminator.io |
| CD-8 | RBAC roles | super_admin, org_admin, team_admin, member, auditor, api_user |
| CD-9 | Constitution | Pinned e6504c2, helixcode-v1.1.0 line |
| CD-10 | Zero-knowledge | HARD for vault items; SSH password-auth explicitly non-ZK |
| CD-11 | Dependencies | One helix-deps.yaml, slash-path imports |
| CD-12 | Test types | 12 mandatory types |

---

## Service Registry Summary

| # | Service | Module Path | Port | Database |
|---|---------|-------------|------|----------|
| 1 | API Gateway | `helixterminator.io/services/gateway` | 8080 | none |
| 2 | Auth | `helixterminator.io/services/auth` | 8081 | `helixterm_auth` |
| 3 | User | `helixterminator.io/services/user` | 8082 | `helixterm_users` |
| 4 | Vault | `helixterminator.io/services/vault` | 8083 | `helixterm_vault` |
| 5 | Host | `helixterminator.io/services/host` | 8084 | `helixterm_hosts` |
| 6 | SSH Proxy | `helixterminator.io/services/ssh-proxy` | 8085 | `helixterm_ssh_proxy` |
| 7 | Terminal | `helixterminator.io/services/terminal` | 8086 | `helixterm_terminal` |
| 8 | SFTP | `helixterminator.io/services/sftp` | 8087 | `helixterm_sftp` |
| 9 | Port Forward | `helixterminator.io/services/port-forward` | 8088 | `helixterm_port_forward` |
| 10 | Snippet | `helixterminator.io/services/snippet` | 8089 | `helixterm_snippets` |
| 11 | Keychain | `helixterminator.io/services/keychain` | 8090 | `helixterm_keychain` |
| 12 | Workspace | `helixterminator.io/services/workspace` | 8091 | `helixterm_workspaces` |
| 13 | Collaboration | `helixterminator.io/services/collab` | 8092 | `helixterm_collab` |
| 14 | Notification | `helixterminator.io/services/notification` | 8093 | `helixterm_notifications` |
| 15 | Audit | `helixterminator.io/services/audit` | 8094 | `helixterm_audit` |
| 16 | Analytics | `helixterminator.io/services/analytics` | 8095 | `helixterm_analytics` |
| 17 | AI | `helixterminator.io/services/ai` | 8096 | `helixterm_ai` |
| 18 | Recording | `helixterminator.io/services/recording` | 8097 | `helixterm_recordings` |
| 19 | PKI | `helixterminator.io/services/pki` | 8098 | `helixterm_pki` |
| 20 | Organization | `helixterminator.io/services/org` | 8099 | `helixterm_org` |
| 21 | Billing | `helixterminator.io/services/billing` | 8100 | `helixterm_billing` |
| 22 | Configuration | `helixterminator.io/services/config` | 8101 | `helixterm_config` |
| 23 | Health | `helixterminator.io/services/health` | 8102 | none |
| 24 | Container Bridge | `helixterminator.io/services/container-bridge` | 8103 | `helixterm_container_bridge` |
| 25 | HelixTrack Bridge | `helixterminator.io/services/helixtrack-bridge` | 8104 | `helixterm_helixtrack_bridge` |

---

## Diagrams Reference

### Mermaid (30 diagrams)

| # | Diagram | Source | Formats |
|---|---------|--------|---------|
| 01 | C4 System Context | `diagrams/mermaid/01_c4_context.mmd` | mmd, svg, png, pdf, jpeg, html |
| 02 | C4 Container | `diagrams/mermaid/02_c4_container.mmd` | mmd, svg, png, pdf, jpeg, html |
| 03 | Microservices Overview | `diagrams/mermaid/03_microservices.mmd` | mmd, svg, png, pdf, jpeg, html |
| 04 | Dependency Graph | `diagrams/mermaid/04_dependency_graph.mmd` | mmd, svg, png, pdf, jpeg, html |
| 05 | Data Flow | `diagrams/mermaid/05_data_flow.mmd` | mmd, svg, png, pdf, jpeg, html |
| 06-15 | Sequence Diagrams (SSH, Auth, Vault, SFTP, Collab, CA, Encryption, Gateway, mTLS, Recording) | `diagrams/mermaid/06-15_*.mmd` | mmd, svg, png, pdf, jpeg, html |
| 16-20 | State Machines (SSH, Vault, SFTP, Auth, Health) | `diagrams/mermaid/16-20_*.mmd` | mmd, svg, png, pdf, jpeg, html |
| 21-23 | ER Diagrams (Core, Auth, SSH/Terminal) | `diagrams/mermaid/21-23_*.mmd` | mmd, svg, png, pdf, jpeg, html |
| 24-25 | Class Diagrams (Go, Flutter BLoC) | `diagrams/mermaid/24-25_*.mmd` | mmd, svg, png, pdf, jpeg, html |
| 26-28 | Deployment (K8s, Network, Multi-region) | `diagrams/mermaid/26-28_*.mmd` | mmd, svg, png, pdf, jpeg, html |
| 29-30 | Project/Pipeline (Gantt, CI/CD) | `diagrams/mermaid/29-30_*.mmd` | mmd, svg, png, pdf, jpeg, html |

### Draw.io (8 diagrams)

| # | Diagram | Source | Formats |
|---|---------|--------|---------|
| 01 | System Architecture | `diagrams/drawio/01_system_architecture.drawio` | drawio, svg, png, pdf, jpeg, html |
| 02 | SSH Connection Flow | `diagrams/drawio/02_ssh_connection_flow.drawio` | drawio, svg, png, pdf, jpeg, html |
| 03 | Vault Encryption | `diagrams/drawio/03_vault_encryption.drawio` | drawio, svg, png, pdf, jpeg, html |
| 04 | Kubernetes Deployment | `diagrams/drawio/04_kubernetes_deployment.drawio` | drawio, svg, png, pdf, jpeg, html |
| 05 | Zero-Trust Network | `diagrams/drawio/05_zero_trust_network.drawio` | drawio, svg, png, pdf, jpeg, html |
| 06 | Data Model ER | `diagrams/drawio/06_data_model_er.drawio` | drawio, svg, png, pdf, jpeg, html |
| 07 | Kafka Message Flow | `diagrams/drawio/07_kafka_message_flow.drawio` | drawio, svg, png, pdf, jpeg, html |
| 08 | Client Architecture | `diagrams/drawio/08_client_architecture.drawio` | drawio, svg, png, pdf, jpeg, html |

---

## Key Numbers

| Metric | Count |
|--------|-------|
| Microservices | 25 |
| REST API endpoints | 221 |
| SQL CREATE TABLE statements | 120 |
| Database indexes | 261 |
| Mermaid diagrams | 30 |
| Draw.io diagrams | 8 |
| Submodule integrations | 17 |
| Canonical decisions | 12 |
| Remediation findings | 253 |
| Test types | 12 |
| Development phases | 5 |
| Use cases | 50 |
| Edge cases | 41 |
| Performance benchmarks | 60+ |
| Design tokens | 750+ |
| UI components | 35 |
| Screen wireframes | 25 |
| Keyboard shortcuts | 130+ |

---

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2026-06-28 | 1.0.0 | Initial 12-document specification package generated |
| 2026-07-04 | 1.1.0 | Canonical Facts (CD-1..CD-12) locked; Service Registry published; Scope & Modules reconciliation authored |
| 2026-07-05 | 2.0.0 | Consolidated implementation documentation (16 sections) created at `docs/research/mvp/final/implementation/` |

---

## Cross-References

All sections link here for canonical facts, service registry, and remediation register.

- [01 — Executive Summary](../01-executive-summary/) — Mission, positioning, pricing
- [02 — System Architecture](../02-system-architecture/) — C4 diagrams, resilience matrix
- [03 — Service Catalog](../03-service-catalog/) — 25 services with module paths
- [04 — API Specification](../04-api-specification/) — OpenAPI 3.1, gRPC protos
- [05 — Database Schema](../05-database-schema/) — SQL schemas, RLS policies
- [06 — Client Specification](../06-client-specification/) — Flutter, BLoC, 6 platforms
- [07 — Testing Strategy](../07-testing-strategy/) — 12 test types, CI gates
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — K8s, Helm, Terraform, DR
- [09 — Security — Zero Trust](../09-security-zero-trust/) — SPIFFE, mTLS, RBAC, PKI
- [10 — UX Design System](../10-ux-design-system/) — Tokens, components, wireframes
- [11 — Performance Analysis](../11-performance-analysis/) — SLOs, benchmarks
- [12 — Product Roadmap](../12-product-roadmap/) — 5 phases, 50 use cases
- [13 — Submodule Integration](../13-submodule-integration/) — 17 submodules, Go code
- [14 — Constitution Compliance](../14-constitution-compliance/) — Governance, CI gates
- [15 — Gap Analysis](../15-gap-analysis-remediation/) — 253 findings, deep-work

---

*Section 16 — References*  
*Consolidated from: CANONICAL_FACTS.md, SERVICE_REGISTRY.md, SCOPE_AND_MODULES.md, REMEDIATION_REGISTER.md, README.md*
