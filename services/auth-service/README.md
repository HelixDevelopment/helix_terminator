# Auth Service

HelixTerminator microservice — Authentication (password, FIDO2/WebAuthn, TOTP, OIDC, SAML); JWT/refresh/device token issuance; SCIM inbound sync

## Features
- Multi-factor authentication (TOTP, FIDO2/WebAuthn)
- SSO/OIDC integration (Google, GitHub, Microsoft, Okta)
- SAML 2.0 Service Provider
- EdDSA (Ed25519) JWT signing
- SCIM 2.0 inbound provisioning
- Device management and revocation
- Session management with rotation

## Module Path

`helixterminator.io/services/auth`

## Database

PostgreSQL helixterm_auth

## Upstream Dependencies

user, vault, pki, notification, audit

## API Endpoints

- `POST` `/api/v1/auth/register` — Create user account
- `POST` `/api/v1/auth/login` — Authenticate
- `POST` `/api/v1/auth/logout` — Invalidate session
- `POST` `/api/v1/auth/refresh` — Exchange refresh token
- `POST` `/api/v1/auth/mfa/totp/setup` — Initiate TOTP setup
- `POST` `/api/v1/auth/mfa/totp/verify` — Verify TOTP code
- `POST` `/api/v1/auth/mfa/fido2/register/begin` — Begin FIDO2 registration
- `POST` `/api/v1/auth/mfa/fido2/register/complete` — Complete FIDO2 registration
- `POST` `/api/v1/auth/mfa/fido2/authenticate/begin` — Begin FIDO2 auth challenge
- `POST` `/api/v1/auth/mfa/fido2/authenticate/complete` — Complete FIDO2 auth
- `GET` `/api/v1/auth/devices` — List trusted devices
- `DELETE` `/api/v1/auth/devices/{deviceId}` — Revoke device
- `POST` `/api/v1/auth/sso/{provider}/authorize` — Initiate SSO OAuth flow
- `POST` `/api/v1/auth/sso/{provider}/callback` — Handle SSO callback
- `POST` `/api/v1/auth/api-keys` — Create API key
- `GET` `/api/v1/auth/api-keys` — List API keys
- `DELETE` `/api/v1/auth/api-keys/{keyId}` — Revoke API key
- `GET` `/api/v1/auth/sessions` — List active sessions
- `DELETE` `/api/v1/auth/sessions/{sessionId}` — Terminate session

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/auth_service
export PORT=8080
go run ./cmd/auth
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

*HelixTerminator Auth Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
