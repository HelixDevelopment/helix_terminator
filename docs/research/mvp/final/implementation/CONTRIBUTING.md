# Contributing to HelixTerminator

**Version:** 1.0.0  
**Date:** 2026-07-05  
**Org:** HelixDevelopment

---

## Getting Started

1. **Clone the repository:**
   ```bash
   git clone --recursive git@github.com:HelixDevelopment/helix_terminator.git
   cd helix_terminator
   ```

2. **Verify the constitution submodule:**
   ```bash
   bash tests/verify_constitution_inheritance.sh
   ```

3. **Read the development kickoff:**
   See `DEVELOPMENT_KICKOFF.md` for environment setup and quick start.

---

## Development Workflow

### Branch Naming

| Prefix | Purpose | Example |
|--------|---------|---------|
| `feature/` | New capability | `feature/ssh-cert-auth` |
| `bugfix/` | Bug fix | `bugfix/vault-sync-race` |
| `hotfix/` | Production patch | `hotfix/auth-mfa-bypass` |
| `docs/` | Documentation only | `docs/api-examples` |
| `chore/` | Maintenance | `chore/dependency-update` |

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `security`

### Pull Request Process

1. **Before opening a PR:**
   - Run `bash tests/verify_constitution_inheritance.sh` — must pass
   - Run `bash tests/docs_consistency_gate.sh` — must pass
   - Run `make test` — all services must pass
   - Run `make lint` — no new lint errors

2. **PR template:**
   - Link to related issue
   - Describe the change and why
   - List verification commands run
   - Attach test output or screenshots

3. **Review requirements:**
   - Minimum 1 approval from code owner
   - All CI checks green
   - No merge conflicts
   - Constitution inheritance gate passes

4. **Merge strategy:**
   - Squash and merge to `main`
   - Delete feature branch after merge

---

## Code Standards

### Go

- **Go version:** 1.25
- **Module path:** `helixterminator.io/services/<name>`
- **Framework:** Gin Gonic
- **DB driver:** pgx/v5
- **Test coverage:** ≥80% (auth/vault ≥90%, crypto 100%)
- **Linting:** golangci-lint with `.golangci.yml`

### Flutter/Dart

- **Flutter version:** 3.24
- **State management:** flutter_bloc
- **HTTP client:** Dio with interceptors
- **Local DB:** drift
- **Routing:** go_router

### Infrastructure

- **Terraform:** 1.9.0+
- **Kubernetes:** 1.31
- **Helm:** 3.15.0+
- **Docker:** Multi-stage, distroless base

---

## Service Development

### Adding a New Service

1. Copy the scaffold from `services/gateway-service/`
2. Update `go.mod` module name to `helixterminator.io/services/<name>`
3. Implement `model.go`, `repository.go`, `handler.go`, `server.go`, `main.go`
4. Write tests for all packages (≥80% coverage)
5. Add migration SQL in `migrations/001_init.sql`
6. Add to Docker Compose and Helm values
7. Add to CI/CD matrix in `.github/workflows/pr.yml`
8. Update `SERVICE_REGISTRY.md` and `helix-deps.yaml`
9. Add OpenAPI spec in `docs/research/mvp/final/implementation/api/<name>.yaml`
10. Add service-specific README in `services/<name>/README.md`

### API Standards

- Base path: `/api/v1/`
- Health: `GET /healthz` (200 = healthy)
- Readiness: `GET /healthz/ready` (200 = ready, 503 = not ready)
- Content-Type: `application/json`
- Authentication: Bearer token in `Authorization` header
- Request ID: `X-Request-ID` header (UUID)
- Rate Limiting: `X-RateLimit-*` headers

---

## Testing

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

## Security

- Never commit secrets to Git (use environment variables)
- All database queries use parameterized statements
- Validate all inputs with Gin binding tags
- Use `uuid.UUID` for all IDs (no sequential IDs)
- Log all authentication failures
- Rate limit all public endpoints
- All images use distroless base (no shell, minimal attack surface)
- Run as non-root user (`nonroot:nonroot`)
- Read-only root filesystem
- Drop all capabilities
- Scan with Trivy before deployment

---

## Documentation

- Update docs when changing behavior
- Keep `SERVICE_REGISTRY.md` current
- Add ADRs for architectural decisions
- Write service-specific READMEs (not stubs)
- Update runbooks when procedures change

---

## Support

| Issue Type | Contact | Response Time |
|-----------|---------|--------------|
| Development questions | #dev-helixterminator Slack | 4 hours |
| Production incidents | #incidents PagerDuty | 15 minutes |
| Security concerns | security@helixdevelopment.io | 1 hour |
| Infrastructure issues | #sre-ops Slack | 2 hours |

---

*HelixTerminator Contribution Guidelines*  
*All contributions must pass constitution inheritance and docs consistency gates*
