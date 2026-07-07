# User Service

HelixTerminator microservice — User CRUD, profile, preferences, onboarding state machine, SCIM provisioning endpoint

## Features
- User CRUD with profile management
- User preferences storage
- Onboarding state machine
- SCIM provisioning endpoint
- Session management (list, revoke)
- Activity tracking

## Module Path

`helixterminator.io/services/user`

## Database

PostgreSQL helixterm_users

## Upstream Dependencies

org, notification, audit

## API Endpoints

- `GET` `/api/v1/users/me` — Get current user profile
- `PATCH` `/api/v1/users/me` — Update profile
- `DELETE` `/api/v1/users/me` — Delete account
- `GET` `/api/v1/users/me/preferences` — Get preferences
- `PUT` `/api/v1/users/me/preferences` — Update preferences
- `POST` `/api/v1/users/me/avatar` — Upload avatar
- `GET` `/api/v1/users/me/sessions` — List sessions
- `DELETE` `/api/v1/users/me/sessions/{sessionId}` — Revoke session
- `GET` `/api/v1/users/me/activity` — Get activity tracking

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/user_service
export PORT=8080
go run ./cmd/user
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

*HelixTerminator User Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
