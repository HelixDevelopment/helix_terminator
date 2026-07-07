# DevOps Challenges

> Infrastructure and operations challenges for HelixTerminator platform engineers.

## Challenge 1: GitOps Deployment Pipeline

**Difficulty:** Hard
**Time:** 6 hours

Build a complete GitOps deployment pipeline:
- ArgoCD or Flux for Kubernetes GitOps
- Automated promotion through environments (dev → staging → prod)
- Canary deployments with Flagger
- Automated rollback on error rate threshold

### Acceptance Criteria
- [ ] All deployments are triggered by Git commits
- [ ] Canary analysis succeeds before full promotion
- [ ] Rollback completes within 60 seconds of threshold breach
- [ ] Git history is the single source of truth for cluster state

---

## Challenge 2: Multi-Region Kubernetes Cluster

**Difficulty:** Hard
**Time:** 8 hours

Design and deploy a multi-region Kubernetes setup:
- Active-active in 3 regions (us-east, eu-west, ap-south)
- Global load balancing with health checks
- Data replication between regions (PostgreSQL, Redis)
- Failover testing and runbook

### Acceptance Criteria
- [ ] Traffic is routed to the nearest healthy region
- [ ] Database replication lag < 1 second under normal load
- [ ] Failover to another region completes in < 5 minutes
- [ ] Runbook is tested and executable by on-call engineer

---

## Challenge 3: Infrastructure as Code Testing

**Difficulty:** Medium
**Time:** 3 hours

Implement comprehensive IaC testing:
- Terraform plan validation in CI
- OPA/Rego policy checks
- Cost estimation with Infracost
- Security scanning with Checkov

### Acceptance Criteria
- [ ] Every PR runs `terraform plan` and posts results
- [ ] Policy violations block merge
- [ ] Cost delta is reported for infrastructure changes
- [ ] Security findings are tracked to resolution

---

## Challenge 4: Observability Stack

**Difficulty:** Medium
**Time:** 4 hours

Deploy a complete observability stack:
- Prometheus + Grafana for metrics
- Loki for log aggregation
- Jaeger/Tempo for distributed tracing
- Alertmanager with PagerDuty integration

### Acceptance Criteria
- [ ] All services expose metrics in Prometheus format
- [ ] Logs are queryable and correlated with traces
- [ ] Traces span all service boundaries
- [ ] Critical alerts reach PagerDuty within 30 seconds

---

## Challenge 5: Secret Management

**Difficulty:** Hard
**Time:** 4 hours

Implement a secret management solution:
- HashiCorp Vault or AWS Secrets Manager
- Automatic secret rotation
- Kubernetes external-secrets operator
- Audit logging of all secret access

### Acceptance Criteria
- [ ] No secrets in Git or container images
- [ ] Rotation occurs without service disruption
- [ ] Pods receive secrets via volume mounts or env vars
- [ ] Audit log captures who accessed what and when

---

## Challenge 6: Disaster Recovery Automation

**Difficulty:** Hard
**Time:** 6 hours

Automate disaster recovery procedures:
- Daily backups of all stateful services
- Automated restore testing in isolated environment
- RPO < 1 hour, RTO < 30 minutes
- DR runbook generated from automation scripts

### Acceptance Criteria
- [ ] Backups are verified for integrity
- [ ] Restore test runs weekly and reports results
- [ ] RPO and RTO are measurable and documented
- [ ] Runbook is always current with automation

---

## Challenge 7: Cost Optimization

**Difficulty:** Medium
**Time:** 3 hours

Implement cloud cost optimization:
- Right-sizing recommendations based on usage
- Spot instance usage for non-critical workloads
- Automated shutdown of dev environments after hours
- Cost allocation tags and monthly reporting

### Acceptance Criteria
- [ ] Right-sizing recommendations are actionable
- [ ] Spot instance usage is > 50% for eligible workloads
- [ ] Dev environments shut down automatically
- [ ] Monthly report attributes costs to teams/services

---

## Challenge 8: Network Policy Hardening

**Difficulty:** Medium
**Time:** 2 hours

Harden Kubernetes network policies:
- Default deny-all policy
- Explicit allow rules per service
- Egress restrictions (no public internet except via proxy)
- Policy validation in CI

### Acceptance Criteria
- [ ] No pod can communicate without an explicit policy
- [ ] Policies are documented with traffic flow diagrams
- [ ] Egress is restricted to known destinations
- [ ] CI validates policies before deployment

---

## Challenge 9: Certificate Management

**Difficulty:** Medium
**Time:** 2 hours

Automate TLS certificate management:
- cert-manager in Kubernetes
- Let's Encrypt for public endpoints
- Internal CA for service-to-service mTLS
- Certificate expiry monitoring and alerting

### Acceptance Criteria
- [ ] Certificates are automatically issued and renewed
- [ ] mTLS is enforced between all services
- [ ] Expiry alerts fire 30 days before expiration
- [ ] No manual certificate operations required

---

## Challenge 10: Chaos Engineering Platform

**Difficulty:** Hard
**Time:** 5 hours

Build a chaos engineering platform:
- Litmus or Gremlin integration
- Predefined experiment templates (pod kill, network latency, CPU stress)
- Safety checks (auto-abort if error rate exceeds threshold)
- Scheduled experiments with reporting

### Acceptance Criteria
- [ ] Experiments are safe and reversible
- [ ] Auto-abort prevents production impact
- [ ] Reports include blast radius and recovery time
- [ ] Experiments run automatically on a schedule

---

## Submission Guidelines

1. Fork the repository and create a branch: `challenge/<your-name>-<challenge-number>`
2. Include Terraform/Helm/Kubernetes manifests
3. Provide a README with deployment instructions
4. Open a draft PR for review
5. Tag `@helix-devops-reviewers` for feedback

## Scoring

- **Pass:** All acceptance criteria met, manifests validate, docs complete
- **Merit:** Pass + automation is fully hands-off, monitoring included
- **Distinction:** Merit + reusable module or tool contribution
