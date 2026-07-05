# 07 — Testing Strategy

**Status:** `Complete`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-12: 12 test types) + `SERVICE_REGISTRY.md`  

---

## Overview

HelixTerminator is a security-critical application. Testing is the primary mechanism by which we prove the software deserves user trust. The HelixConstitution establishes the base-layer rule set; HelixTerminator extends it with service-specific coverage targets.

**Test Type Count:** 12 mandatory test types (canonical per CD-12; doc 03 previously claimed 17 — reconciled to 12).

---

## Testing Philosophy

### Anti-Bluff Covenant (§11.4)

The bar for shipping is not "tests pass." The bar is "users can use the feature." Every test that emits a PASS signal **must carry positive runtime evidence** captured during execution. A green summary line without runtime evidence is a critical defect.

### Five-Layer Git-Hook Discipline (§11.4.75)

No commit reaches `main` or any release branch without all required gates passing.

### Coverage Targets

| Component | Line Coverage | Branch Coverage | Mutation Score |
|-----------|--------------|-----------------|----------------|
| Auth Service | ≥ 90% | ≥ 95% | ≥ 90% |
| Vault Service | ≥ 90% | ≥ 95% | ≥ 90% |
| SSH Proxy Service | ≥ 85% | ≥ 90% | ≥ 85% |
| Keychain Service | ≥ 90% | ≥ 95% | ≥ 90% |
| Host Service | ≥ 80% | ≥ 90% | ≥ 80% |
| API Gateway | ≥ 80% | ≥ 88% | ≥ 80% |
| Audit Service | ≥ 80% | ≥ 85% | ≥ 75% |
| Flutter Clients | ≥ 80% | ≥ 85% | N/A |
| **Global Floor** | **≥ 80%** | **≥ 90% (critical paths)** | **≥ 80%** |

"Critical path" = authentication state transitions, vault cryptographic operations, SSH host-key verification, RBAC privilege enforcement. These require 100% mutation kill rate.

---

## Test Pyramid

```
                    ┌──────────────┐
                    │     E2E      │  10%  (~300 tests)
                    │  (Flutter +  │
                    │  backend)    │
                  ┌─┴──────────────┴─┐
                  │   Integration    │  20%  (~600 tests)
                  │ (DB, Kafka,      │
                  │  RabbitMQ,Redis) │
              ┌───┴──────────────────┴───┐
              │        Unit Tests        │  70%  (~2100 tests)
              │  (Go services, Flutter   │
              │   widgets, repositories) │
              └──────────────────────────┘
```

The CI PR pipeline fails if the ratio deviates by > 5 percentage points from the target.

---

## 12 Mandatory Test Types

| # | Test Type | Framework | Scope | Frequency |
|---|-----------|-----------|-------|-----------|
| 1 | Unit Tests | Go `testing` / `flutter_test` | Individual functions, methods, widgets | Every PR |
| 2 | Integration Tests | Go `testcontainers` / `flutter integration_test` | DB, Kafka, RabbitMQ, Redis seams | Every PR |
| 3 | E2E Tests | `flutter integration_test` + k6 | Full user journeys | Every PR |
| 4 | Performance / Load Tests | k6, `go test -bench` | Throughput, latency, resource usage | Nightly |
| 5 | Security Tests | `gosec`, `zap`, `nuclei`, custom fuzz | Vulnerability scanning, penetration | Nightly |
| 6 | Contract Tests | Pact | Consumer-driven API contracts | Every PR |
| 7 | Chaos Engineering | Litmus, custom fault injection | Resilience under failure | Weekly |
| 8 | Mutation Testing | `gremlins-go` | Test quality (kill rate) | Weekly |
| 9 | Accessibility Tests | `flutter_test` a11y matchers | WCAG 2.1 AA compliance | Every PR |
| 10 | Device/Topology Matrix | Firebase Test Lab, BrowserStack | iOS/Android/desktop OS versions | Weekly |
| 11 | Static Analysis | `golangci-lint`, `dart analyze`, SonarQube | Code quality, security | Every PR |
| 12 | Compliance / Constitution Gates | `scripts/audit_antibluff.sh` | §11.4 anchor, meta-test | Every PR |

---

## CI/CD Pipeline Gates

### PR Pipeline

1. **Pre-build:** lint, format, `go mod verify`, `go vet`, SAST
2. **Build:** compile all services, Flutter build for all 6 platforms
3. **Unit Tests:** all Go + Flutter unit tests
4. **Integration Tests:** DB (Testcontainers), Kafka, Redis
5. **Contract Tests:** Pact verification
6. **E2E Smoke:** critical path (login → connect → command → logout)
7. **Anti-Bluff Audit:** `scripts/audit_antibluff.sh`

### Nightly Pipeline

1. Full E2E suite (all user journeys)
2. Performance / load tests (k6)
3. Security scans (SAST, DAST, dependency audit)
4. Chaos engineering (random pod kills, network partitions)
5. Mutation testing (gremlins)

---

## Test Data Management

- **Seeded fixtures:** `internal/testdata/` per service
- **Factory pattern:** `go-factory` for deterministic test data generation
- **No production data in tests:** ever
- **PII scrubbing:** all test data uses synthetic identities

---

## Cross-References

- [03 — Service Catalog](../03-service-catalog/) — Per-service coverage targets
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — CI/CD pipeline definitions
- [09 — Security — Zero Trust](../09-security-zero-trust/) — Security test requirements
- [11 — Performance Analysis](../11-performance-analysis/) — SLOs and load test benchmarks
- [16 — References](../16-references/) — Canonical test-type count (CD-12)

---

*Section 07 — Testing Strategy*  
*Consolidated from: 03_testing_strategy.md, CANONICAL_FACTS.md (CD-12)*
