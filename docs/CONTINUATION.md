# CONTINUATION — helix_terminator

Standing session-resumption record (Constitution §12.10 / §11.4.131). Keep current.

## One-line resume
MVP spec is hardened end-to-end: Integrity+Canonical AND the deep-work increment are DONE, gate
GREEN, exports in sync, all pushed to GitHub. Remaining: a few explicitly-deferred polish items
(below) and, when the operator says so, **mvp4**.

## Where we are (all committed + pushed to GitHub `main`, remote==local)
- `b45811f` Constitution submodule wired (`constitution/`, pinned e6504c2) + inheritance gate + §1.1 mutation.
- `7f3a0c4` Extracted MVP spec to `docs/research/mvp/output/`.
- `b7af519` Analysis (6 audits → ~253 findings → `docs/research/mvp/REMEDIATION_REGISTER.md`), locked
  `docs/research/mvp/output/CANONICAL_FACTS.md`, consistency gate + §1.1 mutation, export regenerator.
- `f8667a9` Wave 2 Integrity+Canonical across all 12 docs + README; gate wired into pre-build; exports synced.
- `a2ca4ff` Deep-work Wave A: 07 RLS/vault/audit/injection/WS; 05 ZK-keygen-removal/WORM/break-glass/threat-model;
  04 PG DR-HA/FinOps/RabbitMQ/canary; 01 C4/circuit-breakers/API-versioning.
- `abcd85f` Deep-work Wave B: 08 Phases 2-5/risk-register; 02 collab/auto-update/mobile-bg; 06 wireframes/Button/tokens;
  09 collab-SLO/soak-chaos-plans; 03 device-matrix/a11y/Pact/SBOM.
- (this commit) Cross-cutting: `SERVICE_REGISTRY.md` (single canonical 25-service list) + `SCOPE_AND_MODULES.md`
  (dual-scope reconciliation); CHANGELOG + this file synced.

## Verification (all GREEN)
`bash tests/docs_consistency_gate.sh` → PASS (DUP/ANCHOR/LATEST/EMPTYTEST clean); wired into
`scripts/pre_build_verification.sh` (runs on every commit via pre-commit hook). §1.1 mutation
`scripts/testing/meta_test_docs_gate.sh` proves the gate bites. Exports: `bash scripts/docs/regenerate_exports.sh --force`.

## Remaining / deferred (HONEST — not done)
- **Go module-path standardization** (`digital.vasic.*` dot-paths, 600+ refs) — deliberately DEFERRED
  (high-churn, error-prone); left in place and flagged. This is the main open register item.
- **PDF internal TOC-links do not resolve** (pandoc→weasyprint id-slug mismatch) — corpus-wide,
  pre-existing; source anchors are clean per the gate. A pandoc `--section-divs`/id-matching pass would fix it.
- Minor: not every ASCII diagram converted to mermaid; docs 01/10/11 reference "doc 01 canonical set"
  rather than the new `SERVICE_REGISTRY.md` by name (registry derives from doc 01, so not wrong).
- `regenerate_exports.sh` leaves `toPdfViaTempFile*` in CWD (gitignored; add a trap-cleanup later).

## Next actions
1. **"continue"** → resume from this file; pick remaining/deferred items above OR the register's tail.
2. **"mvp4"** → operator will send mvp4 instructions; this MVP is in a shippable-spec state.

## Canonical facts (authoritative copy in output/CANONICAL_FACTS.md; service set in output/SERVICE_REGISTRY.md; scope in output/SCOPE_AND_MODULES.md)
Dual scope (Module A Secure Terminal Platform + Module B Zero-Trust Connection Broker) · org HelixDevelopment ·
domain helixterminator.io · ZK hard for vault, SSH-pw non-ZK · PG 17.2 / Go 1.25 / Kafka 3.9 / Redis 8 /
K8s 1.31 / Flutter 3.24 / Istio 1.22 · ports 443 edge / 8080 internal · regions us-east-1 primary / eu-west-1 DR ·
JWT EdDSA · RBAC {super_admin,org_admin,team_admin,member,auditor,api_user} · constitution pinned e6504c2.

## Resume infrastructure (all in sync)
- This file (tracked) — read FIRST.
- `.superpowers/sdd/progress.md` — full controller ledger (git-ignored scratch) with every commit SHA.
- Persistent memory: `~/.claude-claude*/projects/-home-milos-Factory-.../memory/` (MEMORY.md + helix-terminator-resume.md).
- `.remember/` is git-ignored plugin scratch (no tracked remember.md).
