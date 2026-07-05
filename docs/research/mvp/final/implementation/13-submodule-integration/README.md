# 13 — Submodule Integration

**Status:** `Draft`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-9, CD-11) + `SERVICE_REGISTRY.md`

---

## Overview

HelixTerminator integrates 17 submodules from three organizations: `vasic-digital`, `HelixDevelopment`, and `Helix-Track`. These shared libraries provide container runtime abstraction, security primitives, authentication, documentation chain-of-custody, and AI-driven QA.

| Org | Submodules | Purpose |
|-----|-----------|---------|
| `vasic-digital` | 8 | Containers, security, auth, docs_chain, recovery, observability, middleware, challenges |
| `HelixDevelopment` | 6 | HelixConstitution, helixqa, helixtrack, open-design, codegraph, analytics |
| `Helix-Track` | 3 | Core integration, OAuth2 bridge, deployment sync |

---

## Submodule Catalogue

| # | Name | Org | Module Path | Version | Used By |
|---|------|-----|-------------|---------|---------|
| 1 | `containers` | vasic-digital | `digital.vasic/containers` | v2.1.0 | Container Bridge, SSH Proxy |
| 2 | `security` | vasic-digital | `digital.vasic/security` | v1.8.0 | All services |
| 3 | `auth` | vasic-digital | `digital.vasic/auth` | v1.5.0 | Auth Service |
| 4 | `docs_chain` | vasic-digital | `digital.vasic/docs_chain` | v1.2.0 | Documentation pipeline |
| 5 | `recovery` | vasic-digital | `digital.vasic/recovery` | v1.3.0 | Circuit breakers, retry |
| 6 | `observability` | vasic-digital | `digital.vasic/observability` | v1.4.0 | Logging, metrics, tracing |
| 7 | `middleware` | vasic-digital | `digital.vasic/middleware` | v0.2.4 | API Gateway |
| 8 | `challenges` | vasic-digital | `digital.vasic/challenges` | v1.0.0 | AI Service, User Service |
| 9 | `HelixConstitution` | HelixDevelopment | `constitution/` | e6504c2 | All (governance) |
| 10 | `helixqa` | HelixDevelopment | `qa/helixqa` | v1.1.0 | Testing pipeline |
| 11 | `helixtrack` | HelixDevelopment | `integrations/helixtrack` | v1.0.0 | HelixTrack Bridge |
| 12 | `open-design` | HelixDevelopment | `design/open-design` | v2.0.0 | UX Design System |
| 13 | `codegraph` | HelixDevelopment | `tools/codegraph` | v1.0.0 | Code intelligence |
| 14 | `analytics` | HelixDevelopment | `tools/analytics` | v1.0.0 | Analytics Service |
| 15 | `helixtrack-core` | Helix-Track | `helixtrack.ru/core` | v3.2.0 | HelixTrack Bridge |
| 16 | `helixtrack-oauth` | Helix-Track | `helixtrack.ru/oauth` | v2.0.0 | HelixTrack Bridge |
| 17 | `helixtrack-deploy` | Helix-Track | `helixtrack.ru/deploy` | v1.5.0 | HelixTrack Bridge |

---

## Import Path Convention

Canonical per CD-11: **slash-path** (`digital.vasic/<module>`), not dot-path (`digital.vasic.<module>`).

```go
// Correct (canonical)
import "digital.vasic/security"
import "digital.vasic/containers"

// Incorrect (pre-convergence, deferred mass-rewrite)
import "digital.vasic.security"
```

> **DEFERRED:** Go module-path standardization (`digital.vasic.*` dot-paths, 600+ refs) is high-churn and deferred per CANONICAL_FACTS.md.

---

## HelixConstitution Integration

```bash
# In the HelixTerminator repository root:
git submodule add git@github.com:HelixDevelopment/HelixConstitution.git constitution
cd constitution
git checkout e6504c2   # pin to verified commit (helixcode-v1.1.0 line; CD-9)
cd ..
git add constitution .gitmodules
git commit -m "chore: add HelixConstitution submodule pinned to e6504c2"
```

Every clone:
```bash
git clone --recurse-submodules git@github.com:HelixDevelopment/helix_terminator.git
```

---

## Go Module Path Standardization

All 25 services use the canonical module path:

```go
module helixterminator.io/services/<name>

go 1.25

require (
    digital.vasic/containers v2.1.0
    digital.vasic/security v1.8.0
    digital.vasic/auth v1.5.0
    digital.vasic/recovery v1.3.0
    digital.vasic/observability v1.4.0
)
```

---

## CI/CD Submodule Compliance

Every PR gate verifies:

1. `submodules-catalogue.md` consistency
2. All submodules pinned to declared versions
3. No unlisted submodules in `go.mod`
4. Constitution inheritance integrity (`scripts/verify-all-constitution-rules.sh`)

---

## Cross-References

- [03 — Service Catalog](../03-service-catalog/) — Which services use which submodules
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — CI/CD pipeline definitions
- [14 — Constitution Compliance](../14-constitution-compliance/) — Governance, CI gates
- [16 — References](../16-references/) — CD-9 (constitution version), CD-11 (helix-deps)

---

*Section 13 — Submodule Integration*  
*Consolidated from: 10_submodule_integration.md, CANONICAL_FACTS.md (CD-9, CD-11)*
