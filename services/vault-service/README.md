# Vault Service

HelixTerminator microservice — Zero-knowledge encrypted storage for SSH keys, passwords, API tokens, TLS certs, secret notes; sharing + versioning

## Features
- Zero-knowledge encrypted storage (client-side encryption)
- Multiple item types (password, SSH key, API token, TLS cert, note)
- Vault sharing with role-based access
- Item versioning and history
- Key rotation support
- Shamir's secret sharing for recovery

## Module Path

`helixterminator.io/services/vault`

## Database

PostgreSQL helixterm_vault

## Upstream Dependencies

keychain, audit, pki

## API Endpoints

- `GET` `/api/v1/vaults` — List vaults
- `POST` `/api/v1/vaults` — Create vault
- `GET` `/api/v1/vaults/{vaultId}` — Get vault
- `PUT` `/api/v1/vaults/{vaultId}` — Update vault
- `DELETE` `/api/v1/vaults/{vaultId}` — Delete vault
- `GET` `/api/v1/vaults/{vaultId}/items` — List items
- `POST` `/api/v1/vaults/{vaultId}/items` — Create item
- `GET` `/api/v1/vaults/{vaultId}/items/{itemId}` — Get item
- `PUT` `/api/v1/vaults/{vaultId}/items/{itemId}` — Update item
- `DELETE` `/api/v1/vaults/{vaultId}/items/{itemId}` — Delete item
- `GET` `/api/v1/vaults/{vaultId}/members` — List members
- `POST` `/api/v1/vaults/{vaultId}/members` — Add member
- `DELETE` `/api/v1/vaults/{vaultId}/members/{userId}` — Remove member
- `POST` `/api/v1/vaults/{vaultId}/sync` — Sync vault
- `GET` `/api/v1/vaults/{vaultId}/history` — Item history

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/vault_service
export PORT=8080
go run ./cmd/vault
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

*HelixTerminator Vault Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
