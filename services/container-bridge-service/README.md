# Container Bridge Service

HelixTerminator microservice — Kubernetes cluster registration, pod exec/shell sessions, container log streaming, Docker/Podman host registration

## Features
- Kubernetes cluster registration and management
- Pod exec/shell sessions
- Container log streaming
- Docker/Podman host registration
- Secure credential injection via Vault

## Module Path

`helixterminator.io/services/container-bridge`

## Database

PostgreSQL helixterm_container_bridge

## Upstream Dependencies

vault, org, audit

## API Endpoints

- `POST` `/api/v1/container-bridge/clusters` — Register cluster
- `GET` `/api/v1/container-bridge/clusters` — List clusters
- `GET` `/api/v1/container-bridge/clusters/{clusterId}` — Get cluster
- `DELETE` `/api/v1/container-bridge/clusters/{clusterId}` — Deregister cluster
- `POST` `/api/v1/container-bridge/clusters/{clusterId}/exec` — Execute command in pod
- `GET` `/api/v1/container-bridge/clusters/{clusterId}/logs` — Stream pod logs
- `POST` `/api/v1/container-bridge/hosts` — Register Docker/Podman host
- `GET` `/api/v1/container-bridge/hosts` — List container hosts

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/container-bridge_service
export PORT=8080
go run ./cmd/container-bridge
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

*HelixTerminator Container Bridge Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
