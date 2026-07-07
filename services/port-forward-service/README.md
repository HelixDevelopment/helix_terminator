# Port Forwarding Service

HelixTerminator microservice — Port-forward rule catalog + lifecycle (local/remote/dynamic/reverse), auto-reconnect, tunnel metrics

## Features
- Port-forward rule catalog (local, remote, dynamic, reverse)
- Auto-reconnect with exponential backoff
- Tunnel metrics and monitoring
- Lifecycle management (create, start, stop, delete)
- Integration with SSH Proxy for tunnel establishment

## Module Path

`helixterminator.io/services/port-forward`

## Database

PostgreSQL helixterm_port_forward

## Upstream Dependencies

ssh-proxy, vault, audit

## API Endpoints

- `GET` `/api/v1/port-forwards` — List rules
- `POST` `/api/v1/port-forwards` — Create rule
- `GET` `/api/v1/port-forwards/{ruleId}` — Get rule
- `PUT` `/api/v1/port-forwards/{ruleId}` — Update rule
- `DELETE` `/api/v1/port-forwards/{ruleId}` — Delete rule
- `POST` `/api/v1/port-forwards/{ruleId}/start` — Start tunnel
- `POST` `/api/v1/port-forwards/{ruleId}/stop` — Stop tunnel
- `GET` `/api/v1/port-forwards/{ruleId}/metrics` — Tunnel metrics

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/port-forward_service
export PORT=8080
go run ./cmd/port-forward
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

*HelixTerminator Port Forwarding Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
