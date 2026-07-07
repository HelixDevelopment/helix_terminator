# Collaboration Service

HelixTerminator microservice — Real-time session sharing (observer/co-pilot/owner roles), CRDT buffer sync, broadcast mode, chat sidebar

## Features
- Real-time session sharing with role-based access
- CRDT buffer synchronization
- Broadcast mode for presentations
- Chat sidebar for collaboration
- Observer, co-pilot, and owner roles

## Module Path

`helixterminator.io/services/collab`

## Database

PostgreSQL helixterm_collab (+ Redis pub/sub)

## Upstream Dependencies

terminal, user, org, notification

## API Endpoints

- `POST` `/api/v1/collab/sessions/{sessionId}/join` — Join shared session
- `POST` `/api/v1/collab/sessions/{sessionId}/leave` — Leave shared session
- `GET` `/api/v1/collab/sessions/{sessionId}/participants` — List participants
- `POST` `/api/v1/collab/sessions/{sessionId}/broadcast` — Broadcast event
- `GET` `/api/v1/collab/sessions/{sessionId}/buffer` — Get CRDT buffer
- `POST` `/api/v1/collab/sessions/{sessionId}/chat` — Send chat message
- `GET` `/api/v1/collab/sessions/{sessionId}/chat` — Get chat history

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/collaboration_service
export PORT=8080
go run ./cmd/collaboration
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

*HelixTerminator Collaboration Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
