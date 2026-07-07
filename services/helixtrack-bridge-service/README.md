# HelixTrack Bridge Service

HelixTerminator microservice — real JWT-authenticated link to a HelixTrack
Core instance (`submodules/helixtrack-core`); ties terminal sessions to
HelixTrack issues/sprints; deployment-event sync

## Features
- Real JWT authentication against HelixTrack Core's unified `/do`
  endpoint (`{"action":"authenticate"}`) — **not** OAuth2. A bridge is
  only ever marked `active` after a genuine authenticate call succeeds
  against a running Core instance; a rejected or unreachable Core yields
  `503 {"status":"error", ...}` and no bridge record is written.
- Terminal session to issue linking
- Sprint data synchronization
- Deployment event sync
- Issue tracking integration

## HelixTrack Core authentication

`internal/coreclient` (backed by the owned-org
`digital.vasic.auth/pkg/tokenmanager` library, wired via `go.mod`'s
`replace digital.vasic.auth => ../../submodules/auth`) authenticates as
follows on every `POST /api/v1/helixtrack-bridges`:

1. POST `{HELIXTRACK_CORE_BASE_URL}/do` with
   `{"action":"authenticate","data":{"username":"...","password":"..."}}`.
2. HelixTrack Core's unified response envelope
   (`{"errorCode":-1,"data":{"token":"<JWT HS256, 24h>",...}}`) is parsed;
   any HTTP-non-200 or `errorCode != -1` is treated as an authentication
   failure.
3. The returned JWT is cached (24h TTL, matching Core's known expiry) via
   `tokenmanager.Manager.StoreTokenInfo`; subsequent requests reuse the
   cached token (`HasValidToken`) and only re-authenticate once it has
   expired.

No production `tokenmanager.Storage` implementation ships in
`digital.vasic.auth` — `internal/coreclient` provides its own minimal
in-memory adapter (process-local; does not survive a restart), mirroring
the `memoryStorage` test template in
`submodules/auth/pkg/tokenmanager/tokenmanager_test.go`.

## Module Path

`helixterminator.io/services/helixtrack-bridge`

## Database

PostgreSQL helixterm_helixtrack_bridge

## Upstream Dependencies

user, org, audit

## API Endpoints

- `POST` `/api/v1/helixtrack/link` — Link session to issue
- `GET` `/api/v1/helixtrack/links` — List session-issue links
- `POST` `/api/v1/helixtrack/sync` — Sync sprint data
- `GET` `/api/v1/helixtrack/issues` — Get linked issues
- `POST` `/api/v1/helixtrack/deployments` — Sync deployment event

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/helixtrack-bridge_service
export PORT=8080
go run ./cmd/helixtrack-bridge
```

## Testing

```bash
go test -v -race -cover ./...
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PORT` | No | 8080 | HTTP/gRPC port |
| `LOG_LEVEL` | No | info | Log level (debug/info/warn/error) |
| `KAFKA_BROKERS` | No | — | Kafka bootstrap servers |
| `REDIS_URL` | No | — | Redis connection string |
| `HELIXTRACK_CORE_BASE_URL` | Yes (for `CreateBridge` to ever mark a bridge active) | — | Base URL of the HelixTrack Core instance, e.g. `http://127.0.0.1:8080` (no trailing slash). Unset ⇒ `CreateBridge` fails closed with `503`. |
| `HELIXTRACK_CORE_USERNAME` | Yes (with the above) | — | Username to authenticate against Core's `/do {action:authenticate}`. |
| `HELIXTRACK_CORE_PASSWORD` | Yes (with the above) | — | Password for the above username. Never log or commit this value (§11.4.10). |

---

*HelixTerminator HelixTrack Bridge Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
