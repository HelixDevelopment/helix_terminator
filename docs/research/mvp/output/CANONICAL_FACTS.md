# helix_terminator MVP — Canonical Facts (Single Source of Truth)

**Locked:** 2026-07-04 · **Authority:** operator decisions (CD-1..CD-12) + verified repository facts.
Every spec document under `docs/research/mvp/output/docs/markdown/` MUST conform to the values
below. Where a document currently disagrees, the document is wrong and is being reconciled.
This record exists so no contributor (human or agent) has to guess a canonical value — guessing
canonical facts is a Constitution §11.4.6 / §11.4 anti-bluff violation.

## Product scope (CD-1) — DUAL
`helix_terminator` is a **dual-scope** product family, not one product:
- **Module A — Secure Terminal Platform** (primary): SSH / SFTP / vault / terminal / real-time
  collaboration. Owns docs 01–10, 12.
- **Module B — Zero-Trust Connection Broker**: WireGuard / VPN / connection-broker. The compliance
  domain historically described in doc 11.
Neither description is an "error"; they are two modules. A **Scope & Module Boundary** section
MUST reconcile them and every doc must state which module(s) it addresses. (DEEP-WORK: author the
reconciling scope section — next increment.)

## Identity (CD-2)
- **Org / GitHub namespace:** `HelixDevelopment` (verified: real remote `github.com/HelixDevelopment/helix_terminator`).
- **Primary domain:** `helixterminator.io`.
- Auth issuer: `auth.helixterminator.io`.
- Do NOT use `vasic-digital`, `digital-vasic`, `helixterm.io` for org/domain identity going forward.

## Zero-knowledge posture (CD-10)
- **HARD** for vault items: client-side end-to-end only. Remove server-side key generation / re-wrap
  for vault secrets from the spec's design (DEEP-WORK: security redesign — next increment).
- SSH **password-auth** hosts are explicitly **non-ZK**; the spec must stop describing them as ZK.

## Version pins (CD-4 — latest-stable)
| Component | Canonical |
|---|---|
| PostgreSQL | 17.2 |
| Go | 1.25 |
| Apache Kafka | 3.9 |
| Redis | 8 |
| Kubernetes | 1.31 |
| Flutter | 3.24 |
| Istio | 1.22 |

Replace all drifted version strings (16.x/16.2/16.3/17.0 → 17.2; 1.23 → 1.25; 3.7 → 3.9; Redis 7 → 8;
1.30 → 1.31; Flutter 3.22 → 3.24). Do NOT change versions inside negative/attack-test example context
where an old version is the point of the example — flag those instead.

## Networking (CD-5, CD-6)
- **API gateway ports:** `443` (edge, TLS) and `8080` (internal). Drop `8000`.
- **Regions:** `us-east-1` primary, `eu-west-1` DR. DR runbook owned by doc 04.

## Auth & access (CD-7, CD-8)
- **JWT signing:** `EdDSA` (Ed25519). Replace `RS256` as the *signing choice* (leave RS256/ES256/HS256
  where they appear as attack-test or negative examples).
- **RBAC roles (single vocabulary):** `super_admin`, `org_admin`, `team_admin`, `member`, `auditor`,
  `api_user`. Replace the other two role vocabularies; normalize the RBAC schema to these.

## Governance (CD-9) — VERIFIED, overrides doc claims
- **HelixConstitution** is the submodule at `constitution/`, pinned at commit `e6504c2`
  (`git describe` = `helixcode-v1.1.0-39-ge6504c2`). Cite it as **"HelixConstitution (pinned e6504c2,
  helixcode-v1.1.0 line)"**. Do NOT assert "v2.0" / "v1.0.0" / "v4.1.0" — those are unverified doc claims.

## Microservices (CD-3)
- Adopt doc **01**'s SSH-domain service set as canonical. Publish ONE service registry; other docs
  reference it rather than re-enumerating divergently. (DEEP-WORK: build the single registry — next increment.)

## Deferred (explicitly NOT done this increment — no bluff)
- Go module-path standardization (`digital.vasic.*` dot-paths, 600+ refs) — high-churn, DEFERRED. Do
  NOT mass-rewrite import paths yet; leave and flag.
- RLS-everywhere, audit WORM anchoring, PostgreSQL DR/HA + RPO/RTO authoring, item-level vault +
  key-rotation endpoints, full real-time-collaboration spec, roadmap Phases 2–5 acceptance criteria,
  client auto-update / mobile background exec, device/native-a11y test coverage, missing diagrams,
  ZK server-keygen removal, dual-product scope section, single service registry.
- Test-type count (CD-12): default **12** types; reconcile the 17-type list against this in a later pass.
