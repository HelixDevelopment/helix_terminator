# HEL-002 Evidence Summary (T009)

**Date**: 2026-07-23 (host clock logs 2026-07-22T21:0xZ UTC) · **Worktree**:
`.claude/worktrees/hel-002-provingground` · **Branch**:
`feat/hel-002-speckit-provingground`

## Success-criteria → evidence map

| SC | Claim | Evidence | Result |
|----|-------|----------|--------|
| SC-001 | GOWORK=off + workspace matrices all exit 0 | `final_matrix.log` + `us2_run/{ws_*,off_*}.log` (8× rc=0); RED pair: `red_baseline3.log` (off matrix 4× rc=1 pre-fix, ws 4× rc=0) | PASS (RED→GREEN observed) |
| SC-002 | recorded == built, both modes | `us2_run/drift.log`: recorded=v1.12.0 workspace=v1.12.0 gowork_off=v1.12.0; RED: `red_run3/drift.log` recorded=v1.10.0 vs workspace=v1.12.0 | PASS |
| SC-003 | tidy-stable | `us2_run/tidy_stable.log` (empty diff, rc=0); RED: `red_run3/tidy_stable.log` (292-line diff, rc=1) | PASS |
| SC-004 | container context reaches 100% of replace targets | `us2_run/ctx_reach.log` (9/9 OK + root-context Dockerfile checks); RED: `red_run3/ctx_reach.log` | PASS |
| SC-005 | captured evidence, real exit codes | every step logs its own rc; harness exit = mandatory-failure count | PASS |

## Beyond-mandatory result

REAL container image built (no skip needed): podman (docker shim)
`localhost/hel002-notification-service:proof`, 30.7 MB, digest
`53aef00a4b63…` — `us2_run/docker_build.log`.

## Changed files (scope-audited)

- `services/notification-service/go.mod` — tidied module-mode-correct;
  gin v1.10.0→v1.12.0 recorded (was already BUILT at v1.12.0 in workspace);
  comment block updated (supersedes "deliberately not tidied"; preserves
  slack-go non-replacement rationale verbatim)
- `services/notification-service/go.sum` — herald transitive checksums
- `services/notification-service/Dockerfile` — repo-root context, module-mode
  build, golang:1.26-alpine (old 1.22 base predated even the old go directive)

## Honest OPEN items (HEL-002 stays non-terminal, §11.4.146(D3))

- Siblings NOT fixed (out of implementation scope, recorded in spec):
  helixtrack-bridge-service, container-bridge-service, ai-service share the
  local-replace pattern; their Dockerfiles/manifests unaudited here.
- docker-compose wiring referenced by the SSoT item does not exist in-repo
  (measured, depth ≤3); compose-side proof impossible until it exists.
- Integration-tagged tests COMPILE-verified only (`go vet -tags integration`),
  not run (live infra required).

## Environment repairs made in the worktree (NOT committed; fleet finding F3)

- `go.work`+`go.work.sum` copied from the main checkout (both gitignored —
  a bare worktree inherits the MAIN checkout's workspace via Go's upward
  walk: wrong-module-set errors).
- Submodules initialized in-worktree: herald (+16 nested), auth, containers,
  llmprovider.
