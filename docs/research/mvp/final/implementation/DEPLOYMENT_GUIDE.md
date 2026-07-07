# HelixTerminator — Deployment Guide

**Version:** 1.0.0
**Status:** Draft
**Date:** 2026-07-05
**Authority:** `CANONICAL_FACTS.md` (CD-4, CD-5, CD-6) + `SERVICE_REGISTRY.md`

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Backend services |
| Docker | 24.0+ | Container builds |
| Docker Compose | 2.20+ | Local development |
| kubectl | 1.31+ | Kubernetes operations |
| Terraform | 1.9.0+ | Infrastructure provisioning |
| Helm | 3.15.0+ | Package management |
| Flutter | 3.24.0+ | Client development |
| AWS CLI | 2.x+ | Cloud operations |

---

## Environments

| Environment | Purpose | Region | Auto-deploy |
|-------------|---------|--------|-------------|
| Local | Development | localhost | N/A |
| Dev | Feature testing | us-east-1 | On PR merge |
| Staging | Integration testing | us-east-1 | On main merge |
| Production | Live traffic | us-east-1 | Manual approval |
| DR | Disaster recovery | eu-west-1 | Manual failover |

---

## Local Development

### 1. Clone and Setup

```bash
git clone --recursive git@github.com:HelixDevelopment/helix_terminator.git
cd helix_terminator
bash tests/verify_constitution_inheritance.sh
```

### 2. Start Local Infrastructure

```bash
cd infrastructure/docker/compose
docker-compose up -d postgres redis kafka rabbitmq
```

### 3. Build and Test a Service

```bash
cd services/auth-service
go mod tidy
go build ./...
go test -v -cover ./...
```

### 4. Run the Flutter Client

```bash
cd clients/flutter
flutter pub get
flutter run
```

---

## Staging Deployment

### 1. Build and Push Images

```bash
make docker-build
make docker-push REGISTRY=ghcr.io/helixdevelopment
```

### 2. Deploy Infrastructure

```bash
cd infrastructure/terraform/environments/staging
terraform init
terraform plan
terraform apply
```

### 3. Deploy Application

```bash
cd infrastructure/helm/helixterm
helm upgrade --install helixterm . -f values-staging.yaml
```

### 4. Verify

```bash
kubectl get pods -n helixterminator
kubectl get svc -n helixterminator
kubectl get ingress -n helixterminator
```

---

## Production Deployment

### 1. Pre-deployment Checklist

- [ ] All staging tests pass
- [ ] Security scan clean (Trivy, govulncheck)
- [ ] Constitution inheritance gate passes
- [ ] Docs consistency gate passes
- [ ] Database migrations reviewed
- [ ] Rollback plan documented

### 2. Canary Deployment

```bash
# Deploy canary (10% traffic)
helm upgrade --install helixterm . \
  -f values-production.yaml \
  -f values-canary.yaml \
  --set canary.enabled=true \
  --set canary.weight=10

# Monitor for 15 minutes
kubectl logs -n helixterminator -l app=helixterm-canary

# Gradually increase traffic
helm upgrade helixterm . \
  -f values-production.yaml \
  -f values-canary.yaml \
  --set canary.enabled=true \
  --set canary.weight=50

# Full rollout
helm upgrade helixterm . -f values-production.yaml
```

### 3. Post-deployment Verification

```bash
# Health checks
for svc in gateway auth user vault host ssh-proxy terminal; do
  curl -sf https://api.helixterminator.io/healthz || echo "FAIL: $svc"
done

# Smoke tests
cd test/e2e
go test -v -run TestSmoke

# Alert check
kubectl get pods -n helixterminator
```

---

## DR Failover

### Trigger Conditions

- Primary region unavailable for >5 minutes
- Data corruption in primary database
- Security incident requiring isolation

### Failover Procedure

```bash
# 1. Promote DR database
cd infrastructure/terraform/environments/dr
terraform apply -var="failover=true"

# 2. Update DNS to point to DR
aws route53 change-resource-record-sets \
  --hosted-zone-id Z123456789 \
  --change-batch file://dr-failover.json

# 3. Verify DR services
kubectl --context=dr get pods -n helixterminator

# 4. Notify stakeholders
# Use incident-response runbook
```

### Rollback

```bash
# 1. Restore DNS to primary
aws route53 change-resource-record-sets \
  --hosted-zone-id Z123456789 \
  --change-batch file://primary-restore.json

# 2. Verify primary
kubectl --context=primary get pods -n helixterminator
```

---

## Rollback Procedures

### Helm Rollback

```bash
# List releases
helm history helixterm -n helixterminator

# Rollback to previous revision
helm rollback helixterm <revision> -n helixterminator
```

### Database Rollback

```bash
# Restore from backup
pg_restore --host=$DB_HOST --dbname=helixterm_auth backup.sql

# Or use PITR (see runbook)
# See docs/runbooks/postgres-pitr-restore.md
```

---

## Monitoring & Observability

### Metrics (Prometheus)

- Request rate, latency, errors (RED metrics)
- Database connection pool stats
- Cache hit/miss rates
- Message queue lag

### Tracing (Jaeger)

- Distributed tracing across all 25 services
- Request correlation via `X-Request-ID`
- Performance bottleneck identification

### Logging (Loki)

- Structured JSON logs
- Log levels: debug, info, warn, error
- Sensitive data redaction

### Alerting (PagerDuty)

- High error rate (>1% for 5 min)
- High latency (p99 > 500ms for 5 min)
- Service down (health check fails for 2 min)
- Database connection failures
- Security events (Falco alerts)

---

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Pods stuck Pending | Resource limits | Check node capacity, scale cluster |
| ImagePullBackOff | Registry auth | Verify image pull secrets |
| CrashLoopBackOff | App error | Check logs, fix config |
| High latency | DB connection pool | Scale DB connections, check query performance |
| Kafka lag | Consumer slow | Scale consumer replicas |

See `TROUBLESHOOTING.md` for detailed troubleshooting steps.

---

*HelixTerminator Deployment Guide*
*Consolidated from: 08-devops-infrastructure/README.md, DEVELOPMENT_KICKOFF.md*
