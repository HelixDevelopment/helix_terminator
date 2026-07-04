# CONTINUATION — helix_terminator

Standing session-resumption record (Constitution §12.10 / §11.4.131). Keep current.

## One-line resume
MVP spec hardening is mid-flight: analysis + decisions + tooling are DONE and committed; the
per-doc integrity/canonical edits were interrupted by a session rate-limit and must be re-run by
fresh subagents after the limit resets. Nothing is broken — half-edited docs were reverted to clean.

## Where we are (as of commit after 7f3a0c4)
DONE and committed:
- Constitution submodule wired at `constitution/` (pinned e6504c2, helixcode-v1.1.0 line) + inheritance gate + §1.1 mutation (Effort 1).
- MVP spec analyzed: 6 parallel audits → ~253 deduped findings (~55 Critical). Consolidated in
  `docs/research/mvp/REMEDIATION_REGISTER.md`.
- Canonical source of truth locked (operator decisions CD-1..CD-12) in
  `docs/research/mvp/output/CANONICAL_FACTS.md`.
- Tooling: `tests/docs_consistency_gate.sh` (DUP/ANCHOR/LATEST/EMPTYTEST/DRIFT) + its §1.1 mutation
  `scripts/testing/meta_test_docs_gate.sh`; export regenerator `scripts/docs/regenerate_exports.sh`
  (pandoc/weasyprint/mmdc — all verified).

NOT done (honest — no bluff):
- The consistency gate is currently RED on the corpus by design (RED baseline, §11.4.115): DUP in 02,
  13 broken ANCHORs in 04/08/10/11, 12 `:latest` in 03/04/09, 2 EMPTYTEST in 10/11. These get fixed
  in Wave 2 and the gate goes GREEN, then it is wired into pre-build.
- Wave 2 (per-doc integrity + canonical edits) — INTERRUPTED by rate-limit, docs reverted to clean.
- Deep-work (~30 items): RLS-everywhere, audit WORM, DR/RPO-RTO, ZK server-keygen removal, roadmap
  Phases 2–5 acceptance criteria, single service registry, dual-scope reconciliation section,
  import-path standardization, missing diagrams, real-time-collaboration spec. Scoped, not started.

## Next actions (resume after rate-limit reset — 11:10pm Asia/Aqtau)
1. Re-dispatch Wave 2 per-doc implementers (disjoint file ownership), applying CANONICAL_FACTS +
   the per-doc packages in REMEDIATION_REGISTER.md, each self-verifying against
   `tests/docs_consistency_gate.sh` and a weasyprint render:
   - Batch 1: 02 (remove ~1,440-line dup tail near L5352 + repair spliced code), 06 (split interleaved
     UX/backend docs, recompute wrong WCAG tables, OpenDesign §11.4.162 cite, fix ⌘K/⌘⇧Z collisions),
     11 (fix anchors, replace empty stub test, remove `--check` bluff gate, canonical constitution
     cite, Module-B scope note), 04 (`:latest`→pinned, dedupe Prometheus groups, region/version/port
     canonicalization, rootless-Podman note).
   - Batch 2: 01, 10, 03, 08.  Batch 3: 05, 07, 09, 12, README.
2. Run `bash tests/docs_consistency_gate.sh` → must be GREEN. Then wire it into
   `scripts/pre_build_verification.sh` / pre-commit.
3. Regenerate ALL exports: `bash scripts/docs/regenerate_exports.sh`.
4. Add `CHANGELOG.md`; refresh this file; commit + push GitHub (only parent upstream); verify remote HEAD.
5. Then tackle the deep-work increment(s).

## Canonical facts (summary; authoritative copy in output/CANONICAL_FACTS.md)
Dual scope (SSH platform + VPN module) · org HelixDevelopment · domain helixterminator.io ·
ZK hard for vault, SSH-pw non-ZK · PG 17.2 / Go 1.25 / Kafka 3.9 / Redis 8 / K8s 1.31 / Flutter 3.24 /
Istio 1.22 · ports 443 edge / 8080 internal · regions us-east-1 primary / eu-west-1 DR · JWT EdDSA ·
RBAC {super_admin,org_admin,team_admin,member,auditor,api_user} · constitution pinned e6504c2.

## Ledger
Controller ledger (git-ignored scratch): `.superpowers/sdd/progress.md`.
