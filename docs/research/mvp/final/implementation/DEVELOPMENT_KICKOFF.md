# HelixTerminator — Development Kick-Off Document

> **Status**: READY FOR DEVELOPMENT TEAM KICK-OFF
> **Date**: 2026-07-05
> **Version**: 1.0.0
> **Repository**: `github.com:HelixDevelopment/helix_terminator.git`

---

## Executive Summary

The HelixTerminator platform is a **25-service Go microservices architecture** with a **Flutter cross-platform client**, designed for enterprise-grade remote infrastructure management. All backend services, frontend screens, infrastructure manifests, security policies, and CI/CD pipelines are implemented and ready for development team onboarding.

**Key Metrics**:
- 25 Go microservices — all build, all test (426 tests)
- 32 Flutter screens, 278 widgets, 19 BLoCs, 17 services
- Production-hardened Terraform/K8s/Helm infrastructure
- 15-phase CI/CD pipeline from commit to production
- Zero-trust security with Falco, Cosign, Trivy, network policies

---

## Architecture Overview

### Service Mesh (25 Microservices)

| Stream | Services | Purpose |
|--------|----------|---------|
| **A — Core** | Gateway, Auth, User, Vault, Host, SSH-Proxy, Terminal | Authentication, host management, terminal access |
| **B — Workspace** | Keychain, Workspace, Config, PKI, Billing | SSH keys, workspaces, configuration, billing |
| **C — Organization** | Org, Notification, Audit, Health, AI | Teams, notifications, audit logs, health checks |
| **D — Integration** | Collaboration, Container-Bridge, HelixTrack-Bridge, Port-Forward | Real-time collaboration, container management |
| **E — Operations** | SFTP, Recording, Snippet, Analytics | File transfers, session recording, snippets, analytics |

### Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend | Go | 1.22 |
| Web Framework | Gin | 1.10.0 |
| Database | PostgreSQL | 17.2 |
| DB Driver | pgx/v5 | 5.6.0 |
| Cache | Redis | 8.0.0 |
| Messaging | Kafka | 3.9 |
| Frontend | Flutter | 3.24.0 |
| State Management | flutter_bloc | 8.1.6 |
| Container Runtime | Docker + distroless | latest |
| Orchestration | Kubernetes (EKS) | 1.31 |
| IaC | Terraform | 1.9.0 |
| Package Manager | Helm | 3.15.0 |

---

## Development Environment Setup

### Prerequisites

```bash
# Go 1.22+
# Docker 24.0+
# Docker Compose 2.20+
# Flutter 3.24.0+ (for client development)
# kubectl 1.31+ (for K8s operations)
# Terraform 1.9.0+ (for infrastructure)
# Helm 3.15.0+ (for package management)
```

### Quick Start

```bash
# 1. Clone the repository
git clone git@github.com:HelixDevelopment/helix_terminator.git
cd helix_terminator

# 2. Start local infrastructure
cd infrastructure/docker/compose
docker-compose up -d postgres redis kafka

# 3. Build and test a service
cd services/auth-service
go mod tidy
go build ./...
go test -v -cover ./...

# 4. Run the Flutter client
cd clients/flutter
flutter pub get
flutter run
```

---

## Service Development Guide

### Service Structure (Standard Layout)

```
services/{service-name}/
├── cmd/{service-name}/
│   ├── main.go              # Entry point
│   └── main_test.go         # Main tests
├── internal/
│   ├── model/
│   │   ├── model.go         # Domain models
│   │   └── model_test.go    # Model tests
│   ├── repository/
│   │   ├── repository.go    # Database access (pgx/v5)
│   │   └── repository_test.go
│   ├── handler/
│   │   ├── handler.go       # HTTP handlers (Gin)
│   │   └── handler_test.go
│   └── server/
│       ├── server.go         # HTTP server setup
│       └── server_test.go
├── migrations/
│   └── 001_init.sql         # Database schema
├── Dockerfile               # Multi-stage build
└── go.mod                   # Module definition
```

### Adding a New Service

1. Copy the scaffold from `services/gateway-service/`
2. Update `go.mod` module name
3. Implement `model.go`, `repository.go`, `handler.go`, `server.go`, `main.go`
4. Write tests for all packages
5. Add migration SQL
6. Add to Docker Compose and Helm values
7. Add to CI/CD matrix in `.github/workflows/pr.yml`

### API Standards

- Base path: `/api/v1/`
- Health: `GET /healthz` (200 = healthy)
- Readiness: `GET /healthz/ready` (200 = ready, 503 = not ready)
- Content-Type: `application/json`
- Authentication: Bearer token in `Authorization` header
- Request ID: `X-Request-ID` header (UUID)
- Rate Limiting: `X-RateLimit-*` headers

### Response Format

```json
{
  "data": { ... },
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

### Error Format

```json
{
  "error": "invalid_request",
  "message": "The request body is malformed",
  "code": 400
}
```

---

## Flutter Client Development

### Project Structure

```
clients/flutter/lib/
├── main.dart                  # App entry point
├── bloc/                      # BLoC state management (19 files)
├── models/                    # Data models (14 files)
├── screens/                   # UI screens (32 files)
├── services/                  # API clients (17 files)
├── widgets/                   # Reusable widgets (278 files)
└── themes/                    # Light/dark themes
```

### Key Patterns

- **BLoC**: Every screen has a dedicated BLoC with Events, States, and emitters
- **RepositoryProvider**: Services injected at app root
- **Material 3**: Uses `Theme.of(context).colorScheme`
- **Responsive**: `LayoutBuilder`, `GridView`, `ConstrainedBox`
- **Error Handling**: `ErrorWidget` with retry callback
- **Loading States**: `CircularProgressIndicator` during async ops

### Adding a New Screen

1. Create screen in `lib/screens/`
2. Create BLoC in `lib/bloc/`
3. Create service in `lib/services/` (if new API needed)
4. Add route in `main.dart`
5. Add to bottom navigation if primary screen

---

## Infrastructure Development

### Terraform Modules

```
infrastructure/terraform/modules/
├── vpc/           # VPC, subnets, NAT gateways, security groups
├── eks/           # EKS cluster, node groups, IAM roles
├── rds/           # PostgreSQL, Multi-AZ, encryption, backups
├── iam/           # IAM policies, roles
├── elasticache/   # Redis clusters
└── msk/           # Kafka clusters
```

### Kubernetes Manifests

```
infrastructure/kubernetes/
├── base/
│   ├── namespace.yaml         # helixterminator namespace (PSS restricted)
│   ├── service-account.yaml   # IRSA-compatible SA
│   ├── network-policy.yaml    # Default-deny + allow rules
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    ├── staging/
    └── production/
```

### Helm Chart

```
infrastructure/helm/helixterm/
├── Chart.yaml                 # Dependencies: PostgreSQL, Redis, Kafka, ingress-nginx
├── values.yaml                # All 25 services + infrastructure config
└── templates/                  # Deployment, Service, Ingress, HPA, PDB
```

### Deploying to Staging

```bash
# 1. Build and push images
make docker-build
make docker-push REGISTRY=ghcr.io/helixdevelopment

# 2. Deploy infrastructure
cd infrastructure/terraform/environments/staging
terraform init
terraform apply

# 3. Deploy application
cd infrastructure/helm/helixterm
helm upgrade --install helixterm . -f values-staging.yaml

# 4. Verify
kubectl get pods -n helixterminator
kubectl get svc -n helixterminator
```

---

## Security Guidelines

### Development

- Never commit secrets to Git (use environment variables)
- All database queries use parameterized statements (pgx handles this)
- Validate all inputs with Gin binding tags
- Use `uuid.UUID` for all IDs (no sequential IDs)
- Log all authentication failures
- Rate limit all public endpoints

### Container Security

- All images use distroless base (no shell, minimal attack surface)
- Run as non-root user (`nonroot:nonroot`)
- Read-only root filesystem
- Drop all capabilities
- Scan with Trivy before deployment

### Kubernetes Security

- Pod Security Standards: **restricted**
- Network Policies: default-deny + explicit allows
- Service Accounts: automount disabled, IRSA for AWS
- Secrets: encrypted at rest, external secret management for production

### CI/CD Security

- Trivy vulnerability scan (CRITICAL/HIGH)
- govulncheck for Go vulnerabilities
- TruffleHog for secret detection
- Cosign image signing
- OWASP ZAP baseline scan (scheduled)

---

## Testing Strategy

### Test Pyramid

| Level | Type | Count | Tools |
|-------|------|-------|-------|
| Unit | Go service tests | 426 | go test, testify |
| Contract | API contract tests | 22 | go test, JSON validation |
| Integration | Cross-service tests | 16 | go test, PostgreSQL, Redis |
| E2E | Full user flows | TBD | Flutter integration_test |
| Performance | Load tests | TBD | k6 |
| Security | Vulnerability scans | Continuous | Trivy, ZAP, govulncheck |

### Running Tests

```bash
# Unit tests for a service
cd services/auth-service
go test -v -race -cover ./...

# All services
cd /home/milos/Factory/projects/tools_and_research/helix_terminator
make test

# Contract tests
cd test/contracts
go test -v ./...

# Integration tests
cd test/integration
go test -v ./...

# Flutter tests
cd clients/flutter
flutter test
```

---

## CI/CD Pipeline

### Pull Request Flow

1. **Pre-build**: Constitution inheritance, docs consistency
2. **Lint**: golangci-lint, go vet
3. **Unit Tests**: All 25 services in parallel matrix
4. **Flutter**: `flutter analyze`, `flutter test`
5. **Terraform**: `terraform fmt`, `terraform validate`
6. **Helm**: `helm lint`, `helm template`

### Main Branch Flow

1. All PR checks
2. **Contract Tests**: API structure validation
3. **Integration Tests**: With PostgreSQL + Redis
4. **Security Scan**: Trivy, govulncheck, TruffleHog
5. **Docker Build**: Multi-arch (amd64/arm64), signed with Cosign
6. **Flutter Build**: Web + APK
7. **Performance Tests**: k6 load tests
8. **Deploy Staging**: Helm upgrade
9. **Smoke Tests**: Health endpoint verification
10. **Deploy Production**: Manual approval required

### Scheduled Jobs

- **Nightly**: OWASP ZAP baseline scan
- **Weekly**: Dependency update check

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

## Development Team Onboarding Checklist

### Week 1: Environment Setup

- [ ] Clone repository and verify build
- [ ] Set up local Docker Compose environment
- [ ] Run all service tests successfully
- [ ] Run Flutter client locally
- [ ] Review architecture documentation
- [ ] Complete security training

### Week 2: First Contribution

- [ ] Pick up a "good first issue" from backlog
- [ ] Implement feature with tests
- [ ] Submit PR and pass all checks
- [ ] Deploy to staging and verify
- [ ] Participate in code review

### Week 3: Full Ownership

- [ ] Take ownership of one service
- [ ] Understand service dependencies
- [ ] Add monitoring and alerting
- [ ] Document service-specific runbook
- [ ] On-call rotation readiness

---

## Support & Escalation

| Issue Type | Contact | Response Time |
|-----------|---------|--------------|
| Development questions | #dev-helixterminator Slack | 4 hours |
| Production incidents | #incidents PagerDuty | 15 minutes |
| Security concerns | security@helixdevelopment.io | 1 hour |
| Infrastructure issues | #sre-ops Slack | 2 hours |

---

## Appendix: Quick Reference

### Service Ports

| Service | Port | Health | Ready |
|---------|------|--------|-------|
| Gateway | 8080 | /healthz | /healthz/ready |
| Auth | 8080 | /healthz | /healthz/ready |
| ... | ... | ... | ... |

### Database Schemas

Each service has its own schema in PostgreSQL:
- `auth`: users, sessions, mfa
- `host`: hosts, host_tags, host_credentials
- `vault`: secrets, secret_versions
- ... (see each service's `migrations/001_init.sql`)

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | No | Redis connection string |
| `KAFKA_BROKERS` | No | Kafka bootstrap servers |
| `PORT` | No | HTTP port (default: 8080) |
| `LOG_LEVEL` | No | debug/info/warn/error (default: info) |
| `JWT_SECRET` | Yes | JWT signing key (Auth service) |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-07-05 | HelixDevelopment | Initial kick-off document |

---

**END OF DOCUMENT**

> This document is a living document. Update it as the platform evolves.
> For questions, contact the platform team at platform@helixdevelopment.io
