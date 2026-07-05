# Changelog

All notable changes to helix_terminator are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/); dates are ISO-8601.

## [Unreleased]

### Added
- Helix Constitution as a git submodule at `constitution/` (pinned `e6504c2`,
  `helixcode-v1.1.0` line) with an inheritance gate (`tests/verify_constitution_inheritance.sh`)
  and its paired §1.1 mutation proof.
- MVP spec remediation foundation: `docs/research/mvp/REMEDIATION_REGISTER.md`
  (~253 findings from 6 parallel audits) and `docs/research/mvp/output/CANONICAL_FACTS.md`
  (locked single source of truth).
- `tests/docs_consistency_gate.sh` (DUP/ANCHOR/LATEST/EMPTYTEST/DRIFT) + paired §1.1 mutation
  `scripts/testing/meta_test_docs_gate.sh`; now wired into `scripts/pre_build_verification.sh`.
- `scripts/docs/regenerate_exports.sh` — md→html/pdf/docx + mermaid→svg/png regenerator
  (`--force` for in-place §11.4.12 sync).
- `docs/CONTINUATION.md` (§12.10 resume record).
- Extracted MVP spec archive to `docs/research/mvp/output/` (verified byte-identical .tar/.zip).

### Fixed (MVP spec — Wave 2, all 12 docs + README)
- **02 client:** removed a ~1,453-line verbatim duplicate tail + repaired spliced/truncated code;
  sync aligned to CRDT vector-clock merge.
- **03 testing:** removed `:latest`; `TestAuthService_RateLimit_Login` replaced with a real
  asserting test (anti-bluff §1.1); test-type count reconciled to 12 (+5 sub-categories).
- **04 devops:** purged all `:latest` (pinned); de-duplicated Prometheus rule groups; fixed
  runAsUser / PodSecurity self-conflicts; `/healthz` gate corrected; rootless-Podman note.
- **06 ux:** split two interleaved documents (backend → Appendix P; no duplicate section numbers);
  recomputed WCAG contrast tables (2 real failures surfaced); OpenDesign §11.4.162 citation;
  resolved 2 keyboard-shortcut collisions.
- **01 architecture:** fixed §4.15–4.20 ordering (monotonic, no duplicate numbers); embedded §10
  now references canonical doc 10.
- **05 security / 07 api:** JWT canonicalized to EdDSA (Ed25519); RBAC converged to one 6-role
  vocabulary; zero-knowledge claims made honest (hard for vault, SSH-password explicitly non-ZK).
- **08 / 09 / 10 / 11 / 12 / README:** fixed broken TOC anchors; removed a bluff `--check` gate and
  orphaned code fences; reconciled the constitution version to the verified pin; honest SLO/load-test
  labeling; region/port/version/org/domain canonicalization across the corpus.
- Result: `tests/docs_consistency_gate.sh` passes GREEN (independently verified).

### Added (MVP spec — Deep-work Waves A/B + cross-cutting)
- **Security/data (07, 05):** Row-Level Security across all multi-tenant DBs; full-PII audit hash
  chain + crypto-shred GDPR erasure + WORM anchoring; true client-side zero-knowledge vault design
  (server keygen/re-wrap removed); item-level vault endpoints + ZK key rotation; SSO token
  encryption; blast-radius/authz gating; WebSocket resume; break-glass/JIT/SoD; STRIDE threat model.
- **Resilience/infra (04, 01):** PostgreSQL DR/HA (RPO/RTO, cross-region, Patroni, PITR); FinOps/cost
  section; RabbitMQ prod path; C4 context diagram; per-service circuit-breaker table; API versioning.
- **Product/client/UX (08, 02, 06):** roadmap Phases 2–5 (entry/exit/acceptance/DoD) + risk register +
  owners/estimates + Gantt; collaboration client spec + auto-update + mobile background-exec + conflict
  UI + error taxonomy; Vault/Org/Billing/Collab wireframes + Button spec + light-theme tokens + diagrams.
- **Perf/testing (09, 03):** collaboration SLO model + soak/chaos/stress PLANS (labeled planned);
  device/topology matrix + terminal-render perf methodology + native a11y + Pact contracts + SBOM gate.
- **Cross-cutting:** `SERVICE_REGISTRY.md` (single canonical 25-service list) and `SCOPE_AND_MODULES.md`
  (dual-scope Module A + Module B reconciliation).

### Deferred (HONEST — still NOT done; tracked in REMEDIATION_REGISTER.md)
- Go module-path standardization (`digital.vasic.*`, 600+ refs) — high-churn, deliberately deferred.
- PDF internal TOC-links do not resolve (pandoc→weasyprint id-slug mismatch) — corpus-wide,
  pre-existing; source anchors are clean per the gate.
- Minor: not every ASCII diagram converted to mermaid; `regenerate_exports.sh` leaves temp files in CWD.
