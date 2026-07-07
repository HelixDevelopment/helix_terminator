# Terminal Session Service

HelixTerminator microservice — WebSocket terminal I/O proxy, scrollback buffer (Redis), command-boundary detection, collaborative fan-out

## Features
- WebSocket terminal I/O proxy
- Scrollback buffer with Redis caching
- Command-boundary detection
- Terminal resize handling
- Multi-session support
- Collaborative fan-out (CRDT-based)
- Theme and font customization

## Module Path

`helixterminator.io/services/terminal`

## Database

PostgreSQL helixterm_terminal

## Upstream Dependencies

ssh-proxy, collab, recording, ai, audit

## API Endpoints

- `GET` `/api/v1/terminal/stream` — WebSocket terminal I/O
- `POST` `/api/v1/terminal/shell` — Request shell
- `GET` `/api/v1/terminal/themes` — List themes
- `POST` `/api/v1/terminal/themes/{themeId}` — Apply theme
- `GET` `/api/v1/terminal/fonts` — List fonts
- `POST` `/api/v1/terminal/fonts/{fontId}` — Set font
- `POST` `/api/v1/terminal/clipboard` — Clipboard sync
- `GET` `/api/v1/terminal/scrollback` — Get scrollback
- `POST` `/api/v1/terminal/sessions/{sessionId}/resize` — Resize terminal
- `POST` `/api/v1/terminal/sessions/{sessionId}/collaborate` — Start collaboration

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/terminal_service
export PORT=8080
go run ./cmd/terminal
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

*HelixTerminator Terminal Session Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
