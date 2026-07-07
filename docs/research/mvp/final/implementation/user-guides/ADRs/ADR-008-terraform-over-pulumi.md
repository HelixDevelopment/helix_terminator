# ADR-008: Terraform over Pulumi for Infrastructure as Code

## Status
Accepted

## Context
helix_terminator provisions multi-cloud infrastructure (AWS and GCP) including Kubernetes clusters, VPCs, managed databases, load balancers, IAM policies, and DNS. We need an IaC tool that is declarative, stateful, and widely supported by both cloud providers and the operations team.

## Decision
We chose **Terraform** (OpenTofu-compatible syntax) as the primary Infrastructure-as-Code tool. Pulumi is not used for core infrastructure, though it is permitted for ad-hoc cloud SDK scripts.

## Consequences

### Positive
- **Provider ecosystem**: Terraform has first-party providers for every service we use (AWS, GCP, Cloudflare, Kubernetes, Helm, Vault).
- **Team familiarity**: The SRE team has extensive Terraform experience, reducing onboarding time and error rates.
- **State management**: Remote state backends (S3 + DynamoDB, GCS) with locking are well-understood and battle-tested.
- **Plan/apply workflow**: The explicit planning stage catches unintended changes before they reach production.
- **Module registry**: Internal and public modules enable reuse and standardization across environments.

### Negative
- **HCL limitations**: HashiCorp Configuration Language is less expressive than general-purpose languages; complex logic requires workarounds.
- **State file risks**: The state file is sensitive; accidental corruption or exposure can have serious consequences.
- **Drift detection**: Resources modified outside Terraform require periodic `terraform plan` scans or drift-detection automation.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Pulumi** | General-purpose languages (TypeScript, Python, Go) are appealing for logic-heavy infrastructure, but Pulumi’s provider lag, smaller community, and lack of a native plan/apply workflow made it riskier for production. Retained as an option for application-level infrastructure scripts. |
| **AWS CDK / CDKTF** | Cloud-specific; we require multi-cloud consistency. CDKTF is Terraform-backed but adds abstraction layers that complicate debugging. |
| **Ansible** | Imperative and stateless; better for configuration management than infrastructure provisioning. Rejected for core cloud resource management. |
| **CloudFormation / Deployment Manager** | Cloud-specific and vendor-locked; rejected to preserve multi-cloud portability. |

## References
- `infrastructure/terraform/` — All Terraform modules and root configurations
- `infrastructure/terraform/modules/` — Reusable modules
- `docs/guides/runbooks/KEY_ROTATION.md` — Terraform-managed secret rotation
