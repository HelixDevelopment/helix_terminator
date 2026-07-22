# Workable Items — §11.4.202 reporting directives + §11.4.93/§11.4.95 SQLite SSoT

**Revision:** 1
**Last modified:** 2026-07-22T15:01:00Z

This guide documents how helix_terminator consumes the constitution-shipped
`ISSUE:` / `BUG:` / `TASK:` / `FEATURE:` reporting directives and the
`workable-items` SQLite single-source-of-truth. Both mechanisms are inherited
**by reference** from `constitution/` (§11.4.28 / §11.4.177) — nothing in
this project reimplements them; this document + `.helix/reporting.yaml` are
the project's own consumer-owned wiring around them.

## What lands where

```
ISSUE: <text>  /  BUG: <text>  /  TASK: <text>  /  FEATURE: <text>
   (any of the six §11.4.140 grammar forms — see constitution/actions/registry.yaml)
        │
        ▼
constitution/scripts/reporting/report_item.sh   (§11.4.202 engine, inherited by reference)
        │  reads .helix/reporting.yaml (this project's consumer config)
        ▼
  (1) CREATE  → docs/workable_items.db  (§11.4.93/§11.4.95 SQLite SSoT)
                 Type∈{Bug,Feature,Task} + Status=Queued + stable HEL-NNN id
                 + comprehensive structured description (§11.4.148/§11.4.171)
  (2) SYNC    → NOT YET WIRED here — honest SKIP (see below)
  (3) PUSH    → NOT YET WIRED here — zero trackers configured, honest no-op
```

`ISSUE:` / `BUG:` / `TASK:` create the item directly. `FEATURE:` (§11.4.213)
additionally **schedules** — never synchronously runs — a deep
research-and-planning effort by appending to `docs/requests/feature_queue.md`
and creating a Type=Task item via this SAME engine (no second, divergent
item-creation path).

## Where the pieces live

| Piece | Path | Owner |
|---|---|---|
| Reporting engine (script) | `constitution/scripts/reporting/report_item.sh` | constitution submodule — inherited by reference, never copied |
| Reporting engine (companion doc) | `constitution/docs/scripts/report_item.md` | constitution submodule |
| §11.4.140 grammar + action registry | `constitution/actions/registry.yaml` | constitution submodule |
| `workable-items` DB-SSoT binary | `constitution/scripts/workable-items/bin/workable-items` (pre-built) + `cmd/workable-items/` (source) | constitution submodule |
| **This project's consumer config** | `.helix/reporting.yaml` | **this project** — consumer-owned DATA |
| **This project's DB** | `docs/workable_items.db` | **this project** — tracked in git (§11.4.95) |

## Consumer config: `.helix/reporting.yaml`

Copied from `constitution/scripts/reporting/reporting.example.yaml` and
filled in with this project's values. Key decisions, with the evidence
behind each (§11.4.6 — no guessing):

- **`db: docs/workable_items.db`** — the canonical path, matching
  `reporting.example.yaml`'s own default and §11.4.95 ("the DB at
  `docs/workable_items.db` is TRACKED in git, NEVER gitignored").
- **`id_prefix: HEL`** — EXPLICITLY PINNED rather than left to the binary's
  runtime auto-derivation, so the prefix can never silently drift.
  Evidence chain (full detail + citations in the config file's own
  comments):
  1. §11.4.151 resolution order checked in this project: no
     `HELIX_RELEASE_PREFIX` env var; no `.env` file anywhere in this
     project (only `constitution/.env.example` carries a **commented**
     example for the constitution submodule itself, `helix_constitution`
     — not applicable here); falls through to the lowercased snake_case
     project-root directory name, `helix_terminator`.
  2. No pre-existing ticket-tracker-prefix convention exists in this
     project. The only near-miss, `HT-SSHCA-20260714`
     (`.superpowers/sdd/progress.md`, the S-PKI-PLAN entry), is a
     §11.4.83 QA **run-id directory name** (`docs/qa/<run-id>/`), a
     completely different ID namespace — not a ticket prefix. No
     `docs/Issues.md`/`Fixed.md` tracker has ever existed here to have
     minted one.
  3. The `workable-items` binary's own §11.4.151 derivation
     (`defaultKeyPrefix()` in `constitution/scripts/workable-items/cmd/workable-items/prefix.go`),
     run live from this project's root during this session, derived
     exactly `HEL` from `helix_terminator` (first 3 ASCII letters,
     uppercased). Verified live — see "End-to-end proof" below.
- **`sync_command: ""`** and **`trackers: []`** — see "What is NOT yet
  wired" below.
- **`evidence_dir: qa-results/reporting`** — the shipped default; every
  directive run writes `result.json` + `create.log` (+ `sync.log` /
  `tracker_<name>.log` when those steps run) here (§11.4.5/§11.4.69
  captured evidence).

## The DB: `docs/workable_items.db`

- **Tracked in git, never gitignored** (§11.4.95) — it IS authoritative
  source data, not a build artifact. Only its transient `.db-wal` /
  `.db-shm` sidecars are gitignored (see `.gitignore`); a
  `PRAGMA wal_checkpoint(TRUNCATE)` runs before every commit-stage of this
  DB so those sidecars are always safely discardable at commit time.
- **Bootstrap mechanism**: the engine (`report_item.sh`) does **not**
  create the DB itself — it requires the DB file to already exist
  (`reporting.example.yaml`'s own comment: "MUST already exist"). The
  `workable-items` binary IS the bootstrap tool: **any** subcommand run
  against a non-existent path applies the full schema via `openDB()`
  (`CREATE TABLE IF NOT EXISTS …`, idempotent). This project's DB was
  bootstrapped with:
  ```bash
  constitution/scripts/workable-items/bin/workable-items validate --db docs/workable_items.db
  ```
  `validate` was chosen specifically because it creates the schema
  **without inserting any row** — the DB now exists with the full
  §11.4.93 schema (`items`, `item_history`, `doc_segments`,
  `logic_groups`, `obsolete_details`, `operator_block_details`,
  `firebase_metadata`, `test_diary`, `test_diary_summary`, `meta`) and
  **zero items** (`validate: OK — 0 items, all invariants satisfied`).
  No historical items were imported in this pass — that is explicitly a
  separate, future migration task (see below).

## End-to-end proof (real binary, real SQLite DB, no mocks — §11.4.27)

Every claim above was proven live in this session against a **temporary**
copy of the DB (never the real `docs/workable_items.db`, which stayed at
0 items throughout testing except for a final `--dry-run` acceptance check
that writes nothing):

| Test | Command shape | Result |
|---|---|---|
| Prefix derivation | `workable-items add Bug Medium --db <temp>` (no `--prefix`, run from project root) | `HEL-001` — confirms the `id_prefix: HEL` pinned in config matches the binary's own derivation |
| `BUG:` report | `report_item.sh --kind bug --config .helix/reporting.yaml --db <temp>` | Real row landed: `HEL-001`, Type=Bug, Status=Queued, `sync.verdict=SKIPPED` (honest, reason cited), `trackers=[]` |
| `TASK:` report | `report_item.sh --kind task ...` | `HEL-002`, Type=Task, Status=Queued |
| `ISSUE:` report, explicit type | `report_item.sh --kind issue --type Feature ...` | `HEL-003`, Type=Feature, Status=Queued |
| `ISSUE:` report, `--autonomous` (no type) | `report_item.sh --kind issue --autonomous ...` | `HEL-004`, Type=Task, `classification_note` recorded verbatim: `"defaulted-to-Task (§11.4.16 ambiguity default — RECLASSIFY once the type is determined)"` |
| Tracker honest-skip | temp config with two unavailable trackers (unset required env; empty command) + `report_item.sh --kind bug ...` | `HEL-005` created; `trackers: [{"name":"fake_tracker_no_creds","verdict":"SKIP","reason":"credentials_absent: unset env: FAKE_TRACKER_TOKEN_NEVER_SET"},{"name":"fake_tracker_no_client","verdict":"SKIP","reason":"tracker_client_absent: no command configured for 'fake_tracker_no_client' (PENDING-OPERATOR-INPUT — never faked, §11.4.6)"}]` — item creation was **never blocked** by tracker absence |
| Real-config, real-DB dry-run acceptance | `report_item.sh --kind bug --config .helix/reporting.yaml --dry-run` | `DRY-RUN — nothing written`, exit 0, resolved `db=.../docs/workable_items.db` correctly; real DB confirmed still at 0 items afterward |

Full JSON transcripts + `result.json`/`create.log` evidence for every row
above are captured in the delivering session's scratchpad report (not
committed to this repo — they reference temp-DB-only ids that do not exist
in the real `docs/workable_items.db` and would be misleading if left in
`qa-results/`).

## What is NOT yet wired (honest gaps — §11.4.6, tracked as follow-up)

1. **Historical-item migration.** No pre-existing Issues have been
   imported into `docs/workable_items.db`. This project has never had a
   `docs/Issues.md` / `docs/Fixed.md` tracker, so there is nothing to
   migrate FROM yet — but any future decision to backfill known issues
   (e.g. from `docs/research/mvp/REMEDIATION_REGISTER.md` or
   `.superpowers/sdd/progress.md`) is a **separate, deliberate task**, not
   a byproduct of this consumer-wiring pass.
2. **`sync_command` (§11.4.202 step 2 / §11.4.106 doc-regen) — left
   empty.** This project has no `docs/Issues.md` / `docs/Fixed.md`
   generation pipeline and no HTML/PDF/DOCX export pipeline
   (§11.4.65/§11.4.106 docs_chain) — the `workable-items sync db-to-md`
   subcommand exists and COULD regenerate `Issues.md`/`Fixed.md` alone,
   but wiring only that here would (a) silently create the *first-ever*
   `Issues.md`/`Fixed.md` for this project as a side effect of a
   narrowly-scoped wiring task, and (b) still leave the HTML/PDF/DOCX
   siblings unregenerated — a **partial** sync that would read as
   complete. An empty `sync_command` makes the engine report an honest
   `SKIPPED` verdict (§11.4.3) every time instead. Wiring the full
   doc-regeneration pipeline (`Issues.md`/`Fixed.md` generation +
   HTML/PDF/DOCX export, most naturally via `workable-items sync db-to-md`
   plus this project's own export tooling once one exists) is tracked as
   follow-up work.
3. **External trackers (§11.4.148 D5) — zero configured.** No tracker
   credentials exist anywhere in this project's environment (no `.env` at
   all) and no external tracker client (GitHub Issues API, HelixTrack,
   etc.) is wired. `trackers: []` in the config is itself honest: with
   zero entries the engine's tracker loop never runs, so nothing is ever
   pushed, faked, or silently dropped. The honest-skip mechanism
   (`credentials_absent` / `tracker_client_absent`) was verified live (see
   above) so the moment a real tracker is provisioned, adding its entry to
   `.helix/reporting.yaml` per `reporting.example.yaml`'s documented
   schema (`name` / `command` / `required_env` / `env_passthrough`) is all
   that is required.
4. **README reachability (§11.4.212).** This guide is not yet linked from
   `README.md`'s doc-link section — out of scope for this pass (README.md
   was not in this task's allowed write-path list; another in-flight
   session stream owns README doc-link maintenance per
   `docs/CONTINUATION.md`). Tracked as a follow-up so this document is not
   left an orphan.

## Cross-references

- `constitution/Constitution.md` §11.4.202 (reporting directives),
  §11.4.93 / §11.4.95 (SQLite SSoT), §11.4.140 (grammar), §11.4.148 /
  §11.4.171 (comprehensive description), §11.4.28 / §11.4.177
  (decoupling), §11.4.151 (prefix resolution), §11.4.213 (`FEATURE`
  scheduling).
- `constitution/docs/scripts/report_item.md` — the engine's own companion
  doc (prerequisites, usage, edge-case table).
- `docs/requests/history.md` / `docs/requests/feature_queue.md` — the
  §11.4.208 / §11.4.213 request ledgers this project already maintains.
