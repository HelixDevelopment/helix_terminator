# Tasks: notification-service builds outside the workspace (HEL-002)

**Input**: plan.md (phases A–E), spec.md (US1 P1, US2 P2), research.md (R1–R6),
quickstart.md · **Branch**: `feat/hel-002-speckit-provingground` ·
**Date**: 2026-07-23

**Tests**: REQUESTED — Constitution Principle II (test-first) binds this work;
the RED verification harness precedes every fix and doubles as the GREEN gate.

**Execution context**: worktree `.claude/worktrees/hel-002-provingground` is
cwd for every task. Evidence lands under
`specs/001-hel-002-gowork-off-build/evidence/`. Never edit outside scope
(.specify/, specs/, services/notification-service). Capture real exit codes —
never through a pipeline tail.

## Phase 1: Setup (worktree prerequisite)

- [ ] T001 Verify `submodules/herald` + nested submodules are populated in this
      worktree (per research.md R4; commands already run) — re-verify control
      needles `submodules/herald/commons/go.mod` and
      `submodules/herald/commons_messaging/go.mod` exist and
      `git submodule status submodules/herald` shows commit 4b7a85e; record to
      `specs/001-hel-002-gowork-off-build/evidence/t001_prereq.log`

## Phase 2: Foundational (RED harness — MUST be RED before any fix)

- [ ] T002 Write the verification harness
      `specs/001-hel-002-gowork-off-build/evidence/verify_hel002.sh`: runs the
      full quickstart matrix (GOWORK=off build/vet/test + integration-tag vet;
      workspace build/vet/test + integration-tag vet; gin recorded-vs-built
      drift check both modes; `GOWORK=off go mod tidy -diff` stability;
      Docker-context replace-target reachability; real `docker build` iff a
      container runtime exists, else prints `SKIP container_runtime_absent`),
      each step capturing its own exit code to a per-step log + a summary
      verdict line, exiting non-zero if any mandatory step fails
- [ ] T003 Execute the harness on the UNFIXED tree and confirm it is RED for
      exactly the recorded reasons (GOWORK=off build FAIL, drift FAIL,
      tidy-diff FAIL, Dockerfile-reachability FAIL; workspace matrix PASS) —
      save output as
      `specs/001-hel-002-gowork-off-build/evidence/red_baseline.log`
      (RED provenance: observed, §11.4.115(G))

## Phase 3: User Story 1 — Standalone (module-mode) build works (P1) 🎯 MVP

**Goal**: `GOWORK=off` matrix green, drift genuinely resolved, tidy-stable.

**Independent test**: quickstart steps 1–4 all exit 0 on this checkout.

- [ ] T004 [US1] Run `GOWORK=off go mod tidy` in
      `services/notification-service/` (accepts the MVS-forced gin
      v1.10.0→v1.12.0 bump per research.md R2), then verify the replace block
      and the slack-go NON-replacement survived intact (R6/FR-005)
- [ ] T005 [US1] Update the explanatory comment block in
      `services/notification-service/go.mod`: supersede the "DELIBERATELY NOT
      `go mod tidy`'d" rationale with the HEL-002 acceptance record (bump now
      reviewed + tested on its own merits, both modes), PRESERVE the slack-go
      non-replacement rationale verbatim, keep the submodule-init remedy note
- [ ] T006 [US1] Re-run the T002 harness: module-mode matrix (build/vet/test +
      integration-tag vet), workspace matrix, drift check (both modes report
      gin v1.12.0; go.mod records v1.12.0), tidy-stability
      (`GOWORK=off go mod tidy -diff` exit 0, empty) — ALL GREEN; save
      `specs/001-hel-002-gowork-off-build/evidence/green_us1.log`

## Phase 4: User Story 2 — Container image build path is possible (P2)

**Goal**: repo-root build context reaches every replace target.

**Independent test**: quickstart step 5 — mechanical reachability 100%; real
image build green iff a runtime exists (else recorded SKIP).

- [ ] T007 [US2] Rewrite `services/notification-service/Dockerfile` for
      repo-root context (invoked `docker build -f
      services/notification-service/Dockerfile .`): builder stage copies
      `services/notification-service/` + `submodules/herald/` preserving the
      `../../submodules/herald` relative layout, `WORKDIR` at the service
      module, `GOWORK=off` build; header comment documents the invocation +
      context requirement
- [ ] T008 [US2] Run the harness's container leg: mechanical reachability check
      (every `../../` replace target in go.mod exists inside the repo-root
      context) must be 100%; attempt real `docker build` iff runtime present,
      else record honest `SKIP container_runtime_absent`; save
      `specs/001-hel-002-gowork-off-build/evidence/green_us2.log`

## Final Phase: Polish & evidence bundle

- [ ] T009 Full harness run end-to-end (all legs) GREEN; save
      `specs/001-hel-002-gowork-off-build/evidence/final_matrix.log`; verify
      `git status` shows ONLY in-scope paths changed
      (services/notification-service/{go.mod,go.sum,Dockerfile} + specs/ +
      .specify/); write
      `specs/001-hel-002-gowork-off-build/evidence/SUMMARY.md` mapping each
      SC-001..SC-005 to its evidence file + exit codes, including the honest
      status of the container-runtime leg and the OPEN siblings follow-up
      (HEL-002 stays non-terminal per §11.4.146(D3))

## Dependencies

- T001 → T002 → T003 (RED) → {T004 → T005 → T006} (US1) → {T007 → T008} (US2) → T009
- US2 depends on US1 only for the shared harness verdict (the Dockerfile edit
  itself is independent of the manifest fix but its build leg needs the fixed
  manifest to succeed) — execute sequentially; no [P] parallelism marked
  because every task touches the same service/harness surface.

## Implementation strategy

MVP = Phase 3 (US1): module-mode correctness is the recorded defect. US2 is a
small additive packaging change. Stop-on-RED discipline: any unexpected
failure triggers systematic-debugging (Constitution Principle V) before
proceeding.
