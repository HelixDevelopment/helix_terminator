# Host Service

HelixTerminator microservice — SSH host/group CRUD, health ping, import/export, bastion/jump-host chains, host templates

## Features
- SSH host CRUD with connection parameters
- Host groups and tags for organization
- Health ping and connectivity testing
- Import/export (CSV, JSON, SSH config)
- Bastion/jump-host chain configuration
- Host templates for quick provisioning

## Module Path

`helixterminator.io/services/host`

## Database

PostgreSQL helixterm_hosts

## Upstream Dependencies

vault, org, audit

## API Endpoints

- `GET` `/api/v1/hosts` — List hosts
- `POST` `/api/v1/hosts` — Create host
- `GET` `/api/v1/hosts/{hostId}` — Get host
- `PUT` `/api/v1/hosts/{hostId}` — Update host
- `DELETE` `/api/v1/hosts/{hostId}` — Delete host
- `POST` `/api/v1/hosts/{hostId}/connect` — Initiate connection
- `POST` `/api/v1/hosts/{hostId}/disconnect` — Disconnect
- `GET` `/api/v1/hosts/{hostId}/connections` — Connection history
- `GET` `/api/v1/hosts/{hostId}/health` — Health check
- `POST` `/api/v1/hosts/import` — Bulk import
- `GET` `/api/v1/host-groups` — List groups
- `POST` `/api/v1/host-groups` — Create group
- `PUT` `/api/v1/host-groups/{groupId}` — Update group
- `DELETE` `/api/v1/host-groups/{groupId}` — Delete group

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/host_service
export PORT=8080
go run ./cmd/host
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

*HelixTerminator Host Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
