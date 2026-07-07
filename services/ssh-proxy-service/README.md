# SSH Proxy Service

HelixTerminator microservice — Brokers SSH connections (password/pubkey/certificate auth), container-native sessions, proxy-jump chains, agent forwarding

## Features
- SSH connection brokering (password, pubkey, certificate auth)
- Container-native sessions (exec into containers)
- Proxy-jump chain configuration
- SSH agent forwarding
- Host key verification and caching
- Session recording integration

## Module Path

`helixterminator.io/services/ssh-proxy`

## Database

PostgreSQL helixterm_ssh_proxy (connection state only)

## Upstream Dependencies

auth, vault, host, terminal, audit, recording, pki, container-bridge

## API Endpoints

- `POST` `/api/v1/ssh/sessions` — Create SSH session
- `GET` `/api/v1/ssh/sessions/{sessionId}` — Get session
- `DELETE` `/api/v1/ssh/sessions/{sessionId}` — Terminate session
- `POST` `/api/v1/ssh/sessions/{sessionId}/exec` — Execute command
- `POST` `/api/v1/ssh/sessions/{sessionId}/shell` — Request shell
- `POST` `/api/v1/ssh/sessions/{sessionId}/agent` — Enable agent forwarding
- `POST` `/api/v1/ssh/sessions/{sessionId}/jump` — Add proxy jump
- `GET` `/api/v1/ssh/sessions/{sessionId}/keys` — List host keys
- `POST` `/api/v1/ssh/sessions/{sessionId}/keys/verify` — Verify host key

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/ssh-proxy_service
export PORT=8080
go run ./cmd/ssh-proxy
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

*HelixTerminator SSH Proxy Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
