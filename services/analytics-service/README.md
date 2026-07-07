# Analytics Service

HelixTerminator microservice — Session/command/transfer usage aggregation, dashboard data, SLO tracking, Grafana/Prometheus export

## Features
- Usage aggregation across all services
- Dashboard data generation
- SLO tracking and error-budget calculation
- Grafana/Prometheus metric export
- Time-series data with weekly partitioning

## Module Path

`helixterminator.io/services/analytics`

## Database

PostgreSQL helixterm_analytics (time-series, partitioned by week)

## Upstream Dependencies

Kafka consumer only (no synchronous upstream)

## API Endpoints

- `GET` `/api/v1/analytics/metrics` — Get usage metrics
- `GET` `/api/v1/analytics/dashboard` — Get dashboard data
- `GET` `/api/v1/analytics/slo` — Get SLO status
- `GET` `/api/v1/analytics/export` — Export analytics data

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/analytics_service
export PORT=8080
go run ./cmd/analytics
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

*HelixTerminator Analytics Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
