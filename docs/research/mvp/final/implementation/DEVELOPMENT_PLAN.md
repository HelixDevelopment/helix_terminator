# HelixTerminator — Full Development Plan

**Revision:** 1
**Last modified:** 2026-07-07T00:00:00Z
**Authority:** Operator kick-off (2026-07-07, "kick-off full development"). Executed under the Helix Constitution — anti-bluff covenant (§11.4), autonomous-loop default (§11.4.126), Subagent-Driven Development.
**Status:** ACTIVE — Phase 0 (ground truth) near-complete; backend-verification stream in flight; execution streams launching.

> **This plan is grounded in *verified* repository state, not the kickoff doc's claims.** Every row's status is either backed by captured evidence (path cited) or explicitly marked `PENDING` / `SKIP-with-reason`. No "done" without physical proof (§11.4.5 / §11.4.69 / §11.4.123). No false results, no bluff, anywhere.

---

## 0. Ground-truth snapshot (captured 2026-07-07 — evidence in `scratchpad/groundtruth/`)

| Area | Kickoff-doc claim | Verified reality | Evidence |
|---|---|---|---|
| Backend build/vet | "25 services build" | **TRUE — 25/25 build + vet clean** (Go 1.26.4, not doc's 1.22/1.25) | `groundtruth/go.md` |
| Backend tests | "426 tests pass" | **FALSE — 21/25 pass, 4 FAIL** (audit/config/host/notification: identical CORS bug); 408 func / 428 leaf (380P/4F/44S); `test/banks` 150/175 modules fail to compile; integration/contracts orphaned | `groundtruth/go.md` |
| SQL migrations | "19 files, ~121 tables" | **APPLY GREEN** — 25 files (not 19) apply cleanly to real PG 17.2; **38 tables** (not ~121, 3.2× overstated); gateway+health = TODO stubs | `groundtruth/sql.md` |
| Flutter file counts | 32 screens / 278 widgets / 19 BLoCs / 17 svc / 14 models | **TRUE literally** (32/278/19/17/14 exact) | `groundtruth/flutter.md` |
| Flutter depth | "client implemented" | **FALSE** — `terminal_view.dart` (core) is a stub; ~110 widgets are 1-line demos; ~30 placeholder stubs; **65 TODO/FIXME in prod source** (§11.4.27 violation); 7/19 BLoCs tested; 4 models lack JSON | `groundtruth/flutter.md` |
| Flutter toolchain | — | flutter/dart **MISSING on host** → containerized (§11.4.161) or SKIP (§11.4.3) | host `command -v` |
| Design system | (new, uncommitted) | Real: 1,351 lines tokens/theme/components; **not yet wired into `lib/`** | `groundtruth/flutter.md` |
| Governance gates | — | **GREEN + mutation-proof**: inheritance 10/10 (exit0), docs gate (exit0), meta-test (exit0), false-positive-proof (exit0) | `groundtruth/infra.md`, `gate_*.log` |
| Docs drift | — | Real version-string drift (postgres/go/kafka/redis) — WARN-only, unfixed | `groundtruth/infra.md` |
| CI/CD | "15-phase pipeline" | **LIVE (5 workflows)** — violates §11.4.156 (must be disabled); `main.yml` auto-deploys prod; `release.yml` publishes on `v*` tag. **Safety-relevant.** | `groundtruth/infra.md` |
| Infra tools | — | terraform/helm/kubectl/kustomize **MISSING on host**; modules/charts exist on disk → containerized validation | `groundtruth/infra.md` |
| Structure | 25 services | Confirmed: 25 services (own `go.mod`); `test/banks` = 177 test-bank modules (full taxonomy); 24 proto; 19 sql | host survey |
| Submodules | — | `constitution` + `open-design` only. **Missing per constitution:** `containers` (§11.4.76), Challenges+HelixQA (§11.4.27) — local `test/` dirs instead. **Operator-scope.** | `.gitmodules` |

**Uncommitted working tree at kickoff:** 925 paths (789 new + 131 modified) — prior-session WIP, mutation-residue scan CLEAN (§11.4.84). **Not blindly committable** — the workflow changes add more §11.4.156 violations and backend must be proven green first (§11.4.121).

---

## Global constraints (binding on EVERY task — copied into every subagent dispatch)

1. **Anti-bluff.** Every closure carries captured physical evidence (real command output / real test run / real DB delta / rendered pixels / sink-side). Metadata-only / config-only / absence-of-error / grep-without-runtime PASS is FORBIDDEN (§11.4 / §11.4.1 / §11.4.123).
2. **RED-first (§11.4.43 / §11.4.115).** Each fix ships a test that reproduces the defect on the pre-fix state, then flips GREEN. "Test added after the fix" is a bluff.
3. **4-layer coverage (§11.4.4(b)).** pre-build gate + on-target test + paired §1.1 mutation + (user-visible) HelixQA challenge.
4. **No fakes beyond unit tests (§11.4.27).** Mocks/stubs/TODOs live ONLY in unit tests. The 65 prod-source TODOs are themselves defects to burn down.
5. **Missing host toolchain → rootless podman (§11.4.161 / §11.4.173)** OR honest SKIP-with-reason (§11.4.3). Never a fake pass. podman + docker are available.
6. **Git discipline.** Commit only via the official wrapper (`scripts/commit_all.sh`); **no `git add -A` inside submodules**; **no force-push, merge-onto-latest-main (§11.4.113)**; background push (§11.4.88); commit only when the tree is quiescent + green (§11.4.121 / §11.4.84).
7. **Code review (§11.4.142 / §11.4.125 / §11.4.134).** Every batch passes an independent reviewer; iterate to a clean GO before build/commit.
8. **UI proof (§11.4.170).** Every UI surface proven by device-independent host-rendered pixels per screen×state×{light,dark}, dual-validated (golden diff + OCR/vision oracle).
9. **Host safety (§12.6 60% RAM, §12.11).** Bound parallelism; one heavy build at a time; single-owner per exclusive resource (§11.4.119).

---

## Phase 0 — Ground truth & green baseline  *(IN FLIGHT)*

| Task | Subtasks | Evidence | Exit |
|---|---|---|---|
| **T0.1** Verify backend | ✅ done — 25/25 build+vet; 4 CORS fails; test/banks 150/175 broken; SQL 25/25 apply on real PG17.2 | `groundtruth/go.md`, `sql.md` | ✅ |
| **T0.2** Verify Flutter | ✅ done (SKIP host; file+depth audit) | `groundtruth/flutter.md` | ✅ |
| **T0.3** Verify infra+gates | ✅ done (tools SKIP; gates GREEN; CI-live finding) | `groundtruth/infra.md` | ✅ |
| **T0.4** Reconcile CI to §11.4.156 | Disable all 5 workflows (gut to no-op with an explanatory header), preserving files; re-enable is trivial. **Precondition to committing workflow changes.** | disabled workflows + gate | §11.4.156 compliant |
| **T0.5** Safe commit of verified batch | After T0.1 green + T0.4: categorize 925 paths; commit in reviewed, logically-grouped commits via wrapper; background push; verify remote==local | commit SHAs, push log | Baseline committed |

**Exit criteria:** verified-green backend baseline committed + pushed; real numbers recorded; CI compliant.

---

## Phase 1 — Backend correctness & data layer  *(P0)*

Per-service template (applied to each of the 25: gateway, auth, user, vault, host, ssh-proxy, terminal, keychain, workspace, config, pki, billing, org, notification, audit, health, ai, collaboration, container-bridge, helixtrack-bridge, port-forward, sftp, recording, snippet, analytics):

- **T1.a** Fix any build/vet/test break found in T0.1 (RED-first).
- **T1.b** `go test -race ./...` green with real coverage number recorded.
- **T1.c** Validate the service's `migrations/*.sql` against **real PostgreSQL 17.2** in rootless podman (GAP-03, P0) — apply, capture schema, assert no error.
- **T1.d** Contract tests (`test/contracts`) green against the real handler surface.
- **T1.e** Structured-logging + health/readiness endpoints exercised (real HTTP probe).

Cross-service:
- **T1.f** Proto: `buf lint` + `protoc` compile all 24 protos in a container (GAP-01); assert no interface drift vs Go structs.
- **T1.g** OpenAPI coverage reconcile (GAP-02) — document gRPC/WS gaps honestly.

**Evidence:** per-service `go test -race` logs, `psql` migration transcripts, buf-lint output. **Exit:** all 25 build+test+race green; all 19 SQL files apply to PG17.2; protos compile.
**Streams:** parallelizable across disjoint services (§11.4.58 / §11.4.119 — one DB owner per test).

---

## Phase 2 — Security hardening  *(P0 — GAP-08)*

- **T2.a** `govulncheck` all modules (container); triage + fix CRITICAL/HIGH.
- **T2.b** Trivy scan service images; TruffleHog secret sweep (§11.4.10 credential audit).
- **T2.c** mTLS certificate-rotation SOP + PKI-service exercise; WAF rule inventory.
- **T2.d** auth-service crypto review (`internal/crypto` was modified) — RED test for any weakness.
- **T2.e** `test/security` bank green against real services.
- **T2.f** Threat model + zero-trust doc completion (GAP-08).

**Evidence:** govulncheck/trivy/trufflehog reports, security-bank run, cert-rotation transcript. **Exit:** no open CRITICAL/HIGH; security bank green; GAP-08 closed with evidence.

---

## Phase 3 — Flutter client  *(P0 — namesake feature is a stub)*

- **T3.a** Provision Flutter 3.24 via rootless podman image (§11.4.161); `pub get` + `flutter analyze` (target: 0 issues) + `flutter test`.
- **T3.b** **Implement the real terminal emulator** (`terminal_view.dart`) — the app's core; wire a real terminal/SSH package (pubspec currently lacks it).
- **T3.c** Burn down the **65 prod-source TODO/FIXME** (§11.4.27): replace the ~30 placeholder stubs + ~110 demo wrappers with real implementations or remove; add JSON to the 4 models.
- **T3.d** Wire `design_system/` (OpenDesign §11.4.162) tokens/theme into `lib/`; light+dark variants; no label overlap.
- **T3.e** BLoC test coverage to 19/19 (12 currently missing).
- **T3.f** **Host-rendered visual proof (§11.4.170)** for every screen×state×{light,dark}: golden image-diff + OCR/vision oracle.
- **T3.g** `integration_test` E2E of core flows.

**Evidence:** analyze log (0), test run, rendered PNGs + oracle verdicts, container build log. **Exit:** analyze clean, tests green, terminal works (proven), TODO count 0 in prod source, design system wired, visual proofs captured.
**Honest constraint:** all Flutter work is containerized (host has no SDK); if the container path is infeasible for a step, honest SKIP + operator-blocked item.

---

## Phase 4 — Infrastructure  *(P1)*

- **T4.a** Terraform validate+plan all 6 modules + 3 envs via container (GAP-07).
- **T4.b** `helm lint` + `helm template` helixterm chart; reconcile `values.yaml` (uncommitted MM).
- **T4.c** `kustomize build` base + dev/staging/production overlays.
- **T4.d** Rootless podman compose boot of PG/Redis/Kafka (§11.4.76 note: no containers submodule — flag; use local compose) with healthchecks.
- **T4.e** Reconcile modified TF modules (eks/elasticache/iam/msk/rds) — validate each change.

**Evidence:** validate/lint/kustomize exit0 logs, compose healthcheck output. **Exit:** all infra validates; local stack boots healthy.

---

## Phase 5 — Integration & E2E  *(P1 — §11.4.98 full-automation, §11.4.27 real system)*

- **T5.a** Boot real PG/Redis/Kafka (podman); run `test/integration` against **real services** (no mocks).
- **T5.b** `test/e2e` full user journeys; `test/contracts` cross-service (Pact-style).
- **T5.c** Test banks (`test/banks/*`): **REWRITE to exercise the real system, OR remove** (§11.4.27 / §11.4.124) — NEVER silence the unused-import errors to green (that would manufacture ~100 empty passing tests = the exact §11.4 bluff). Root cause of the 150 compile-fails = a bug in `scripts/generate_test_banks.go` (6/7 templates import packages their `t.Skip` bodies never use); fix the generator + regenerate REAL bodies. Per `banks_analysis.md`: unit/a11y/compat → REMOVE (dupes/unrunnable in Go); contract/integration/security → rewrite to call real internal packages (absorb then delete the orphaned `test/contracts`+`test/integration`); chaos/e2e → gated on Phase 6/Phase 4 infra.
- **T5.d** `test/devicematrix` + `test/challenges` where applicable.

**Evidence:** real service logs, DB deltas, sink-side reports, per-bank result.json. **Exit:** integration+e2e+contract green against real infra; banks **rewritten-to-real or removed** (NEVER silenced-to-green — §11.4.27).

---

## Phase 6 — Performance, stress & chaos  *(P1 — GAP-10/11, §11.4.85)*

- **T6.a** k6 load (`test/performance/k6`) via container; capture p50/p95/p99 vs SLO (<200ms p99).
- **T6.b** Soak + stress (N≥100 / concurrency≥10); latency.json.
- **T6.c** Chaos injection (`test/chaos`): process-kill, network-fault, resource-exhaustion; recovery traces.
- **T6.d** Capacity model + SLO dashboard refs.

**Evidence:** latency.json, categorized_errors.txt, recovery_trace.log. **Exit:** SLOs measured (met or gap tracked); chaos recovery proven.

---

## Phase 7 — Observability  *(P2)*

- **T7.a** Prometheus RED metrics wired + scraped (real).
- **T7.b** Jaeger tracing across services via `X-Request-ID`.
- **T7.c** Loki structured logs + redaction.
- **T7.d** Alert rules (error-rate, latency, service-down).

**Evidence:** live metric scrape, trace span capture, alert-fire test. **Exit:** observability stack proven end-to-end locally.

---

## Phase 8 — Docs, gap-register closure & coverage ledger  *(P2/P3)*

- **T8.a** Close GAP-01…15 each with captured evidence; update `GAP_REGISTER.md` status truthfully.
- **T8.b** Fix docs DRIFT (postgres/go/kafka/redis version strings) → gate DRIFT clean.
- **T8.c** Regenerate all exports (md+html+pdf+docx) in sync (pandoc/weasyprint OK, §11.4.65); fingerprint (§11.4.86).
- **T8.d** Coverage ledger (§11.4.25): feature × platform × invariant-1..6.
- **T8.e** Per-feature Status + Status_Summary (§11.4.153 / §11.4.45 / §11.4.56); README doc-links (§11.4.57).
- **T8.f** Render 30 mermaid + 8 drawio diagrams (GAP-05) where a renderer exists (drawio honest SKIP if no CLI).

**Evidence:** gate runs, export mtimes, ledger doc. **Exit:** gaps closed or honestly deferred; exports synced; ledger published.

---

## Phase 9 — Release preparation  *(gated by all above)*

- **T9.a** Full-suite retest from clean baseline (§11.4.40) — all phases green together.
- **T9.b** Production-readiness planning + realistic timeline (§11.4.172).
- **T9.c** Verify CI compliant/disabled (§11.4.156) so tagging is safe.
- **T9.d** Prefixed release tag `<prefix>-<version>` (§11.4.151); push to all upstreams (§2.1) merge-onto-latest-main, **no force-push** (§11.4.113).
- **T9.e** Refresh CONTINUATION + CHANGELOG + memory (§11.4.131 / §12.10).

**Exit (terminal condition A, §11.4.126):** validated+verified tag published across repo + submodules to all remotes, remote==local.

---

## Execution model

- **Method:** Subagent-Driven Development — one implementer subagent per task → task review (spec + quality) → fix loop → ledger line; broad whole-branch review before release.
- **Parallelism:** 3–4 concurrent streams on disjoint scope (§11.4.103), backfilled the moment a stream finishes; bounded by §12.6 (60% RAM) and single-owner-per-resource (§11.4.119, esp. the shared PG/podman).
- **Durable progress:** `.superpowers/sdd/progress.md` ledger (EFFORT 4) — every task's commit SHAs + evidence path; survives compaction.
- **Priority order (§11.4.132 risk-descending):** most-recently-worked + most-fragile first → P0 (backend build/DB, security, terminal-stub) before P1/P2/P3.

## Honest constraints & operator-scope items (not silently resolved)

1. **Toolchain:** flutter/terraform/helm/kubectl/kustomize/buf/protoc/k6 absent on host → containerized (rootless podman) or honest SKIP. Container images must be pulled (network); if blocked → operator-blocked item, not a fake pass.
2. **CI/CD live (§11.4.156):** will be disabled (T0.4) — reversible, mandated, and removes accidental-prod-deploy risk. Surfaced for operator awareness.
3. **Missing submodules:** `containers` (§11.4.76), Challenges + HelixQA (§11.4.27) are mandated as submodules but exist as local dirs. Adding submodules is an operator decision — flagged, not auto-added.
4. **Kickoff-doc overstatement:** "production-ready / implemented" is not supported by evidence (Flutter core stub, 65 TODOs). This plan treats the platform as **scaffolded, not finished**.
5. **Backend reality** stamped on T0.1 return.
