# Audit Service

HelixTerminator microservice — Append-only Merkle-chained audit log, compliance query/export API, SOC 2/ISO 27001/FedRAMP evidence, retention enforcement

## Features
- Append-only Merkle-chained audit log
- Compliance query and export API
- SOC 2 / ISO 27001 / FedRAMP evidence collection
- Retention policy enforcement
- Partitioned by organization and month

## Module Path

`helixterminator.io/services/audit`

## Database

PostgreSQL helixterm_audit (partitioned by org + month)

## Upstream Dependencies

Kafka-only leaf; consumes nothing synchronously

## API Endpoints

- `GET` `/api/v1/audit/events` — Query audit events
- `GET` `/api/v1/audit/events/{eventId}` — Get specific event
- `GET` `/api/v1/audit/export` — Export audit log
- `GET` `/api/v1/audit/compliance` — Compliance dashboard
- `GET` `/api/v1/audit/retention` — Retention policy status

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/audit_service
export PORT=8080
go run ./cmd/audit
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

*HelixTerminator Audit Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
