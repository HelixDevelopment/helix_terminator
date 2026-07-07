# Health Service

HelixTerminator microservice — Aggregates /health/live + /health/ready from all services, unified health dashboard, SLO error-budget calc, alert routing

## Features
- Aggregated health status from all 25 services
- SLO tracking and error-budget calculation
- Unified health dashboard data
- Alert routing and aggregation
- Cached health with stale aggregation

## Module Path

`helixterminator.io/services/health`

## Database

none (aggregates other services; no dedicated database)

## Upstream Dependencies

all services (health-check fan-out)

## API Endpoints

- `GET` `/api/v1/health/aggregate` — Aggregated health status
- `GET` `/api/v1/health/services/{serviceName}` — Per-service health
- `GET` `/api/v1/health/slo` — SLO status and error budget
- `GET` `/api/v1/health/dashboard` — Health dashboard data
- `GET` `/api/v1/health/alerts` — Active alerts

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/health_service
export PORT=8080
go run ./cmd/health
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

*HelixTerminator Health Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
