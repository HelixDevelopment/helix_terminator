# ADR-005: SPIFFE/SPIRE over HashiCorp Vault for Workload Identity

## Status
Accepted

## Context
Services in helix_terminator must authenticate each other over mTLS without relying on long-lived shared secrets or human-provisioned certificates. We need an automated, auditable, and platform-agnostic workload identity system that integrates with Kubernetes.

## Decision
We chose **SPIFFE/SPIRE** as the workload identity framework. HashiCorp Vault is retained for secret management and encryption-as-a-service, but not as the primary workload identity provider.

## Consequences

### Positive
- **Standardized identity**: SPIFFE IDs (e.g., `spiffe://helix.internal/ns/production/sa/audit-service`) are uniform across clouds and clusters.
- **Automatic rotation**: SPIRE agents issue short-lived SVIDs (X.509 or JWT) with automatic renewal, eliminating manual certificate provisioning.
- **No bootstrap secrets**: Workloads authenticate via node attestation (Kubernetes Service Account, AWS IID, etc.), reducing secret sprawl.
- **Federation**: SPIFFE federation enables cross-cluster and cross-cloud trust without shared CA infrastructure.
- **Platform agnostic**: SPIRE runs on Kubernetes, VMs, and bare metal with consistent semantics.

### Negative
- **Infrastructure overhead**: SPIRE server and agent deployment adds pods and etcd/stateful storage to manage.
- **Learning curve**: SPIFFE concepts (SVID, attestation, federation) are less ubiquitous than Vault’s token-based model.
- **Integration gaps**: Some third-party tools natively support Vault TLS certs but not SPIFFE; adapters or sidecars are required.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **HashiCorp Vault (PKI + AppRole)** | Vault’s PKI engine can issue short-lived certs, but AppRole authentication requires bootstrap secrets (role ID / secret ID) and Vault itself becomes a single point of failure for identity issuance. Retained for secrets and encryption, not identity. |
| **Kubernetes cert-manager + internal CA** | Works for in-cluster mTLS, but lacks standardized cross-cluster identity and requires manual CA distribution and rotation. |
| **AWS IAM / GCP Service Accounts** | Cloud-native but not portable; multi-cloud and on-prem deployments would require separate identity silos. |
| **Istio Citadel (now istiod)** | Convenient for service mesh mTLS, but ties identity to the mesh control plane; SPIFFE is mesh-agnostic and future-proof. |

## References
- `infrastructure/helm/spire/` — SPIRE server and agent Helm charts
- `infrastructure/security/spiffe/` — Trust domain and federation configuration
- `docs/guides/runbooks/SSH_CA_INCIDENT.md` — Related CA compromise procedures
