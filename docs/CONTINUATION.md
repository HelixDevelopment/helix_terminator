# CONTINUATION — helix_terminator

Standing session-resumption record (Constitution §12.10 / §11.4.131). Keep current.

## One-line resume
MVP spec Wave-2 remediation is COMPLETE and gate-GREEN across all 12 docs + README; the deep-work
increment (~30 substantial authoring items) is scoped but NOT started. Next work = deep-work, or mvp4.

## Where we are
DONE and committed:
- Constitution submodule wired (`constitution/`, pinned e6504c2) + inheritance gate + §1.1 mutation.
- MVP spec analyzed (6 audits → ~253 findings) → `docs/research/mvp/REMEDIATION_REGISTER.md`.
- Canonical source of truth locked → `docs/research/mvp/output/CANONICAL_FACTS.md`.
- Wave 2 integrity + canonical reconciliation applied to ALL 12 docs + README. The docs consistency
  gate (`tests/docs_consistency_gate.sh`) passes GREEN (DUP/ANCHOR/LATEST/EMPTYTEST clean),
  independently verified; the §1.1 mutation still proves it bites. Gate is wired into
  `scripts/pre_build_verification.sh` (runs on every commit via the pre-commit hook).
- Multi-format exports (html/pdf/docx) regenerated in sync (§11.4.12) via
  `scripts/docs/regenerate_exports.sh --force`.
- CHANGELOG.md added.

NOT done (honest — no bluff):
- Deep-work (~30 items), tracked in REMEDIATION_REGISTER.md + CANONICAL_FACTS "Deferred":
  RLS-enforced tenant isolation; audit WORM anchoring; PostgreSQL DR/HA + RPO/RTO; ZK server-keygen
  removal (design); roadmap Phases 2–5 acceptance criteria/DoD/owners/estimates + risk register;
  single service registry (reconcile the divergent 25-service enumerations); dual-product scope
  reconciliation section; Go module-path standardization (`digital.vasic.*`, 600+ refs); missing
  diagrams (C1 context, DR topology, Vault/Key-Manager/Org/Billing wireframes); full real-time
  collaboration spec; client auto-update + mobile background-exec; device/native-a11y test coverage.
- These are genuine engineering authoring, several requiring careful review — the next increment(s).

## Canonical facts (authoritative copy in output/CANONICAL_FACTS.md)
Dual scope (SSH platform + VPN module) · org HelixDevelopment · domain helixterminator.io ·
ZK hard for vault, SSH-pw non-ZK · PG 17.2 / Go 1.25 / Kafka 3.9 / Redis 8 / K8s 1.31 / Flutter 3.24 /
Istio 1.22 · ports 443 edge / 8080 internal · regions us-east-1 primary / eu-west-1 DR · JWT EdDSA ·
RBAC {super_admin,org_admin,team_admin,member,auditor,api_user} · constitution pinned e6504c2.

## Next actions
1. Deep-work increment: pick items from REMEDIATION_REGISTER.md DEEP-WORK section; author with
   subagents (disjoint doc ownership), each verified against the gate + a render; regenerate exports
   (`scripts/docs/regenerate_exports.sh --force`); update CHANGELOG + this file; commit + push.
2. OR proceed to mvp4 once the operator sends instructions.

## Ledger
Controller ledger (git-ignored scratch): `.superpowers/sdd/progress.md`.
