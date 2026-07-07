# Keychain Service

HelixTerminator microservice — Hardware-backed key storage (Secure Enclave / Android Keystore / DPAPI / kernel keyring / HSM), vault key wrap/unwrap, key rotation; gRPC-only, no REST

## Features
- Hardware-backed key storage (Secure Enclave, Android Keystore, DPAPI)
- Vault key wrap/unwrap operations
- Automatic key rotation
- HSM integration (AWS KMS, HashiCorp Vault)
- gRPC-only interface (no REST surface)

## Module Path

`helixterminator.io/services/keychain`

## Database

PostgreSQL helixterm_keychain

## Upstream Dependencies

audit

## API Endpoints

- `gRPC` `WrapKey` — Wrap a key for vault storage
- `gRPC` `UnwrapKey` — Unwrap a key from vault storage
- `gRPC` `RotateKey` — Rotate encryption key
- `gRPC` `GetHardwareKey` — Retrieve hardware-backed key
- `gRPC` `GenerateKey` — Generate new key pair
- `gRPC` `ImportKey` — Import external key
- `gRPC` `ExportKey` — Export key (with authorization)

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/keychain_service
export PORT=50051
go run ./cmd/keychain
```

## Testing

```bash
go test -v -race -cover ./...
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PORT` | No | 50051 | HTTP/gRPC port |
| `LOG_LEVEL` | No | info | Log level (debug/info/warn/error) |
| `KAFKA_BROKERS` | No | — | Kafka bootstrap servers |
| `REDIS_URL` | No | — | Redis connection string |

---

*HelixTerminator Keychain Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
