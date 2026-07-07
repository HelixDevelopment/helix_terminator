# CONTINUATION (MVP-spec subdir) — helix_terminator

**Revision:** 1
**Last modified:** 2026-07-07T00:00:00Z

> **This is NOT the canonical session-resumption file.** Per §11.4.131 there is exactly ONE
> standing resumption record for this project: **[`docs/CONTINUATION.md`](../../../../CONTINUATION.md)**.
> A fresh session MUST read that file first (then `git fetch --all`). This file was a stale
> duplicate describing the earlier MVP-spec-hardening phase; it is retained ONLY as a local
> record of the MVP-spec deferred-polish items below, which belong to this subdirectory.

## Scope of this file
Local tracking note for the `docs/research/mvp/**` spec corpus. Live project state, the active
work queue, and every commit SHA live in the canonical `docs/CONTINUATION.md` (Revision 6+) and
the controller ledger `.superpowers/sdd/progress.md`. The MVP spec reached a shippable-spec state
(Integrity+Canonical + deep-work increments DONE, docs-consistency gate GREEN, exports in sync,
all pushed — see the canonical file / `docs/research/mvp/REMEDIATION_REGISTER.md`).

## MVP-spec deferred polish (HONEST — not done; tracked here + in REMEDIATION_REGISTER.md)
- **Go module-path standardization** (`digital.vasic.*` dot-paths, 600+ refs) — deliberately
  DEFERRED (high-churn, error-prone); left in place and flagged. Main open register item.
- **PDF internal TOC-links do not resolve** (pandoc→weasyprint id-slug mismatch) — corpus-wide,
  pre-existing; source anchors are clean per the gate. A pandoc `--section-divs`/id-matching pass
  would fix it.
- Minor: not every ASCII diagram converted to mermaid; docs 01/10/11 reference "doc 01 canonical
  set" rather than `docs/research/mvp/output/SERVICE_REGISTRY.md` by name (registry derives from
  doc 01, so not wrong — a naming refresh only).
- `regenerate_exports.sh` leaves `toPdfViaTempFile*` in CWD (gitignored; add a trap-cleanup later).

## Canonical facts (authoritative copies)
`docs/research/mvp/output/CANONICAL_FACTS.md` (facts) · `.../output/SERVICE_REGISTRY.md` (25-service
set) · `.../output/SCOPE_AND_MODULES.md` (dual-scope). Dual scope (Module A Secure Terminal Platform
+ Module B Zero-Trust Connection Broker) · org HelixDevelopment · domain helixterminator.io ·
constitution pinned e6504c2.
