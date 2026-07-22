# Implementation Plan: notification-service builds outside the workspace (HEL-002)

**Branch**: `feat/hel-002-speckit-provingground` (spec dir `001-hel-002-gowork-off-build`) | **Date**: 2026-07-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-hel-002-gowork-off-build/spec.md`

## Summary

`services/notification-service` is workspace-only: its go.mod/go.sum lack the
Herald transitive requirements (GOWORK=off build exits 1) and record a stale
gin v1.10.0 pin while the workspace really builds v1.12.0; its Dockerfile's
service-local context cannot reach `../../submodules/herald`. Approach: make
the manifest module-mode-correct with `GOWORK=off go mod tidy` (accepting the
MVS-forced gin v1.12.0 bump, reviewed + tested on its own merits exactly as the
manifest comment prescribes), update the manifest's recorded rationale, switch
the Dockerfile to a repo-root build context, and prove everything with captured
exit-code evidence in BOTH build modes (RED first — the failing baseline is
already captured).

## Technical Context

**Language/Version**: Go (toolchain go1.26.4; module directive currently
`go 1.25.0`, tidy will move it to `go 1.25.3`)

**Primary Dependencies**: gin-gonic/gin (v1.10.0 → v1.12.0, MVS-forced by
herald/commons), Herald modules via local replaces rooted at
`submodules/herald` (commons, commons_messaging, commons_infra,
commons_storage + digital.vasic.* + telebot), slack-go/slack v0.16.0
(deliberately NOT replaced — resolved from public proxy, preserved per FR-005)

**Storage**: N/A (build/packaging change only; no runtime behaviour change
intended — proven by tests, not assumed)

**Testing**: `go build` / `go vet` / `go test` (default tags) in BOTH modes
(workspace + GOWORK=off); `go vet -tags integration` compile gate in both
modes; tidy-stability check (`go mod tidy -diff` empty); Docker-context
reachability check (mechanical); real `docker build` only if a runtime exists
on this host (else honest SKIP)

**Target Platform**: Linux dev/CI hosts + container image (distroless)

**Project Type**: Go microservice inside a go.work monorepo (build-infra fix)

**Performance Goals**: N/A (no runtime perf change; build must simply succeed)

**Constraints**: No behaviour regression in workspace mode; tidy-stable
manifest; deliberate slack-go non-replacement preserved; scope =
notification-service ONLY (siblings stay on HEL-002 follow-up); NO edits
outside the worktree scope (.specify/, specs/, services/notification-service)

**Scale/Scope**: 1 service, 2 files expected to change
(`services/notification-service/go.mod`, `go.sum`) + `Dockerfile` + evidence;
worktree prerequisite: `submodules/herald` content must be present in THIS
worktree (git worktrees do not auto-populate submodules — established as
Phase A task, method per research.md R4)

## Constitution Check

*GATE: evaluated against `.specify/memory/constitution.md` v1.0.0 (derived
projection; canonical authority = constitution submodule).*

| Principle | Gate | Status |
|---|---|---|
| I. Anti-Bluff Evidence | Every claim backed by captured exit codes/logs; no pipeline-tail exit codes; null results control-needled | PASS — evidence files planned per task; RED baseline already captured (`/tmp/hel002_red_build.log`, `/tmp/hel002_tidy_diff.log`) |
| II. Test-First for All Work | RED observed before fix | PASS — RED is OBSERVED provenance (§11.4.115(G)): real `GOWORK=off go build` exit 1 captured pre-change; the same commands flip GREEN post-change; verification script husk is written & run RED before the manifest change lands |
| III. Independent Review | Fable xhigh review before acceptance | PASS — scheduled as the final lifecycle step before commit |
| IV. SSoT & Traceability | Spec/plan/tasks cite HEL-002; no terminal status flip without custody chain | PASS — HEL-002 cited throughout; item stays non-terminal (siblings + container-runtime leg open) |
| V. Systematic Debugging | Root cause proven before fix | PASS — root cause established from `go mod tidy -diff` + `go list -m` divergence + Dockerfile context read, not guessed |
| Safety & Integrity | No force-push; sanctioned commit path; no constitution/ edits; host safety | PASS — commit lands on feat branch in isolated worktree; constitution/ untouched; builds are small (single service) |

**Violations requiring Complexity Tracking**: none.

**Post-Phase-1 re-check (2026-07-23)**: design artifacts introduce no new
violations. PASS.

## Project Structure

### Documentation (this feature)

```text
specs/001-hel-002-gowork-off-build/
├── spec.md              # Feature spec (done)
├── plan.md              # This file
├── research.md          # Phase 0 output (done)
├── quickstart.md        # Phase 1 output (validation guide)
├── checklists/
│   └── requirements.md  # Spec quality checklist (done, 16/16)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

(`data-model.md` and `contracts/` are intentionally omitted: this is a pure
build/packaging fix with no data entities beyond the manifests themselves and
no externally exposed interface — per the plan skill's "skip if purely
internal" rule. The manifests' before/after states are documented in
research.md instead.)

### Source Code (repository root)

```text
services/notification-service/
├── go.mod        # CHANGED — tidy-correct requires + go directive + updated comments
├── go.sum        # CHANGED — herald transitive checksums added, stale entries dropped
├── Dockerfile    # CHANGED — repo-root build context (-f invocation), copies submodules/herald
├── cmd/…         # unchanged
└── internal/…    # unchanged (incl. internal/delivery/slack_herald.go)

submodules/herald/…   # READ-ONLY prerequisite (populated in worktree; never edited)
```

**Structure Decision**: single-service change inside the existing monorepo
layout; no new directories, no new projects.

## Phase Outline (consumed by /speckit-tasks)

- **Phase A — Worktree prerequisite**: populate `submodules/herald` content in
  this worktree (R4 method), verify with a control needle (a known herald
  go.mod path exists).
- **Phase B — RED verification harness**: write + run the verification script
  capturing the CURRENT failing state (GOWORK=off build exit 1; version-drift
  detector reports drift; Docker-context reachability reports unreachable) —
  all three must be RED for the recorded reasons before any fix lands.
- **Phase C — Manifest fix**: `GOWORK=off go mod tidy`; update the go.mod
  comment block (supersede the "deliberately not tidied" rationale, preserve
  the slack-go non-replacement rationale); verify tidy-stability.
- **Phase D — Dockerfile fix**: repo-root context variant; mechanical
  reachability proof; real `docker build` iff runtime present (else recorded
  SKIP).
- **Phase E — GREEN verification + evidence**: full both-modes matrix green;
  evidence bundle written under specs/001-hel-002-gowork-off-build/evidence/.

## Complexity Tracking

No Constitution Check violations — table intentionally empty.
