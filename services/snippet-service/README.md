# Snippet Service

HelixTerminator microservice — Command/script/SQL snippet CRUD, folders/namespaces, parameterization, execution history, versioning

## Features
- Command/script/SQL snippet CRUD
- Folders and namespaces for organization
- Variable parameterization
- Execution history and versioning
- Full-text search across snippets

## Module Path

`helixterminator.io/services/snippet`

## Database

PostgreSQL helixterm_snippets

## Upstream Dependencies

audit

## API Endpoints

- `GET` `/api/v1/snippets` — List snippets
- `POST` `/api/v1/snippets` — Create snippet
- `GET` `/api/v1/snippets/{snippetId}` — Get snippet
- `PUT` `/api/v1/snippets/{snippetId}` — Update snippet
- `DELETE` `/api/v1/snippets/{snippetId}` — Delete snippet
- `POST` `/api/v1/snippets/{snippetId}/execute` — Execute snippet
- `GET` `/api/v1/snippets/{snippetId}/history` — Execution history
- `GET` `/api/v1/snippet-categories` — List categories
- `POST` `/api/v1/snippet-categories` — Create category
- `GET` `/api/v1/snippets/search` — Full-text search

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/snippet_service
export PORT=8080
go run ./cmd/snippet
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

*HelixTerminator Snippet Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
