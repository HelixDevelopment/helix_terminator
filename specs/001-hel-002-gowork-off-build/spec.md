# Feature Specification: notification-service builds outside the workspace (HEL-002)

**Feature Branch**: `feat/hel-002-speckit-provingground` (spec dir `001-hel-002-gowork-off-build`)

**Created**: 2026-07-23

**Status**: Draft

**Workable item**: **HEL-002** (`docs/workable_items.db` — the §11.4.93/§11.4.95 SSoT; the
item's recorded description is the authoritative statement of this problem)

**Governing docs**: `constitution/Constitution.md` + `constitution/CLAUDE.md`
(canonical authority, submodule-wins), project `CLAUDE.md`/`AGENTS.md`,
`.specify/memory/constitution.md` (derived projection v1.0.0)

**Input**: User description: "HEL-002 — Make notification-service buildable outside the
Go workspace (GOWORK=off) and via its container build path" (full dispatch text in the
conversation; problem statement below restates the SSoT item).

## Problem Statement (from the SSoT item, verified 2026-07-23)

After the Slack notification channel wired a direct Herald import via the Go
workspace plus local `replace` directives, notification-service became
**workspace-only**:

- `GOWORK=off go build ./...` exits 1 ("updates to go.mod needed") — the module's
  own manifest lacks the Herald-module transitive requirements
  (captured: `/tmp/hel002_red_build.log`).
- The recorded dependency pin is stale and masked: the manifest records the HTTP
  framework at v1.10.0 while the workspace actually selects and builds v1.12.0
  (captured: `go list -m` divergence, workspace vs `GOWORK=off`).
- The service's container build (build context = the service directory, `COPY . .`)
  cannot reach `../../submodules/herald`, so the container image cannot be built
  at all for this service.

The same local-replace-to-out-of-context-submodule CLASS exists in at least 3
sibling services (helixtrack-bridge-service → `submodules/auth`,
container-bridge-service → `submodules/containers`, ai-service →
`submodules/llmprovider`). **Implementation scope here is notification-service
only**; the siblings are recorded for the pattern and follow-up (see Out of
Scope).

## Clarifications

### Session 2026-07-23

Run autonomously (proving-ground; §11.4.101 decision rule applied — reversible,
evidence-determined, bounded blast radius). Decisions recorded, not guessed:

- Q: Which container-path strategy satisfies FR-004 — change the build context
  to the repository root, or eliminate the local replaces (vendoring)?
  → A: Repo-root build context. The Dockerfile is updated to be invoked as
  `docker build -f services/notification-service/Dockerfile .` from the repo
  root, copying the service AND `submodules/herald` into the build; the
  replace directives stay. Vendoring is rejected: it would freeze a large,
  drift-prone copy of Herald into the service tree. NOTE (recorded fact, not a
  guess): no docker-compose file exists in this repository at depth ≤3 outside
  submodules — the SSoT item's compose reference points at wiring that is not
  present in-repo today; compose wiring therefore stays with the HEL-002
  follow-up, and this spec proves context-reachability mechanically plus a real
  image build ONLY where a container runtime is available.
- Q: What test scope proves the both-modes acceptance — default tags only, or
  integration tags too?
  → A: Default-tag `go build` + `go vet` + `go test` in BOTH modes are
  mandatory gates; the integration-tagged tree additionally must COMPILE in
  both modes (`go vet -tags integration`), while RUNNING integration tests
  stays out of scope (needs live infra; honest skip recorded per §11.4.3).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Standalone (module-mode) build works (Priority: P1)

A developer or CI job builds notification-service on a clean checkout WITHOUT
the workspace (`GOWORK=off`), e.g. for per-service CI, vendoring audits, or any
tooling that operates on the module alone — and the build, vet, and tests all
pass, selecting the SAME dependency versions the workspace selects (no hidden
drift between what is recorded and what is actually built).

**Why this priority**: this is the defect as recorded — the module is currently
un-buildable outside the workspace, and the recorded pins are a fiction relative
to what actually ships. Every downstream consumer (CI, container build,
security scanning of the real dependency set) depends on it.

**Independent Test**: on a clean checkout with submodules initialized, run
`GOWORK=off go build ./...`, `GOWORK=off go vet ./...`, `GOWORK=off go test ./...`
in the service directory; all exit 0. Then run the same in workspace mode; all
exit 0 with the identical selected version of every shared dependency.

**Acceptance Scenarios**:

1. **Given** a clean checkout with submodules initialized, **When** the service is
   built with `GOWORK=off`, **Then** build/vet/tests exit 0.
2. **Given** the same checkout, **When** the service is built in workspace mode,
   **Then** build/vet/tests exit 0 and the selected version of every shared
   dependency equals the `GOWORK=off` selection (drift genuinely resolved, not
   re-masked).
3. **Given** the updated manifest, **When** the recorded pins are compared with
   what the build actually selects, **Then** they match (no stale-pin fiction).

---

### User Story 2 - Container image build path is possible (Priority: P2)

A release engineer builds the notification-service container image and the
build can resolve every dependency, including the Herald modules that live
outside the service directory.

**Why this priority**: the container path is half of the recorded acceptance,
but it depends on P1's dependency-graph correctness and on a build-context
decision that affects packaging only, not code correctness.

**Independent Test**: the container build definition, when executed with its
declared context, reaches every `replace` target (proven either by a real image
build where a container runtime is available, or — where no runtime is
available in the execution environment — by a mechanical simulation of the
build context that verifies every path the build copies/resolves exists inside
the context, with the limitation recorded honestly).

**Acceptance Scenarios**:

1. **Given** the (possibly updated) container build definition, **When** the build
   context is assembled, **Then** every local `replace` target is reachable
   inside it.
2. **Given** a host with a container runtime, **When** the image build runs,
   **Then** it completes green. (If no runtime is available where this work
   executes, this scenario is recorded as an honest SKIP-with-reason —
   §11.4.3/§11.4.69 `artifact_not_yet_built`-class honesty — and stays owed on
   the item; it is NEVER faked.)

---

### Edge Cases

- Herald submodule not initialized on the checkout: the failure message must be
  the well-known "missing directory" class, and the documented remedy
  (`git submodule update --init --recursive submodules/herald`) must be recorded
  in the manifest comment — not silently broken.
- A future `go mod tidy` run by an unrelated change: must now be a no-op (the
  manifest is tidy-stable), so the drift cannot silently reappear.
- Workspace users: the workspace build must keep working unchanged — fixing the
  module-mode path must not break the workspace path.
- The dependency bump (HTTP framework v1.10.0 → v1.12.0) is forced by module
  graph rules once Herald's requirements are recorded; its blast radius (every
  HTTP endpoint of this service) must be covered by the service's existing
  tests re-running green in both modes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The service MUST build, vet, and pass its tests with `GOWORK=off`
  (module mode) on a clean checkout with submodules initialized.
- **FR-002**: The service MUST continue to build, vet, and pass its tests in
  workspace mode, and the integration-tagged tree MUST compile in both modes
  (`go vet -tags integration`); running integration tests is out of scope
  (live infra; honest skip per §11.4.3).
- **FR-003**: The dependency versions recorded in the module manifest MUST equal
  the versions actually selected in BOTH build modes (drift resolved, not
  re-masked; tidy-stable manifest).
- **FR-004**: The container build definition MUST be able to reach every local
  `replace` target from within its build context. Per Clarification 2026-07-23:
  the build context becomes the repository root
  (`docker build -f services/notification-service/Dockerfile .`), the replaces
  stay, and reachability is proven mechanically (every replace target exists
  inside the context) plus a real image build where a container runtime exists.
- **FR-005**: The existing deliberate non-replacement of the public Slack SDK
  (pinned by Herald's own requirement) MUST be preserved — the module keeps
  resolving it from the public proxy, per the manifest's recorded rationale.
- **FR-006**: The manifest's explanatory comments MUST be updated to reflect the
  new reality (the "deliberately not tidied" rationale is superseded by this
  item's acceptance; the bump is now reviewed and tested on its own merits, per
  that comment's own instruction).
- **FR-007**: Every verification MUST produce captured evidence (real command
  exit codes + logs), per Constitution Principle I.

### Key Entities

- **Module manifest (go.mod/go.sum)**: records the service's dependency
  requirements; currently stale relative to the real selection.
- **Workspace (go.work)**: the repo-level build mode that masked the drift.
- **Herald modules**: `commons`, `commons_messaging` (+ transitive local
  submodule replaces) rooted at `submodules/herald`.
- **Container build definition (Dockerfile)**: service-local context that cannot
  currently see `../../submodules/herald`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `GOWORK=off` build/vet/test and workspace build/vet/test all exit
  0 on the same checkout (currently: `GOWORK=off` build exits 1).
- **SC-002**: Recorded-vs-built dependency versions match 100% (currently: the
  HTTP framework diverges v1.10.0 recorded vs v1.12.0 built).
- **SC-003**: A repeat run of the manifest-tidy operation produces zero changes
  (tidy-stable — the drift cannot silently return).
- **SC-004**: The container build context reaches 100% of local replace targets
  (currently 0% of the Herald targets are reachable).
- **SC-005**: All evidence is captured to files with real exit codes (no
  pipeline-tail exit codes), reviewable after the fact.

## Assumptions

- Submodule initialization (`submodules/herald`, recursive) is an already-mandated
  property of every checkout (per the manifest's own recorded note and the
  project's submodule mandate); this item does not remove that prerequisite for
  the module-mode path.
- The HTTP-framework bump v1.10.0 → v1.12.0 is unavoidable for a correct
  module-mode graph (module graph rules select the maximum of requirements once
  Herald's requirements are recorded); it is accepted HERE, reviewed and tested
  on its own merits, exactly as the manifest comment prescribed.
- Sibling services (helixtrack-bridge-service, container-bridge-service,
  ai-service) are OUT of implementation scope; the pattern record + follow-up
  stays on HEL-002 in the SSoT (the item is NOT closed by this spec alone).
- A container runtime may not be available in the execution environment; the
  container-path proof degrades to a mechanical context-reachability check plus
  an honest recorded SKIP for the real image build (never a faked pass).
- This run executes in the worktree `.claude/worktrees/hel-002-provingground`
  (branch `feat/hel-002-speckit-provingground`); all lifecycle scripts run with
  that worktree as cwd.

## Out of Scope

- Fixing the sibling services' identical pattern (recorded for follow-up on
  HEL-002; the item stays open until every identified sibling is green).
- Herald-repo-internal submodule-pin drift (the Slack SDK submodule checked out
  at a tag ahead of what Herald's code compiles against) — explicitly recorded
  in the manifest as out of scope.
- Closing HEL-002 in the SSoT (status custody §11.4.146(D3): requires the full
  guard → RED+GREEN verdict → evidence chain across ALL acceptance including
  siblings; this spec's increment alone does not satisfy it).
