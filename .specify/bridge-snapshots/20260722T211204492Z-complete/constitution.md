<!--
Sync Impact Report — 2026-07-23
Version change: (unfilled template) → 1.0.0 (initial ratification of the projection)
Modified principles: n/a (initial fill — template placeholders replaced)
Added sections: Derived-Projection Authority Notice; Principles I–V; Additional
  Constraints (Safety & Integrity); Development Workflow (Spec Kit / Superpowers
  ownership split); Governance
Removed sections: none (template structure preserved)
Templates status:
  - .specify/templates/plan-template.md — "Constitution Check" section consumes
    the principle gates below unchanged; no edit required
  - .specify/templates/spec-template.md — no constitution-specific sections; no
    edit required
  - .specify/templates/tasks-template.md — TDD-ordered task flow already matches
    Principle II; no edit required
  - .claude/skills/speckit-* — generic (agent-neutral) wording verified; no edit
Deferred TODOs: none — every bracketed ALL-CAPS placeholder is filled.
Sources read for this projection (2026-07-23): constitution/CLAUDE.md,
  constitution/Constitution.md (canonical), CLAUDE.md, AGENTS.md, README.md and
  its §11.4.212-reachable documentation tree, docs/CONSTITUTION_INHERITANCE.md,
  docs/guides/HELIX_TERMINATOR_CONSTITUTION.md.
-->

# helix_terminator Constitution

> **DERIVED PROJECTION — NOT THE AUTHORITY.** This file is a derived projection
> of the Helix Constitution for Spec Kit templating only. The Helix Constitution
> submodule mounted at `constitution/` (`constitution/Constitution.md` +
> `constitution/CLAUDE.md` + `constitution/AGENTS.md`) remains the sole
> canonical authority (Constitution §11.4.35). If anything in this file
> disagrees with the submodule, **the submodule wins** — this projection must
> then be regenerated from the sources, never patched into divergence.
> Project-specific extensions live in `CLAUDE.md`, `AGENTS.md`, and
> `docs/guides/HELIX_TERMINATOR_CONSTITUTION.md` (currently: no overrides in
> force).

## Core Principles

### I. Anti-Bluff Evidence (NON-NEGOTIABLE)

Every claim of working behaviour MUST carry positive captured evidence produced
by really executing the thing claimed (§11.4, §11.4.5, §11.4.69). Metadata-only,
config-only, absence-of-error, and grep-without-runtime PASSes are critical
defects. FAIL-bluffs are equally forbidden (§11.4.1): a check may fail only for
a genuine defect, never for an instrument bug — and a zero/absence result is not
evidence until the instrument is proven able to see (§11.4.201 control needle;
exit codes read from the real command, never from a pipeline tail). Guessing
vocabulary ("likely", "probably", "seems") is forbidden in causes and closures
(§11.4.6): prove it, or mark it UNKNOWN:/PENDING_FORENSICS: with a tracked
follow-up.

### II. Test-First for All Work (NON-NEGOTIABLE)

TDD binds ALL executable work, not only fixes (§11.4.224): the test is written
first, run first, and observed to FAIL for the right reason (RED on the broken
or absent behaviour, §11.4.115) before implementation exists. A test authored
after its implementation proves only agreement, never detection. RED
preconditions MUST be traceable to the real defect's evidence
({observed|constructed} provenance, §11.4.115(G)); a constructed precondition
mints defensive hardening, never a defect-closing fix. Coverage floor >=85%
(~100% target) is a NECESSARY, never SUFFICIENT proxy — the real bar is a test
that catches its own negation.

### III. Independent Review Before Acceptance (NON-NEGOTIABLE)

Every change — source, tests, docs, config, one-liners — passes an independent
code review BEFORE acceptance/commit/build (§11.4.142, no carve-out). Reviews
enumerate the full input/scenario space, prove every "can't happen" assumption,
and cross-check captured runtime evidence (§11.4.194); they iterate to a
zero-finding, zero-warning GO (§11.4.134). The review substrate is pinned:
Fable model at xhigh effort, Opus xhigh as the only fallback on proven Fable
unavailability (§11.4.209).

### IV. Single Source of Truth & Traceability

Workable items live in the tracked SQLite SSoT `docs/workable_items.db`
(§11.4.93/§11.4.95); every derived document regenerates FROM it. Items carry
stable never-renumbered ids (§11.4.54), and specs/plans/tasks MUST cite the item
id they implement. Status custody (§11.4.146(D3)): a done/ready status write is
refused without the chain registered guard → RED+GREEN verdict pair →
class-matched evidence — an honest open item beats a false close. A returning
defect reopens its existing item, never mints a new id (§11.4.214). Progress is
journaled to the durable session ledger so no crash loses work (§11.4.147).

### V. Systematic Debugging & Root Cause First

NO FIXES WITHOUT ROOT-CAUSE INVESTIGATION FIRST (§11.4.102 Iron Law,
auto-activated on any spotted issue — no operator prompt needed). When a working
reproduction exists, investigations MUST drive its exact sequence (§11.4.199);
a deviating repro that never reaches the precondition proves nothing.
Regressions are isolated against the last known-good state first (§11.4.114).
Deep web research precedes non-trivial fixes (§11.4.8) and every unclear
validation path (§11.4.123) — "untestable" is a research trigger, never a bluff
licence.

## Additional Constraints — Safety & Integrity

- Force-push is absolutely forbidden, everywhere, with no operator-approval
  escape (§11.4.113); integration is merge-onto-latest-main, fast-forward
  pushes to all upstreams.
- Commits go through the project's sanctioned commit path; never `git add -A`
  in shared checkouts; never commit while a concurrent mutation gate or another
  stream's scope is in flight (§11.4.84, §11.4.121).
- The constitution submodule at `constitution/` is never edited as a side
  effect of project work (§11.4.26 pipeline only); owned submodules stay
  decoupled and project-unaware (§11.4.28).
- Host safety §12 binds unconditionally: <=60% RAM (§12.6), thread-headroom
  awareness (§12.12), no host power/session interference.
- Inheritance gate `bash tests/verify_constitution_inheritance.sh` (paired
  §1.1 mutation proof `scripts/testing/meta_test_false_positive_proof.sh`)
  runs before every build/merge.

## Development Workflow — Spec Kit / Superpowers Ownership Split

Spec Kit owns WHAT: `spec.md`, `plan.md`, and `tasks.md` under `specs/` are the
design single-source-of-truth for a feature, produced by the speckit lifecycle
(specify → clarify → plan → tasks → analyze). Superpowers owns HOW: execution of
`tasks.md` runs under Superpowers discipline (test-driven-development RED-first,
systematic-debugging, verification-before-completion, requesting-code-review)
via the speckit-superpowers-bridge. `speckit.implement` is SUPERSEDED and
FORBIDDEN as the executor once a handoff exists; `superpowers:brainstorming` and
`superpowers:writing-plans` are FORBIDDEN once `spec.md` + `plan.md` exist (the
bridge guard enforces both; its allow/deny history is the audit trail in
`.specify/bridge-events.jsonl`). Every spec MUST embed the stable workable-item
id it implements and reference the governing docs.

## Governance

This projection is subordinate governance: the Helix Constitution submodule
supersedes it on every conflict (submodule-wins), and the project files
`CLAUDE.md` / `AGENTS.md` / `docs/guides/HELIX_TERMINATOR_CONSTITUTION.md`
extend but never weaken it. Amendments happen ONLY by regenerating this
projection from the canonical sources (constitution submodule + project
governance files) — never by free-hand edits that diverge from them — and are
versioned semantically (MAJOR: principle removal/redefinition; MINOR: new or
materially expanded principle/section; PATCH: wording). Every plan's
"Constitution Check" gate MUST verify compliance against Principles I–V and the
Safety & Integrity constraints; violations require explicit justification in the
plan's Complexity Tracking table or the work does not proceed. Reviews verify
compliance per Principle III. Runtime development guidance lives in `CLAUDE.md`
(agent-facing) and the README documentation tree (§11.4.212 closure).

**Version**: 1.0.0 | **Ratified**: 2026-07-23 | **Last Amended**: 2026-07-23
