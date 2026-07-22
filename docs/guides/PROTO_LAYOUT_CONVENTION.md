# Proto Layout Convention — `services/*/api/proto`

**Revision:** 2
**Last modified:** 2026-07-22T00:00:00Z

## Scope

This convention governs the **25 `.proto` files** under
`services/*/api/proto/*.proto` — the live, in-scope proto surface
referenced by `go.work` (`docs/research/**` proto trees are historical
design drafts, out of scope; see the S-T3 remediation plan for the full
locate-and-classify analysis).

## Why

`buf lint` (STANDARD ruleset, `services/buf.yaml`) flagged all 25 files
with a uniform 5-finding pattern: `FILE_LOWER_SNAKE_CASE`,
`PACKAGE_DIRECTORY_MATCH`, `PACKAGE_VERSION_SUFFIX`,
`RPC_REQUEST_STANDARD_NAME`, `RPC_RESPONSE_STANDARD_NAME` — 125 findings
total. Every service's fix is file-scope-disjoint and mechanically
identical, so this document records **one canonical target shape** all
25 services converge on, rather than 25 improvised variants.

## Canonical target layout

```
services/<name>/api/proto/<name_snake>/v1/<name_snake>.proto
```

```protobuf
syntax = "proto3";

package <name_snake>.v1;

option go_package = "github.com/helixdevelopment/<name>/api/proto/<name_snake>/v1;<name_snake>v1";

// TODO: define service RPCs and messages

service <PascalName>Service {
	rpc HealthCheck (HealthCheckRequest) returns (HealthCheckResponse);
}

message HealthCheckRequest {}

message HealthCheckResponse {
	bool healthy = 1;
}
```

Where `<name_snake>` is the service's directory name with hyphens
converted to underscores (e.g. `ai-service` → `ai_service`,
`ssh-proxy-service` → `ssh_proxy_service`).

This mirrors the shape already used by the one pre-existing clean tree
in the repo (`docs/research/mvp/final/implementation/api/proto/helixterm/
<service>/v1/<service>.proto`), adapted to a per-service top-level
package (`<name_snake>.v1`) instead of one shared `helixterm` umbrella
package, since each `services/<name>/` tree is independently versioned
and independently buildable (`go.work` module per service).

### What each rename fixes

| Change | Rule(s) cleared |
|---|---|
| Move file into `<name_snake>/v1/` subdirectory | `FILE_LOWER_SNAKE_CASE` + `PACKAGE_DIRECTORY_MATCH` |
| `package <name_snake>.v1;` | `PACKAGE_VERSION_SUFFIX` |
| `HealthRequest` → `HealthCheckRequest` | `RPC_REQUEST_STANDARD_NAME` |
| `HealthResponse` → `HealthCheckResponse` | `RPC_RESPONSE_STANDARD_NAME` |

## Verification

Per service, after applying the rename:

```bash
buf lint services --path services/<name>/api/proto
```

must report **zero** findings for that service's files.

Workspace-wide, after all services are migrated:

```bash
make proto-lint
```

must report **zero** findings across all 25 services.

## Regeneration — honest gap

No `protoc-gen-go` / `protoc-gen-go-grpc` plugin is installed on any
host that has exercised this workspace's tooling to date. `make
proto-generate` (via `services/buf.gen.yaml`) is provided so the
regeneration mechanism exists and is documented, but running it is
expected to fail with a "plugin not found" error until those plugins
are installed — an honest §11.4.3 / §11.4.69 `tool_absent` SKIP, tracked
as its own separate follow-up item, not silently claimed as done. This
is safe to defer because **zero generated code (`*.pb.go`) has ever
existed in this repo**, and no Go service or the Flutter client
currently imports any proto package (verified by grep across
`services/*/**.go` and `clients/flutter/pubspec.yaml` — the Flutter
client talks REST/JSON, not gRPC).

## Breaking-change baseline

`make proto-breaking` runs `buf breaking` against the
`helix_terminator-0.1.0` release tag, **once per service** — not as a
single workspace-wide invocation. Discovered while landing Wave 0/1
(§11.4.6 — real behaviour, not assumed): `services/buf.yaml` declares
one buf module **per service** (25 modules, each rooted at
`services/<name>/api/proto`), but at the `helix_terminator-0.1.0` tag
no `buf.yaml` existed for `services/` at all, so buf treats that
historical ref as **one implicit module**. `buf breaking` cannot
compare a 25-module workspace against a 1-module snapshot in one
invocation (`input contained 25 images, whereas against contained 1
images` — a hard structural error, not a lint finding). The Makefile
target loops per service instead, so each comparison is a genuine
1-module-vs-1-module check; this was verified to correctly report
`CLEAN` on every untouched service and `BREAKING` (file deleted) on
every service actually renamed — proving the mechanism itself works,
not merely that it runs.

This bulk rename is a **formally breaking change** per buf's `FILE`
breaking category (file moves and package/type renames are breaking by
construction) — but it is **safe**, because no `.pb.go` has ever been
generated from these files and no runtime consumer exists (see the
S-T3 remediation plan for the full zero-consumer evidence: no `.pb.go`
files anywhere in the repo, no Go import of `"proto"`/`"grpc"` in any
`services/*/` tree, no Dart/gRPC dependency in the Flutter client, no
`proto`/`generate` Makefile target prior to this change). `make
proto-breaking` is therefore **expected to exit non-zero** for any
service that has been through this rename until the full 25-service
sweep lands and the baseline is reconciled — this is the correct,
honest signal (§11.4.120 fix-breaks-its-own-gate: the gate is doing
its job, not regressing). Per §11.4.120, once the full rename wave
lands across all 25 services, the `helix_terminator-0.1.0` tag
reference should be re-pointed at the post-rename HEAD as the new
baseline going forward, so future proto changes are caught against the
*current* shape rather than the superseded pre-rename one.
