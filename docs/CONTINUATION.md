# CONTINUATION — helix_terminator

**Revision:** 8
**Last modified:** 2026-07-07T00:00:00Z

Standing session-resumption record (Constitution §12.10 / §11.4.131). Keep current.

## One-line resume
**FULL DEVELOPMENT, autonomous multi-stream loop.** Real integration/security tests are being written against real infra (rootless podman) and are **surfacing + fixing real production defects that mocked unit tests hid**. Clean tree at HEAD `f272f31` (remote==local across origin/github/upstream). Resume: read THIS file, then `.superpowers/sdd/progress.md` (controller ledger — latest blocks = live queue, every SHA, every stream status), then `git fetch --all`, then continue the NEXT QUEUE below.

## What shipped this session (all pushed, remote==local; ~13 commits fa56e7c → f272f31)
- **CI/pipeline**: worktree-safe pre-commit gate + TRACKED reproducible hook installer `scripts/install_git_hooks.sh` (a subagent had silently replaced the untracked hook with a bypass probe; the git-worktree GIT_DIR bug had blocked every worktree commit — both fixed).
- **Security**: ssh-proxy pgx→v5.10.0 (GO-2026-5004 closed); gateway SSRF/path-injection reject+encode fix (from an automated review of the new proxy).
- **Real features + the defects their real tests exposed**:
  - migrate startup wiring (auth/user/org, real PG 17.2).
  - Flutter **real xterm terminal** replacing the stub (host-rendered golden PNGs light+dark).
  - **gateway** `proxyTo` was a fabricated-routing STUB → now a real reverse proxy (cross-the-wire test).
  - **pki** lifecycle test found+fixed 3 prod defects (jsonb insert, nullable scan, negative serial).
  - **auth** integration/security found+fixed 5 (logout unreachable/panic, **stateless-JWT-no-revocation**, register broken on real PG, ip_address scan, missing jti) + **/mfa** context bug + 2 MFA correctness defects.
  - **vault** ZK ciphertext-at-rest proof + added the auth/tenant-isolation middleware it lacked.

## MAJOR FINDING (S11 fleet stub-audit, 25/25 services) → register in scratchpad/exec/session_20260707/S11_stub_handler_register.md
9 handlers fabricate success with ZERO backing: **ai** CreateRequest (no LLM), **notification** email/push/webhook (no delivery), **billing** CreateSubscription "active" (no payment — DANGEROUS), 3 **bridges** (no infra client), gateway route-table mismatch (~19/20 services 404 through the sole ingress). Plus readiness-check bluffs (auth/user ready:true w/o DB ping) + dead gateway internal/handler.

## OPERATOR DECISIONS (2026-07-07) — now driving the work
1. **Fabricated handlers → IMPLEMENT REAL BACKENDS.** S15 researches per-service provider specs + credential needs; then implementation streams (some may be §11.4.21 operator-blocked pending credentials).
2. **DB isolation → SCHEMA-PER-SERVICE.** Each service migrates into its own PG schema (search_path) in the shared DB. S14 reworks auth/user/org + establishes the pattern; then roll out to the remaining 22.

## In flight (background streams)
- **S12** gateway route-table alignment (fix the ~19/20 mismatched upstream paths).
- **S14** schema-per-service migrate rework (auth/user/org + reusable pattern).
- **S15** real-backend provider research (read-only → impl plan + operator-input list).

## NEXT QUEUE (priority §11.4.132)
1. Integrate S12/S14/S15 on completion (review → merge-onto-latest-main → push → verify).
2. Implement real backends per S15 plan (priority: billing > ai/notification > bridges); surface credential needs.
3. Schema-per-service rollout to the remaining 22 services (after S14 pattern lands).
4. Follow-ups: T6 gateway dead internal/handler (§11.4.124 investigate); T7 vault ListSecrets/CreateSecret user_id trust; readiness-check honesty (auth/user ping DB); gateway fullHealthHandler hardcoded fields.
5. Curate coverage-ledger §11.4.25 → tracked docs/; QA evidence → docs/qa/ §11.4.83.
6. FINAL whole-branch review (fresh flutter analyze + govulncheck) + §11.4.40 full retest.

## Binding constraints
Anti-bluff §11.4 (captured physical evidence per closure). **No force-push; merge-onto-latest-main (§11.4.113).** git-direct scoped (no commit wrapper); pre-commit gate = inheritance + docs. **CI stays disabled (§11.4.156).** Constitution pinned `e6504c2`. Host 64 cores / 226 GiB free but **PID-constrained** (~4096 ulimit -u) → `GOMAXPROCS=2` + `go build -p 2`, one module at a time; **worktree-isolate every mutating agent (§11.4.84)**; capacity/session limits crash agents — checkpoint-commit early (§11.4.147).
**Worktree env gotchas (learned):** in a nested worktree `go env GOWORK` leaks the parent go.work → use `GOWORK=off` for go build/test/vet. Rootless podman PLAIN default userns only (`:z`/`--userns=keep-id`/`label=disable` CRASH — SELinux pwalkdir). Flutter needs `ulimit -u 5120`. Fresh worktree needs constitution submodule (`git submodule update --init constitution`, or wire `constitution/.git`→`.git/modules/constitution` if network times out).

## Resume infrastructure
- **This file** — read FIRST, then `git fetch --all`. Run `bash scripts/install_git_hooks.sh` after a fresh clone.
- `.superpowers/sdd/progress.md` — controller ledger (git-ignored); latest blocks = live queue + SHAs + stream statuses + T6–T9 + operator decisions.
- Evidence: `scratchpad/exec/session_20260707/S{1..15}_*` (git-ignored, session-local; curate to `docs/qa/` at release prep).
