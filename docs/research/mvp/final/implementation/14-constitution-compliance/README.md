# 14 — Constitution Compliance

**Status:** `Complete`  
**Module:** A + B (product-family-wide governance)  
**Authority:** `CANONICAL_FACTS.md` (CD-9)  

---

## Overview

HelixTerminator unconditionally inherits every clause of the HelixConstitution. The constitution is included as a Git submodule pinned to commit `e6504c2` (`git describe` = `helixcode-v1.1.0-39-ge6504c2`), cited as **"HelixConstitution (pinned e6504c2, helixcode-v1.1.0 line)"**.

**Governance layers:**

```
Layer 1 (BASE)     constitution/Constitution.md   — universal rules, all projects
Layer 2 (PROJECT)  HelixTerminator/Constitution.md — HelixTerminator-specific rules
Layer 3 (SUBDIR)   <subdir>/CLAUDE.md             — module-local overrides (optional)
```

---

## Verbatim Adoption Commitment

HelixTerminator's root `Constitution.md` contains:

```markdown
This constitution EXTENDS the Helix Universal Constitution at
`constitution/Constitution.md`. All clauses there apply unless
explicitly overridden below with an explicit `Override §X.Y`
section. There are NO overrides in HelixTerminator — all universal
clauses apply at full strength.
```

---

## HelixTerminator-Specific Extensions (HT-001..HT-010)

| Extension | Rule |
|-----------|------|
| HT-001 | All 25 microservices MUST be registered in `helix-deps.yaml` |
| HT-002 | Go version MUST be 1.25 or higher in all `go.mod` files |
| HT-003 | Flutter client MUST target iOS 17+ and Android 14+ |
| HT-004 | All Kafka topics MUST follow `helix.terminator.<domain>.<event>` format |
| HT-005 | All databases MUST use `helixterm_<service>_db` naming |
| HT-006 | Service-to-service calls MUST use gRPC with protocol buffers |
| HT-007 | All HTTP APIs MUST serve OpenAPI 3.1 spec at `/openapi.json` |
| HT-008 | Every service MUST expose `/health/live` and `/health/ready` endpoints |
| HT-009 | Circuit breakers MUST be configured with `vasic-digital/recovery` submodule |
| HT-010 | All structured logs MUST use `vasic-digital/observability` submodule |

---

## Mandatory Test Types (12)

Per CD-12, the canonical 12 test types are:

1. Unit Tests
2. Integration Tests
3. E2E Tests
4. Performance / Load Tests
5. Security Tests
6. Contract Tests
7. Chaos Engineering
8. Mutation Testing
9. Accessibility Tests
10. Device/Topology Matrix Tests
11. Static Analysis
12. Compliance / Constitution Gates

---

## CI/CD Constitution Compliance Gates

### PR Pipeline Gates

1. **Anti-Bluff Audit:** `scripts/audit_antibluff.sh`
   - §11.4 anchor present across all submodules
   - Default test suite green
   - Paired §1.1 meta-test green
2. **Submodule Catalogue Check:** `submodules-catalogue.md` consistency
3. **Container Mandate:** All containerized workloads use `vasic-digital/containers`
4. **CodeGraph Integration:** Cross-service impact analysis via `@colbymchenry/codegraph`
5. **Naming Convention:** Lowercase snake_case for all directories, files, submodules
6. **Dependency Manifest:** `helix-deps.yaml` present and valid

### Release Pipeline Gates

1. All PR gates, plus:
2. **Security Scan:** SAST, DAST, dependency audit
3. **Performance Baseline:** No regression > 5% from established SLOs
4. **Compliance Score:** ≥ 95% on constitution compliance dashboard

---

## Anti-Patterns and Forbidden Patterns

| # | Pattern | Why Forbidden | Detection |
|---|---------|-------------|-----------|
| 1 | Secrets in git | `.env` files must be git-ignored | `git-secrets` pre-commit hook |
| 2 | `:latest` tags in prod | Non-reproducible deployments | Helm lint, CI gate |
| 3 | `--no-verify` bypass | Circumvents all gates | Server-side branch protection |
| 4 | Direct `pool.Query` against RLS DB | Bypasses tenant isolation | `go vet` custom analyzer |
| 5 | Unpinned submodule versions | Drift risk, supply chain | `go mod verify` |
| 6 | `digital.vasic.*` dot-path imports | Cannot co-compile with slash-path | `golangci-lint` custom rule |
| 7 | Missing `helix-deps.yaml` | No dependency visibility | CI gate |
| 8 | Hardcoded timeouts | No circuit breaker | `gosec` + review |
| 9 | Plaintext PII in logs | GDPR violation | Log scrubber + audit |
| 10 | Missing `FORCE ROW LEVEL SECURITY` | RLS bypassable by table owner | `rls_test.go` post-build test |

---

## Code Review Checklist (Constitution-Mandated)

- [ ] Tests present and green (with runtime evidence)
- [ ] Docs updated in same commit (§11.4.65)
- [ ] Submodule catalogue updated if new dependency added
- [ ] `helix-deps.yaml` updated if new dependency added
- [ ] RLS policies added for new multi-tenant tables
- [ ] Anti-bluff audit passes (`scripts/audit_antibluff.sh`)
- [ ] No secrets in diff (`git-secrets`)
- [ ] Version pins match CANONICAL_FACTS.md (CD-4)
- [ ] Go module path uses canonical `helixterminator.io/services/<name>`
- [ ] Kafka topic naming follows `helix.terminator.<domain>.<event>`

---

## Repository Structure Compliance

```
helix-terminator/
├── constitution/          # HelixConstitution submodule (pinned e6504c2)
├── submodules/            # Platform library submodules
├── services/              # 25 Go microservices
├── clients/flutter/       # Flutter client
├── infrastructure/        # K8s, Helm, Terraform
├── scripts/               # Dev, testing, DR scripts
├── docs/                  # All documentation
├── .github/workflows/     # CI/CD pipelines
├── submodules-catalogue.md
├── helix-deps.yaml
├── Constitution.md        # Project-specific constitution
├── CLAUDE.md              # Agent instructions
├── AGENTS.md              # Agent rules
└── CHANGELOG.md           # Release notes
```

---

## Cross-References

- [07 — Testing Strategy](../07-testing-strategy/) — Test types, coverage targets, CI gates
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — CI/CD pipeline definitions
- [13 — Submodule Integration](../13-submodule-integration/) — Submodule catalogue, versions
- [16 — References](../16-references/) — CD-9 (constitution version), CD-12 (test count)

---

*Section 14 — Constitution Compliance*  
*Consolidated from: 11_constitution_compliance.md, CANONICAL_FACTS.md (CD-9, CD-12)*
