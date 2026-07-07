# HelixTrack Bridge Service

HelixTerminator microservice — OAuth2 link to helixtrack.ru/core; ties terminal sessions to HelixTrack issues/sprints; deployment-event sync

## Features
- OAuth2 integration with HelixTrack
- Terminal session to issue linking
- Sprint data synchronization
- Deployment event sync
- Issue tracking integration

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

---

*HelixTerminator HelixTrack Bridge Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
