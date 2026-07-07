# Workspace Service

HelixTerminator microservice — Named workspace CRUD (hosts + snippets + vault items + settings), templates, sharing, quick-launch

## Features
- Named workspace CRUD
- Session layout management
- Workspace templates for quick setup
- Workspace sharing with permissions
- Quick-launch from workspace
- Integration with hosts, snippets, and vault

## Module Path

`helixterminator.io/services/workspace`

## Database

PostgreSQL helixterm_workspaces

## Upstream Dependencies

user, org, audit

## API Endpoints

- `GET` `/api/v1/workspaces` — List workspaces
- `POST` `/api/v1/workspaces` — Create workspace
- `GET` `/api/v1/workspaces/{workspaceId}` — Get workspace
- `PUT` `/api/v1/workspaces/{workspaceId}` — Update workspace
- `DELETE` `/api/v1/workspaces/{workspaceId}` — Delete workspace
- `POST` `/api/v1/workspaces/{workspaceId}/restore` — Restore snapshot
- `GET` `/api/v1/workspace-templates` — List templates
- `POST` `/api/v1/workspace-templates` — Create template
- `GET` `/api/v1/workspaces/{workspaceId}/sessions` — Sessions in workspace
- `POST` `/api/v1/workspaces/{workspaceId}/share` — Share workspace

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/workspace_service
export PORT=8080
go run ./cmd/workspace
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

*HelixTerminator Workspace Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
