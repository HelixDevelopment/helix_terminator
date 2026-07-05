# ADR-007: Helm over Kustomize for Kubernetes Packaging

## Status
Accepted

## Context
helix_terminator deploys ~25 microservices, multiple datastores, and infrastructure components (Kafka, RabbitMQ, PostgreSQL, SPIRE, Vault) across staging, production, and disaster-recovery Kubernetes clusters. We need a packaging and templating system that supports versioned releases, parameterization, and reuse across environments.

## Decision
We chose **Helm** as the primary Kubernetes packaging and templating tool. Kustomize is used selectively for cluster-level overlays and patch-only workflows where no chart versioning is needed.

## Consequences

### Positive
- **Versioned artifacts**: Helm charts are semantically versioned and stored in OCI registries, enabling reproducible rollouts and rollbacks.
- **Templating**: Go-template parameterization allows a single chart to serve multiple environments with `values.yaml` overrides.
- **Ecosystem reuse**: We consume upstream charts (Bitnami, Jetstack, SPIRE) directly, reducing boilerplate.
- **Release management**: `helm list`, `helm rollback`, and `helm history` provide built-in release lifecycle tracking.
- **Subcharts and dependencies**: Composite charts (e.g., `helix-platform`) bundle related services with shared configuration.

### Negative
- **Template complexity**: Go templates can become unwieldy; discipline is required to keep logic out of templates.
- **Tiller legacy**: Modern Helm (v3+) is Tiller-less, but older documentation and third-party tools still reference the insecure v2 model.
- **YAML bloat**: Charts can generate verbose YAML; `helm template` output must be reviewed in CI.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Kustomize** | Excellent for GitOps-style patch-and-overlay workflows with no templating, but lacks built-in versioning, release management, and dependency bundling. Used for cluster bootstrap manifests where no parameterization is needed. |
| **Plain YAML + sed** | Fragile and error-prone; rejected for anything beyond one-off debugging. |
| **Jsonnet / Tanka** | Powerful and deterministic, but introduces a non-standard language (Jsonnet) with a smaller community than Helm; steeper learning curve for new team members. |
| **Operator SDK / OLM** | Powerful for complex stateful applications, but overkill for stateless microservices and adds significant development overhead. |

## References
- `infrastructure/helm/` — All Helm charts and values
- `infrastructure/helm/helix-platform/` — Top-level platform chart
- `docs/guides/runbooks/FAILOVER_PROCEDURE.md` — Helm-based DR runbook
