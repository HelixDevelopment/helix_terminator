# Config Service

HelixTerminator microservice — Centralized feature flags + operational parameters, per-org overrides, runtime config propagation via Kafka, config audit

## Features
- Centralized feature flag management
- Per-organization configuration overrides
- Runtime config propagation via Kafka
- Configuration audit trail
- Operational parameter management

## Module Path

`helixterminator.io/services/config`

## Database

PostgreSQL helixterm_config

## Upstream Dependencies

audit

## API Endpoints

- `GET` `/api/v1/config/flags` — List feature flags
- `GET` `/api/v1/config/flags/{flagName}` — Get feature flag
- `POST` `/api/v1/config/flags` — Create feature flag
- `PUT` `/api/v1/config/flags/{flagName}` — Update feature flag
- `DELETE` `/api/v1/config/flags/{flagName}` — Delete feature flag
- `GET` `/api/v1/config/settings` — List settings
- `PUT` `/api/v1/config/settings` — Update settings
- `GET` `/api/v1/config/audit` — Get config audit log

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/config_service
export PORT=8080
go run ./cmd/config
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

*HelixTerminator Config Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
