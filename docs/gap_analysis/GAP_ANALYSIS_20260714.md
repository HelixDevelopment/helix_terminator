# HelixTerminator — Enterprise-Readiness Gap Analysis

**Revision:** 1
**Last modified:** 2026-07-14T00:00:00Z
**Author:** background agent (HelixDevelopment), kick-off pass
**Method:** direct read of the working tree at HEAD `3f65187` + `go build`/`go test`
execution — captured evidence, no guessing (Constitution §11.4.6). Every claim below
cites the command or file that established it. This supersedes, where it disagrees, the
aspirational `docs/DEVELOPMENT_KICKOFF.md` and the (now-stale) `docs/COVERAGE_LEDGER.md`.

---

## 0. Executive summary — honest shipped-vs-ROADMAP

HelixTerminator is a **25-service Go micro-services platform + Flutter client**, released
as `helix_terminator-0.1.0` (tag, 2026-07-09). The honest state:

- **SHIPPED (verified this pass):** all **25 services build** (`go build ./...` on every
  service — 22 immediately, 3 after `git submodule update --init` for the leaf Go libs
  `submodules/{auth,llmprovider,containers}`; not code defects). All 25 carry
  `main + model + repository + handler + server` layers with real HTTP APIs (~38k Go LOC).
  **Stress + chaos tests exist for 25/25 services** (`grep -rln 'go:build stress|chaos'`
  → 26 stress / 25 chaos files). Core crypto is genuinely implemented: ssh-proxy uses real
  `golang.org/x/crypto/ssh` (Dial/NewSession/agent/knownhosts); vault + keychain +
  pki(x509) have real encryption-at-rest and cert generation.
- **PARTIAL / ROADMAP (the real work to "enterprise-ready"):** the platform is
  **scaffold-complete and compiles, but is not yet enterprise-functional end-to-end.**
  Integration tests exist for only ~15/25 services; `docs/qa/` e2e transcripts cover
  3/25 features; several *core product* capabilities are thin or absent (see §2); the
  Flutter client's tests are mostly no-op stubs; and there is **no formal work tracker**
  (no `docs/Issues.md`, no §11.4.93 workable-items DB) — planning lives in a git-ignored
  `progress.md` + `docs/CONTINUATION.md`.
- **OPERATOR-BLOCKED (external credentials/decisions, honestly tracked):** ai cloud-LLM
  keys, billing Stripe keys, push FCM/APNs, and auth production JWT-key management (T15).

**Bottom line:** "0.1.0" is a *scaffold-complete + core-authZ-hardened* milestone, NOT an
enterprise-ready product. This document enumerates the delta and phases the completion.

---

## 1. Fleet-wide facts (captured this pass)

| Fact | Value | Evidence |
|---|---|---|
| Services | 25 | `ls services/` |
| Services that build | **25/25** | `go build ./...` per service (GOWORK=off) |
| Go LOC (services, non-test) | ~38,000 | `find … -name '*.go' -not -name '*_test.go' \| cat \| wc -l` |
| Stress test files | 26 (25/25 services) | `grep -rln 'go:build stress\|_stress_test'` |
| Chaos test files | 25 (25/25 services) | `grep -rln 'go:build chaos\|_chaos_test'` |
| Integration test files | ~15 | `find services -name '*integration*_test.go'` |
| `docs/qa/` feature dirs | 3 (ai, container-bridge, helixtrack-bridge) + this pass adds SSH-CA | `ls docs/qa/` |
| Formal work tracker | **NONE** (no Issues.md / workable-items DB) | `find . -name 'workable_items.db'` → none |
| Release tag | `helix_terminator-0.1.0` (2026-07-09) | `git tag` |

Note: the aspirational `DEVELOPMENT_KICKOFF.md` claims "all build, all test (426 tests)" and
`docs/COVERAGE_LEDGER.md` (rev 1, HEAD `c2718d2`) claimed large gaps (24/25 no stress/chaos,
keychain plaintext keys). BOTH are stale: the keychain plaintext defect is fixed
(`internal/crypto/crypto.go` + anti-bluff `repository_integration_test.go`), and stress/chaos
is now 25/25. This gap analysis is the current source of truth.

---

## 2. Core-product functional gaps (the value of a remote-terminal platform)

A remote-terminal / SSH-workspace product's value is its *terminal + SSH + PKI + recording*
path. Depth probe results:

| Capability | State | Evidence / gap |
|---|---|---|
| SSH brokering (ssh-proxy) | **REAL** | `x/crypto/ssh` Dial/NewSession/agent/knownhosts in `sshclient.go`, `wshandler.go` |
| **Short-lived SSH certificates (PKI)** | **WAS ABSENT → crypto core landed this pass (HT-SSHCA-001)** | pki-service had ONLY x509 (`grep crypto/ssh pki-service` → none), yet `SERVICE_REGISTRY.md §19` mandates SSH user+host certs. New `internal/sshca` package now mints+verifies them. HTTP+persistence wiring = **HT-SSHCA-002** (open). |
| ssh-proxy consumes SSH certs for auth | **ROADMAP** | ssh-proxy authenticates only via long-lived `ssh.ParsePrivateKey` — cert-based auth (the enterprise story) not wired. → **HT-SSHCA-003** |
| Terminal I/O proxy (websocket + PTY) | **THIN / UNVERIFIED** | `grep websocket\|pty terminal-service` (non-test) → no match; terminal-service appears to be metadata CRUD, not a live WS/PTY proxy. → **HT-TERM-001** (investigate + implement) |
| Session recording assembly | **UNVERIFIED depth** | present as service; asciinema/Kafka-segment assembly depth not yet probed. → **HT-REC-001** |
| Flutter client end-to-end | **MOSTLY STUB** | per prior ledger §6: 8/11 client test files are `expect(true, isTrue)`; sole e2e is boot-only with explicit TODO. → **HT-CLIENT-001** |

---

## 3. Cross-cutting quality gaps (risk-ordered per §11.4.132)

1. **[HIGH] No formal work tracker.** No `docs/Issues.md`, no §11.4.93 workable-items DB.
   Planning is ad-hoc/git-ignored → not resumable, not §11.4.15/§11.4.16-trackable.
   → **HT-INFRA-001** (this pass seeds `docs/Issues.md`; DB bootstrap tracked).
2. **[HIGH] Integration coverage ~15/25 services.** §11.4.25 invariant (2) unmet for ~10
   services (analytics, audit, collaboration, config, health, host, org, recording, sftp,
   snippet, ssh-proxy, terminal, workspace — subset). → **HT-TEST-INTEG-001**.
3. **[HIGH] `docs/qa/` e2e transcripts 3/25.** §11.4.83 requires an e2e transcript per
   shipped feature. → **HT-QA-001**.
4. **[HIGH] Core-product depth gaps** (terminal WS/PTY, ssh-proxy cert auth) — see §2.
5. **[MED] Flutter client tests are no-op stubs** (§2). → **HT-CLIENT-001**.
6. **[MED] Operator-blocked backends** (ai LLM keys, Stripe, FCM/APNs, JWT-key mgmt) —
   honestly blocked on external creds/decisions; track, don't fake. → **HT-BLOCKED-\***.
7. **[LOW] Shallow repository/server coverage** on ~13 services (constructor + not-connected
   negative path only). → folds into HT-TEST-INTEG-001.

---

## 4. Phased plan (build on the existing 0.1.0, do not restart)

Each phase is a set of tracked work items in `docs/Issues.md` (HT-NNN ids, Status/Type per
§11.4.15/§11.4.16). Phases are ordered most-critical + most-visible first (§11.4.42/§11.4.72);
security-critical core-product gaps lead.

- **Phase 1 — Core-product security & functionality (in progress).**
  HT-SSHCA-001 (SSH-CA crypto — **Implemented this pass**) → HT-SSHCA-002 (PKI SSH-cert
  HTTP + persistence + DB-gated integ) → HT-SSHCA-003 (ssh-proxy consumes SSH certs) →
  HT-TERM-001 (terminal WS/PTY proxy investigate+implement).
- **Phase 2 — Test-truth & QA.** HT-TEST-INTEG-001 (integration tests → 25/25) ·
  HT-QA-001 (docs/qa e2e transcripts → 25/25) · replace remaining stub tests with real ones.
- **Phase 3 — Client.** HT-CLIENT-001 (Flutter e2e real user journeys, §11.4.143).
- **Phase 4 — Governance/infra.** HT-INFRA-001 (§11.4.93 workable-items DB) ·
  HT-INFRA-002 (module-path standardization `digital.vasic.*`, deferred per CHANGELOG) ·
  PDF TOC-anchor fix (deferred).
- **Phase 5 — Operator-blocked backends** (as creds/decisions arrive): ai LLM, Stripe,
  FCM/APNs, production JWT-key management.
- **Phase 6 — Production hardening & deploy validation.** Real Terraform/K8s/Helm apply +
  smoke, SLO/load tests (currently labeled "planned"), security scans wired.

Anti-bluff (Constitution §11.4): every item ships four-layer coverage — a failing test
first (§11.4.115), real implementation, and captured physical evidence (§11.4.5/§11.4.69) —
before it may be marked Implemented/Completed/Fixed. Manual QA is the final gate (§11.4.185).

---

## 5. What was done in THIS kick-off pass

- Cloned + submodule-initialized the repo; verified **25/25 build** (captured logs).
- Established the honest state above (superseding two stale docs).
- **Implemented HT-SSHCA-001** — `services/pki-service/internal/sshca` (real SSH CA:
  generate CA, sign short-lived user + host certs, verify) + a comprehensive RED→GREEN test
  suite (11 tests, `-race`, 80.8% coverage, deterministic ×3) with golden-bad self-validation
  (§11.4.107(10)) and an independent OpenSSH `ssh-keygen -L` oracle. A real security defect in
  the verify oracle (accepting certs from an untrusted CA) was caught by the golden-bad test
  and fixed. Evidence: `docs/qa/HT-SSHCA-20260714/`.
- Seeded `docs/Issues.md` as the project's first formal tracker.

**This is a kick-off, not a completion.** ~10 tracked items across 6 phases remain (§4).
