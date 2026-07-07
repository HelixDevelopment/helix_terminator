# AI Service

HelixTerminator microservice — Command autocomplete, output explanation, anomaly detection, runbook generation, incident assist

## Features
- AI-powered command autocomplete for terminal sessions
- Output explanation and summarization
- Anomaly detection for session behavior
- Runbook generation from incident patterns
- Incident assist with contextual recommendations

## Module Path

`helixterminator.io/services/ai`

## Database

PostgreSQL helixterm_ai (+ Redis suggestion cache)

## Upstream Dependencies

terminal, user, audit

## API Endpoints

- `POST` `/api/v1/ai/complete` — Command completion
- `POST` `/api/v1/ai/explain` — Explain command output
- `POST` `/api/v1/ai/anomaly` — Anomaly detection
- `POST` `/api/v1/ai/runbook` — Generate runbook
- `POST` `/api/v1/ai/incident` — Incident assist
- `GET` `/api/v1/ai/models` — List available models
- `GET` `/api/v1/ai/feedback` — Get feedback history
- `POST` `/api/v1/ai/feedback` — Submit feedback

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/ai_service
export PORT=8080
go run ./cmd/ai
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

*HelixTerminator AI Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
