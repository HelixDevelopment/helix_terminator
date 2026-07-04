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

### Deferred (tracked in REMEDIATION_REGISTER.md / CANONICAL_FACTS.md — NOT yet done)
- Deep-work (~30 items): RLS-enforced tenant isolation, audit WORM anchoring, PostgreSQL DR/HA +
  RPO/RTO, ZK server-keygen removal, roadmap Phases 2–5 acceptance criteria, single service
  registry, dual-product scope section, Go module-path standardization, missing diagrams,
  full real-time-collaboration spec.
