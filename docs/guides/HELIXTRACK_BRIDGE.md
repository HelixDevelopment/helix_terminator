# HelixTrack Bridge Service — real HelixTrack Core wiring

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## Status

`services/helixtrack-bridge-service` is wired to authenticate against a
**real, self-hosted HelixTrack/Core** instance (`submodules/helixtrack-core`)
using Core's actual `POST /do {action:"authenticate"}` contract — **not**
OAuth2 (an earlier spec draft was wrong; this document cites the verified
Core source). The client (`internal/coreclient`), the honest-503 gate
(`internal/handler.CreateBridge`), and both the unit tests and the
`TestLive_*` real-integration tests already existed in the tree before this
round (see `git log -- services/helixtrack-bridge-service/internal/coreclient`,
commits `2ef5533`, `9fb3964`, `5690435`, `929b60a`). What this round adds is
the missing piece: **actually booting a self-hosted Core sandbox and running
the live tests against it**, so the "real-integration" claim has captured
positive evidence instead of remaining permanently `SKIP`.

## 1. The real Core contract (verified against source, not assumed)

Core is a Go/Gin service. All operations — including authentication — go
through one unified endpoint. Source citations (all paths relative to
`submodules/helixtrack-core/`):

- Route registration: `POST /do` is a **public** route (no JWT
  pre-required at the router layer — auth is decided per-action) —
  `Application/internal/server/server.go:213`.
- The router parses the body into `models.Request` and only demands a
  JWT when `req.IsAuthenticationRequired()` returns true —
  `Application/internal/server/server.go:222-282`. `IsAuthenticationRequired`
  explicitly excludes `authenticate` (and `version`/`jwtCapable`/
  `dbCapable`/`health`) from the JWT requirement —
  `Application/internal/models/request.go:12-20`.
- `models.Request` shape (the actual wire contract) —
  `Application/internal/models/request.go:4-10`:
  ```go
  type Request struct {
      Action string                 `json:"action" binding:"required"`
      JWT    string                 `json:"jwt"`
      Locale string                 `json:"locale"`
      Object string                 `json:"object"`
      Data   map[string]interface{} `json:"data"`
  }
  ```
- Action routing dispatches `authenticate` to `handleAuthenticate` —
  `Application/internal/handlers/handler.go:157-158`.
- `handleAuthenticate` (`Application/internal/handlers/handler.go:1155-1267`):
  requires `data.username` + `data.password` (400 `ErrorCodeMissingData` if
  missing, lines 1156-1174); if the external Authentication microservice is
  configured+enabled it delegates there (lines 1176-1196, **and that path
  currently returns no `token` field** — a real gap in Core, out of this
  service's scope to fix); otherwise it **falls back to local
  username/password auth against Core's own `users` SQLite/Postgres table**
  (bcrypt-hashed, lines 1198-1244) and, on success, generates and returns a
  real HS256 JWT (lines 1245-1266):
  ```go
  response := models.NewSuccessResponse(map[string]interface{}{
      "token":    token,
      "username": user.Username,
      "email":    user.Email,
      "name":     user.Name,
      "role":     user.Role,
  })
  ```
- The default shipped config (`Application/Configurations/default.json`,
  the config `Dockerfile` builds and runs by default) has
  `services.authentication.enabled: false` — so a stock Core sandbox always
  takes the **local fallback** path above and always returns a real `token`.
- Unified response envelope for every `/do` call —
  `Application/internal/handlers/handler.go` via `models.NewSuccessResponse` /
  `models.NewErrorResponse`: `{"errorCode": -1, "data": {...}}` on success,
  `{"errorCode": <n>, "errorMessage": "..."}` on failure (`-1` is Core's
  explicit "no error" sentinel).
- **Seeded sandbox credentials.** Every Core server start calls
  `handlers.InitializeUserTable` (`Application/internal/server/server.go:153`),
  which — beyond creating the `users` table — seeds four bcrypt-hashed test
  users if they don't already exist
  (`Application/internal/handlers/auth_handler.go:240-277`):

  | username | password | role |
  |---|---|---|
  | `admin_user` | `Admin@123456` | user |
  | `viewer` | `Viewer@123456` | user |
  | `project_manager` | `PM@123456` | user |
  | `developer` | `Dev@123456` | user |

  This means a freshly-booted, unmodified Core sandbox is **immediately**
  authenticatable — no manual DB seeding, no `/api/auth/register` call
  required. `internal/coreclient`'s live test already defaults to
  `admin_user` / `Admin@123456` for exactly this reason.

So the bridge's `internal/coreclient.Client.Authenticate` (see
`services/helixtrack-bridge-service/internal/coreclient/coreclient.go:113-168`)
is verified correct against the real Core source: it `POST`s
`{baseURL}/do` with `{"action":"authenticate","data":{"username":...,
"password":...}}`, treats HTTP non-200 or `errorCode != -1` as failure,
and extracts `data.token`.

## 2. Can Core boot locally via rootless podman? — YES (verified)

Core ships `Application/Dockerfile`: a two-stage `golang:1.24-alpine`
build (CGO-enabled, for the SQLite driver) producing a static `htCore`
binary, defaulting to `Configurations/default.json` (SQLite,
`services.authentication.enabled: false`, port 8080). It requires **no
external services** to boot and answer `/do {authenticate}` — the DB
auto-initializes on first start (`Application/main.go:90-104` runs the
DDL migrator against `Database/DDL` when no `Definition.sqlite` file
exists yet), and the user table + seed users are created immediately
after (`Application/internal/server/server.go:153`).

### Sandbox stand-up recipe (rootless podman, plain default userns)

```bash
cd submodules/helixtrack-core/Application

# Build (no host dependencies beyond podman + network access to pull
# golang:1.24-alpine and go module proxies)
podman build -t localhost/helixtrack-core-sandbox:local -f Dockerfile .

# Run — PLAIN rootless defaults only: no :z, no --userns=keep-id, no
# --security-opt label=disable (those crash this host's SELinux config
# per the task brief; podman's default userns is sufficient here).
podman run -d --name helixtrack-core-sandbox \
  -p 18180:8080 \
  localhost/helixtrack-core-sandbox:local

# Wait for health, then verify the real /do contract:
curl -s http://127.0.0.1:18180/health
# {"status":"ok"}

curl -s -X POST http://127.0.0.1:18180/do \
  -H 'Content-Type: application/json' \
  -d '{"action":"authenticate","data":{"username":"admin_user","password":"Admin@123456"}}'
# {"errorCode":-1,"data":{"email":"admin@test.com","name":"Admin User",
#   "role":"user","token":"eyJhbGci...","username":"admin_user"}}
```

Captured evidence for this exact run (2026-07-22): container `HostConfig`
inspected post-boot showed `UsernsMode=""` and `SecurityOpt=[]` — i.e.
podman's own rootless default, no manual namespace/SELinux overrides —
confirming the "plain default userns" constraint was honoured. Boot log
+ inspect output are process-local qa evidence for this run (not
committed — ephemeral container state per §11.4.30, no versioned build
artifacts).

No image registry push is required; the sandbox is disposable
(`podman rm -f helixtrack-core-sandbox`) and rebuilds deterministically
from the vendored `submodules/helixtrack-core` source.

## 3. Wiring the bridge to the sandbox

```bash
export HELIXTRACK_CORE_BASE_URL=http://127.0.0.1:18180
export HELIXTRACK_CORE_USERNAME=admin_user
export HELIXTRACK_CORE_PASSWORD='Admin@123456'
```

`cmd/helixtrack-bridge-service/main.go` only constructs a real
`coreclient.Client` (and only then can `CreateBridge` ever mark a bridge
`active`) when `HELIXTRACK_CORE_BASE_URL` is non-empty:

```go
if coreBaseURL := os.Getenv("HELIXTRACK_CORE_BASE_URL"); coreBaseURL != "" {
    core = coreclient.New(coreBaseURL, os.Getenv("HELIXTRACK_CORE_USERNAME"), os.Getenv("HELIXTRACK_CORE_PASSWORD"))
}
```
(`services/helixtrack-bridge-service/cmd/helixtrack-bridge-service/main.go:67-68`)

If it is unset, `core` stays `nil`, and `Handler.authenticateCore`
(`internal/handler/handler.go:42-47`) fails closed with
`"helixtrack core client not configured"` — `CreateBridge` then always
responds `503 {"status":"error", ...}` and **never writes a bridge
record**. This honest-failure path is unconditional and was re-verified
unchanged in this round (source read, `internal/handler/handler.go:50-87`).

## 4. Running the real-integration tests

```bash
cd services/helixtrack-bridge-service
export GOWORK=off GOMAXPROCS=2
export HELIXTRACK_CORE_BASE_URL=http://127.0.0.1:18180
export HELIXTRACK_CORE_USERNAME=admin_user
export HELIXTRACK_CORE_PASSWORD='Admin@123456'

go test -p 2 -v -run TestLive_Authenticate_RealCore ./internal/coreclient/...
go test -p 2 -v -run TestLive_CreateBridge_ReflectsRealCoreAuth ./internal/handler/...

# Full suite (unit + live, race-checked):
go test -p 2 -race -cover ./...
```

### Captured evidence — 2026-07-22 run against a genuinely running sandbox

```
=== RUN   TestLive_Authenticate_RealCore
=== RUN   TestLive_Authenticate_RealCore/correct_credentials_return_a_real_HS256_24h_JWT
=== RUN   TestLive_Authenticate_RealCore/wrong_password_yields_a_non-active_auth_error,_no_token_cached
--- PASS: TestLive_Authenticate_RealCore (0.14s)
    --- PASS: TestLive_Authenticate_RealCore/correct_credentials_return_a_real_HS256_24h_JWT (0.05s)
    --- PASS: TestLive_Authenticate_RealCore/wrong_password_yields_a_non-active_auth_error,_no_token_cached (0.09s)
PASS
ok  	github.com/helixdevelopment/helixtrack-bridge-service/internal/coreclient	0.144s

=== RUN   TestLive_CreateBridge_ReflectsRealCoreAuth
=== RUN   TestLive_CreateBridge_ReflectsRealCoreAuth/real_correct_Core_credentials_pass_the_auth_gate_(reach_the_DB_layer)
=== RUN   TestLive_CreateBridge_ReflectsRealCoreAuth/real_wrong_Core_password_yields_a_non-active/error_status,_never_reaches_the_DB_layer
--- PASS: TestLive_CreateBridge_ReflectsRealCoreAuth (0.10s)
    --- PASS: TestLive_CreateBridge_ReflectsRealCoreAuth/real_correct_Core_credentials_pass_the_auth_gate_(reach_the_DB_layer) (0.05s)
    --- PASS: TestLive_CreateBridge_ReflectsRealCoreAuth/real_wrong_Core_password_yields_a_non-active/error_status,_never_reaches_the_DB_layer (0.05s)
PASS
ok  	github.com/helixdevelopment/helixtrack-bridge-service/internal/handler	0.105s
```

Full suite (`go test -p 2 -race -cover ./...`, same environment, same
run) was green across every package
(`cmd/helixtrack-bridge-service`, `internal/coreclient` 79.1% cov,
`internal/handler` 37.2% cov, `internal/model`, `internal/repository`
4.5% cov — no live Postgres in this environment so DB-layer coverage is
necessarily thin, see §5 — `internal/server` 57.6% cov, `migrations`
35.8% cov), no race detector findings, `exit 0`.

`TestLive_Authenticate_RealCore` performs a structural proof, not a
shape-only check: it decodes the JWT header and asserts `alg=HS256`,
`typ=JWT`, decodes the payload and asserts `username` matches the
authenticated user and `exp - iat == 86400` (Core's documented 24h
window) — this is the real token Core issued, parsed and inspected, not
a fixture.

### With no sandbox running (honest SKIP, not a fabricated PASS)

```bash
unset HELIXTRACK_CORE_BASE_URL HELIXTRACK_CORE_USERNAME HELIXTRACK_CORE_PASSWORD
go test -v -run TestLive ./...
```
yields (§11.4.3 topology dispatch):
```
--- SKIP: TestLive_Authenticate_RealCore
    coreclient_live_test.go:26: SKIP (§11.4.3): HELIXTRACK_CORE_BASE_URL not set — live HelixTrack Core sandbox not running in this environment
--- SKIP: TestLive_CreateBridge_ReflectsRealCoreAuth
    handler_live_test.go:35: SKIP (§11.4.3): HELIXTRACK_CORE_BASE_URL not set — live HelixTrack Core sandbox not running in this environment
```

## 5. Honest boundaries (§11.4.6 — no guessing)

- **No live Postgres in this environment.** The bridge's own
  `repository`/`migrations` packages talk to Postgres
  (`DATABASE_URL`), which was out of scope for this round (the task
  brief scoped this round to the HelixTrack Core wiring only). The
  `TestLive_CreateBridge_ReflectsRealCoreAuth` test therefore proves the
  auth gate is genuinely driven by the real Core (correct creds reach —
  and only fail at — the DB layer with `500`; wrong creds never reach
  the DB layer and get an honest `503`) rather than proving the full
  create-and-persist round trip. A live-Postgres-backed end-to-end test
  (bridge → real Core auth → real Postgres row) is a natural next PWU,
  not fabricated here.
- **The external-Authentication-service delegation path in Core**
  (`Application/internal/handlers/handler.go:1176-1196`) returns no
  `token` field in its success response — a real gap in Core itself.
  This does not affect the bridge's sandbox recipe above because the
  sandbox's default config (`services.authentication.enabled: false`)
  never takes that path; documented here so it is not silently assumed
  fixed.
- The in-memory `tokenmanager.Storage` adapter in `internal/coreclient`
  is explicitly process-local (does not survive a bridge-service
  restart) — by design, documented in the package doc comment
  (`internal/coreclient/coreclient.go:1-7`, `:39-51`).

## Sources verified 2026-07-22

- `submodules/helixtrack-core` (vendored submodule, HEAD `6edbb5e`,
  `heads/main`) — read directly, not from memory:
  `Application/internal/server/server.go`,
  `Application/internal/models/request.go`,
  `Application/internal/handlers/handler.go`,
  `Application/internal/handlers/auth_handler.go`,
  `Application/internal/services/auth_service.go`,
  `Application/main.go`, `Application/Dockerfile`,
  `Application/Configurations/default.json`,
  `Application/docker-compose.yml`.
- `services/helixtrack-bridge-service` (this repo) —
  `internal/coreclient/coreclient.go`,
  `internal/coreclient/coreclient_live_test.go`,
  `internal/handler/handler.go`,
  `internal/handler/handler_live_test.go`,
  `cmd/helixtrack-bridge-service/main.go`, `README.md`.
- Live evidence: this round's own `podman build` / `podman run` /
  `curl` / `go test` transcripts against a genuinely booted sandbox
  (2026-07-22, container `helixtrack-core-sandbox`, image
  `localhost/helixtrack-core-sandbox:local`), reproduced verbatim in §2
  and §4 above.
