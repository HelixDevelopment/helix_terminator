# PKI Service

HelixTerminator microservice — Issues short-lived SSH certificates (user + host), CA rotation, revocation checked by SSH Proxy

## Features
- Short-lived SSH certificate issuance (TTL=8h)
- User and host certificate signing
- CA rotation with zero-downtime
- CRL and OCSP revocation checking
- Revocation checked by SSH Proxy

## Module Path

`helixterminator.io/services/pki`

## Database

PostgreSQL helixterm_pki

## Upstream Dependencies

vault, audit

## API Endpoints

- `POST` `/api/v1/pki/certificates/user` — Issue user certificate
- `POST` `/api/v1/pki/certificates/host` — Issue host certificate
- `GET` `/api/v1/pki/certificates/{certId}` — Get certificate
- `POST` `/api/v1/pki/certificates/{certId}/revoke` — Revoke certificate
- `GET` `/api/v1/pki/crl` — Get CRL
- `POST` `/api/v1/pki/ca/rotate` — Rotate CA
- `GET` `/api/v1/pki/ca/status` — Get CA status

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/pki_service
export PORT=8080
go run ./cmd/pki
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

*HelixTerminator PKI Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
