# 08 — DevOps Infrastructure

**Status:** `Draft`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-4, CD-5, CD-6) + `SERVICE_REGISTRY.md`  

---

## Overview

HelixTerminator deploys on Kubernetes 1.31 with Helm charts, Terraform IaC for AWS EKS, and a multi-region disaster recovery topology. The monorepo is the single source of truth for all infrastructure definitions.

| Component | Technology | Version |
|-----------|-----------|---------|
| Orchestration | Kubernetes | 1.31 |
| Service Mesh | Istio | 1.22 |
| Package Manager | Helm | 3.x |
| IaC | Terraform | 1.7+ |
| Container Runtime | Docker / Podman | latest |
| Registry | Harbor / ECR | — |
| CI/CD | GitHub Actions | — |
| Observability | Prometheus + Grafana + Loki | — |

---

## Repository Structure

```
helix-terminator/
├── .github/workflows/          # GitHub Actions
├── constitution/              # HelixConstitution submodule
├── submodules/                # Platform library submodules
├── services/                  # 25 Go microservices
├── clients/flutter/           # Flutter client
├── infrastructure/
│   ├── kubernetes/            # Raw K8s manifests
│   ├── helm/                  # Helm charts
│   ├── terraform/             # EKS + AWS resources
│   ├── observability/         # Prometheus, Grafana, Loki
│   └── security/              # Network policies, PSS
├── scripts/
│   ├── dev/                   # Local dev helpers
│   ├── testing/               # Test runners, audit scripts
│   └── dr/                    # Disaster recovery runbooks
└── docs/
```

---

## Kubernetes Deployment

### Namespaces

| Namespace | Purpose |
|-----------|---------|
| `helixterm-prod` | Production workloads |
| `helixterm-staging` | Staging / integration |
| `helixterm-dev` | Developer ephemeral environments |
| `istio-system` | Istio control plane |
| `observability` | Prometheus, Grafana, Loki |

### Pod Security Standards

- Enforced: `restricted` profile (PSS v1.31)
- `runAsNonRoot: true`
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `seccompProfile: RuntimeDefault`
- Container UID: `65534` (distroless)

### Service Mesh (Istio 1.22)

- mTLS between all services (strict mode)
- AuthorizationPolicy per service
- Rate limiting at ingress gateway
- Circuit breaker configuration via `vasic-digital/recovery`

---

## Regions & DR

| Region | Role | Services |
|--------|------|----------|
| `us-east-1` | Primary | All 25 services + databases + Kafka + Redis |
| `eu-west-1` | DR | Hot standby; async replication |

**API Gateway ports:** `443` (edge, TLS) and `8080` (internal). Drop `8000`.

**RTO:** 15 minutes (automated failover)  
**RPO:** 5 minutes (async replication)

> **DEFERRED:** Full DR runbook with concrete failover commands, PostgreSQL PITR, and backup cadence are not yet authored. See [15 — Gap Analysis](../15-gap-analysis-remediation/).

---

## CI/CD Pipelines

### PR Pipeline

1. Lint, format, `go mod verify`, SAST
2. Build all services + Flutter (6 platforms)
3. Unit + integration tests
4. Contract tests (Pact)
5. E2E smoke tests
6. Anti-bluff audit

### Release Pipeline

1. Full E2E suite
2. Security scans
3. Canary deployment (10% → 50% → 100%)
4. SLO verification
5. Automated rollback on error-budget exhaustion

---

## Observability

| Signal | Tool | Retention |
|--------|------|-----------|
| Metrics | Prometheus | 15 days local, 1 year remote |
| Logs | Loki | 30 days |
| Traces | Jaeger / Tempo | 7 days |
| Dashboards | Grafana | — |
| Alerts | Alertmanager → PagerDuty/Slack | — |

### Key SLOs

| Service | Availability | Latency (p99) | Error Rate |
|---------|-------------|---------------|------------|
| API Gateway | 99.99% | 50ms | 0.01% |
| Auth Service | 99.99% | 100ms | 0.01% |
| SSH Proxy | 99.9% | 200ms | 0.1% |
| Terminal | 99.9% | 100ms | 0.1% |
| Vault | 99.99% | 50ms | 0.001% |

---

## Diagrams

| Diagram | Source |
|---------|--------|
| Kubernetes Deployment (Draw.io) | `diagrams/drawio/04_kubernetes_deployment.drawio` |
| Zero-Trust Network (Draw.io) | `diagrams/drawio/05_zero_trust_network.drawio` |
| K8s Deployment (Mermaid) | `diagrams/mermaid/26_kubernetes_deployment.mmd` |
| Network Stack | `diagrams/mermaid/27_network_stack.mmd` |
| Multi-Region DR | `diagrams/mermaid/28_multi_region_dr.mmd` |

---

## Cross-References

- [02 — System Architecture](../02-system-architecture/) — C4 diagrams, resilience matrix
- [03 — Service Catalog](../03-service-catalog/) — 25 services with ports and upstream deps
- [07 — Testing Strategy](../07-testing-strategy/) — CI/CD pipeline gates
- [09 — Security — Zero Trust](../09-security-zero-trust/) — mTLS, network policies, PSS
- [16 — References](../16-references/) — Canonical versions (CD-4: K8s 1.31, Istio 1.22)

---

*Section 08 — DevOps Infrastructure*  
*Consolidated from: 04_devops_infrastructure.md, CANONICAL_FACTS.md (CD-4, CD-5, CD-6)*
