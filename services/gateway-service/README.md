# Gateway Service

HelixTerminator microservice — Single ingress for all client traffic; JWT validation; per-user/IP rate limiting; upstream routing; circuit breaking; WebSocket upgrade proxy

## Features
- Single ingress point for all client traffic
- JWT validation with EdDSA (Ed25519)
- Per-user and per-IP rate limiting
- Upstream routing with health checks
- Circuit breaker pattern
- WebSocket upgrade proxy for terminal
- OpenAPI 3.1 spec serving

## Module Path

`helixterminator.io/services/gateway`

## Database

none (stateless; Redis used for rate-limit + JWKS cache)

## Upstream Dependencies

all 25 downstream services (ingress)

## API Endpoints

- `GET` `/healthz` — Health check
- `GET` `/healthz/ready` — Readiness check
- `GET` `/api/v1/openapi.json` — OpenAPI 3.1 spec (JSON)
- `GET` `/api/v1/openapi.yaml` — OpenAPI 3.1 spec (YAML)
- `GET` `/api/v1/docs` — Swagger/ReDoc UI
- `GET` `/metrics` — Prometheus metrics

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/gateway_service
export PORT=8080
go run ./cmd/gateway
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

*HelixTerminator Gateway Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
