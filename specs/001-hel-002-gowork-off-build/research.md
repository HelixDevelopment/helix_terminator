# Phase 0 Research: HEL-002 (notification-service builds outside the workspace)

**Date**: 2026-07-23 · **Plan**: [plan.md](plan.md)

All findings below are MEASURED on this checkout (worktree
`.claude/worktrees/hel-002-provingground` @ `dddd323`), not assumed.

## R1 — Root cause of the GOWORK=off failure

- **Decision/Finding**: `GOWORK=off go build ./...` exits 1 with
  "updates to go.mod needed" because go.mod/go.sum omit the transitive
  requirements pulled in by the Herald replaces (slack-go/slack v0.16.0,
  bytedance/gopkg, gorilla/websocket, quic-go, goccy/go-yaml,
  mongo-driver/v2, …). Captured: `/tmp/hel002_red_build.log`,
  `/tmp/hel002_tidy_diff.log` (non-mutating `GOWORK=off go mod tidy -diff`,
  exit 1 = diff non-empty, 292 lines).
- **Rationale**: workspace mode supplies the union module graph + go.work.sum,
  masking the module's own incompleteness — exactly the drift class HEL-002
  records.
- **Alternatives considered**: none applicable (this is diagnosis, not choice).

## R2 — The gin pin is a fiction in workspace mode (drift proof)

- **Finding**: `go list -m github.com/gin-gonic/gin` → **v1.12.0** in workspace
  mode vs **v1.10.0** under `GOWORK=off` (captured in-session 2026-07-23). The
  workspace has been building v1.12.0 all along; the recorded v1.10.0 pin is
  stale. Therefore accepting the tidy bump does NOT change what the deployed
  workspace build already uses — it makes the record TRUE.
- **Rationale for accepting the bump**: MVS selects the maximum of
  requirements; herald/commons requires gin v1.12.0, so a correct module-mode
  graph cannot keep v1.10.0. The go.mod comment itself prescribes: "the
  resulting gin bump should be reviewed and tested on its own merits" — this
  item IS that review, with both-modes test evidence.
- **Alternatives considered**: (a) keep v1.10.0 + hand-pin exclusions —
  impossible under MVS with the herald requirement; (b) drop the herald import
  — reverses a shipped feature (§11.4.122 forbids silent capability removal);
  (c) replace slack-go to the local drifted v0.27.0 submodule — rejected, does
  not compile (recorded in the existing manifest comment; preserved as FR-005).

## R3 — Container path strategy

- **Decision**: repo-root build context; Dockerfile invoked as
  `docker build -f services/notification-service/Dockerfile .`; copy
  `services/notification-service` + `submodules/herald` into the builder
  stage; replaces stay.
- **Rationale**: standard monorepo pattern for local-replace modules; avoids
  vendoring a 171 MB drift-prone Herald copy; measured FACT: no docker-compose
  file exists in this repo at depth ≤ 3 outside submodules, so compose wiring
  is not editable here (stays on HEL-002 follow-up).
- **Alternatives considered**: `go mod vendor` (rejected: large frozen copy,
  drift risk, complicates the deliberate slack-go proxy resolution);
  per-service context with a pre-build "sync herald into context" script
  (rejected: hidden copy step, §11.4.6-unfriendly).

## R4 — Worktree submodule population (execution-environment prerequisite)

- **Decision**: native `git submodule update --init submodules/herald` followed
  by `git -C submodules/herald submodule update --init --recursive` (the exact
  remedy the manifest comment prescribes) — MEASURED working inside this linked
  worktree: herald @ 4b7a85e checked out (~90 s), 16 nested submodules
  populated (~80 s), `commons/go.mod` + `commons_messaging/go.mod` control
  needles present. Captured: `/tmp/hel002_submodule_init.log`,
  `/tmp/hel002_submodule_recursive.log`.
- **Rationale**: proper git wiring (no rsync shadow copies), same command every
  fresh checkout is already mandated to run.
- **Alternatives considered**: rsync from the main checkout excluding `.git`
  (rejected: shadow content invisible to git, diverges from the documented
  remedy).

## R5 — Go directive movement

- **Finding**: tidy moves the module directive `go 1.25.0` → `go 1.25.3`
  (from the tidy diff). Toolchain on host: go1.26.4 (go.work `go 1.26.4`) —
  satisfies both. No action needed beyond accepting tidy's output.

## R6 — Slack SDK non-replacement preservation

- **Finding**: tidy's computed graph resolves `github.com/slack-go/slack
  v0.16.0` from the public proxy (visible in the tidy diff as a normal
  require), NOT from the drifted local submodule — i.e. tidy PRESERVES the
  deliberate non-replacement automatically because no replace directive names
  it. FR-005 is satisfied by keeping the replace block as-is.
