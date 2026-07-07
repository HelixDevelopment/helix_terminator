# SFTP Service

HelixTerminator microservice — SFTP session/file operations, transfer queue + resume, checksum verification, bidirectional directory sync

## Features
- SFTP session and file operations
- Transfer queue with resume capability
- Checksum verification (SHA-256)
- Bidirectional directory sync
- Integration with SSH Proxy for connections

## Module Path

`helixterminator.io/services/sftp`

## Database

PostgreSQL helixterm_sftp

## Upstream Dependencies

ssh-proxy, vault, audit

## API Endpoints

- `POST` `/api/v1/sftp/sessions` — Create SFTP session
- `DELETE` `/api/v1/sftp/sessions/{sessionId}` — Close session
- `GET` `/api/v1/sftp/sessions/{sessionId}/files` — List files
- `GET` `/api/v1/sftp/sessions/{sessionId}/files/{path}` — Get file info
- `POST` `/api/v1/sftp/sessions/{sessionId}/files/{path}` — Upload file
- `GET` `/api/v1/sftp/sessions/{sessionId}/files/{path}/download` — Download file
- `DELETE` `/api/v1/sftp/sessions/{sessionId}/files/{path}` — Delete file
- `POST` `/api/v1/sftp/sessions/{sessionId}/mkdir` — Create directory
- `POST` `/api/v1/sftp/sessions/{sessionId}/rename` — Rename
- `GET` `/api/v1/sftp/sessions/{sessionId}/transfers` — Transfer queue
- `POST` `/api/v1/sftp/sessions/{sessionId}/sync` — Directory sync
- `POST` `/api/v1/sftp/sessions/{sessionId}/checksum` — Verify checksum

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/sftp_service
export PORT=8080
go run ./cmd/sftp
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

*HelixTerminator SFTP Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
