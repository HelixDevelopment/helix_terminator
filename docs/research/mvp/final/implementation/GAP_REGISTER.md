# Gap Register

> **Section:** 15-gap-analysis-remediation  
> **Status:** Complete  
> **Last Updated:** 2026-07-05

This register catalogs the critical gaps identified during the MVP documentation consolidation for `helix_terminator`. Each entry includes severity, impact, owner, and remediation plan.

---

## Gap Register Table

| ID | Gap | Severity | Impact | Owner | Remediation | Status |
|---|---|---|---|---|---|---|
| GAP-01 | **gRPC Service Stubs Unverified** — 25 `.proto` files exist but have not been compiled or tested against actual service implementations. | Medium | Risk of interface drift between proto definitions and Go service code. | Backend Team | Run `protoc` validation in CI; add buf linting. | Open |
| GAP-02 | **OpenAPI YAML Partial Coverage** — `openapi.yaml` covers ~131 explicit REST endpoints; ~90 endpoints (gRPC, WebSocket, internal) are not represented. | Medium | API consumers may miss non-REST interfaces. | API Team | Generate OpenAPI extensions for WebSocket streams; document gRPC gateway mappings. | Open |
| GAP-03 | **SQL Schema Unvalidated** — 19 `.sql` files define 121 `CREATE TABLE` statements but have not been executed against PostgreSQL 17.2. | High | Schema syntax errors or type mismatches may block migration. | DBA Team | Execute all SQL files in a fresh PostgreSQL 17.2 container; fix errors. | Open |
| GAP-04 | **helix-deps.yaml Version Pins Stale** — Version pins (Go 1.25, Flutter 3.24, K8s 1.31, etc.) are aspirational and may not match current production. | Medium | Build reproducibility risk if versions drift. | DevOps Team | Automate version extraction from `go.mod`, `pubspec.yaml`, and cluster specs. | Open |
| GAP-05 | **Mermaid/Draw.io Diagrams Not Rendered** — 30 Mermaid + 8 Draw.io diagrams are defined in READMEs but not pre-rendered to PNG/SVG. | Low | Documentation readers without Mermaid/Draw.io plugins cannot view diagrams. | Docs Team | Add CI step to render Mermaid to SVG; export Draw.io to PNG. | Open |
| GAP-06 | **Section 06 (Client Spec) Incomplete** — Flutter client specification lacks detailed widget tree, state management patterns, and platform-specific handling. | High | Client developers lack canonical reference. | Mobile Team | Expand section with widget catalog, Riverpod/BLoC patterns, and platform matrix. | Open |
| GAP-07 | **Section 08 (DevOps) Incomplete** — Missing Terraform module inventory, Helm chart parameter references, and CI/CD pipeline diagrams. | High | Infrastructure onboarding is blocked. | Platform Team | Import canonical Terraform/Helm docs; add pipeline Mermaid diagrams. | Open |
| GAP-08 | **Section 09 (Security) Incomplete** — Zero-trust architecture described at high level; missing mTLS certificate rotation procedures, WAF rules, and penetration test results. | Critical | Security compliance (SOC 2, ISO 27001) cannot be verified. | Security Team | Add certificate rotation SOPs, WAF rule inventory, and pentest summary. | Open |
| GAP-09 | **Section 10 (UX) Incomplete** — Design system references `opendesign/` but lacks component usage examples, accessibility audit, and dark mode spec. | Medium | UI consistency and a11y compliance at risk. | Design Team | Add component Storybook links, a11y checklist, and theme switching spec. | Open |
| GAP-10 | **Section 11 (Performance) Incomplete** — No load test results, latency benchmarks, or capacity planning models. | High | Cannot validate SLOs (99th percentile < 200ms). | SRE Team | Import k6/Locust results; add capacity model and SLO dashboard links. | Open |
| GAP-11 | **Section 12 (Roadmap) Incomplete** — Milestones are placeholder dates without resource allocation or dependency mapping. | Medium | Planning accuracy is low. | Product Team | Link to Jira/Linear roadmap; add Gantt chart and resource matrix. | Open |
| GAP-12 | **Section 13 (Submodule Integration) Incomplete** — `constitution/` and `opendesign/` submodule version pins and update procedures are not documented. | Medium | Risk of submodule drift and breaking changes. | Platform Team | Document submodule update SOPs and version lock files. | Open |
| GAP-13 | **Cross-Reference Integrity Not Automated** — README cross-references are manual and may break as files move. | Low | Documentation rot over time. | Docs Team | Add markdown-link-check to CI; validate relative paths. | Open |
| GAP-14 | **Canonical Source Docs Not Synced** — Copied `README.md`, `CANONICAL_FACTS.md`, `SCOPE_AND_MODULES.md`, and `SERVICE_REGISTRY.md` in `01-executive-summary/` are static copies; originals in `docs/research/mvp/output/` may diverge. | Medium | Executive summary may become stale. | Docs Team | Replace copies with symlinks or add CI check for divergence. | Open |
| GAP-15 | **Constitution Inheritance Verification** — `tests/verify_constitution_inheritance.sh` gates build but is not documented in the consolidated docs. | Low | New contributors may miss the inheritance requirement. | Governance Team | Add constitution verification step to `16-references/` and `01-executive-summary/`. | Open |

---

## Remediation Priority Matrix

| Priority | Gaps | Rationale |
|---|---|---|
| P0 (Critical) | GAP-08, GAP-03 | Security compliance and database integrity are blockers. |
| P1 (High) | GAP-06, GAP-07, GAP-10, GAP-11 | Client, infrastructure, performance, and roadmap gaps block delivery. |
| P2 (Medium) | GAP-01, GAP-02, GAP-04, GAP-09, GAP-12, GAP-14 | Interface drift, version staleness, design system, and submodule risks. |
| P3 (Low) | GAP-05, GAP-13, GAP-15 | Diagram rendering, link checking, and governance documentation. |

---

## Related Documents

- `15-gap-analysis-remediation/README.md` — Full gap analysis methodology and remediation workflow.
- `01-executive-summary/README.md` — Executive summary with project scope and critical success factors.
- `16-references/README.md` — Canonical references and external links.
- `tests/verify_constitution_inheritance.sh` — Constitution inheritance gate script.
