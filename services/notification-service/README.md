# Notification Service

HelixTerminator microservice — Multi-channel notification delivery (in-app, email, push, Slack, webhooks), templates, dedup, digests

## Features
- Multi-channel delivery (in-app, email, push, Slack, webhooks)
- Template management with variable substitution
- Deduplication and throttling
- Digest bundling for batch delivery
- Delivery tracking and retry logic

## Module Path

`helixterminator.io/services/notification`

## Database

PostgreSQL helixterm_notifications

## Upstream Dependencies

user, audit

## API Endpoints

- `POST` `/api/v1/notifications/send` — Send notification
- `GET` `/api/v1/notifications` — List notifications
- `GET` `/api/v1/notifications/{notificationId}` — Get notification
- `POST` `/api/v1/notifications/{notificationId}/read` — Mark as read
- `DELETE` `/api/v1/notifications/{notificationId}` — Delete notification
- `GET` `/api/v1/notifications/templates` — List templates
- `POST` `/api/v1/notifications/templates` — Create template
- `PUT` `/api/v1/notifications/templates/{templateId}` — Update template
- `GET` `/api/v1/notifications/digests` — Get digest settings
- `PUT` `/api/v1/notifications/digests` — Update digest settings

## Delivery Channels

`POST /api/v1/notifications` accepts `channel: email|in_app|push|webhook` and,
for `email`/`webhook`, a required `target` field (recipient email address, or
destination URL respectively). The persisted `status` reflects the REAL
delivery outcome — it is never left permanently `pending`:

| Channel | Delivery mechanism | Requires | Success status | Failure status |
|---|---|---|---|---|
| `email` | Real SMTP send (`net/smtp`) to `target` | `SMTP_HOST` configured | `sent` | `failed` |
| `webhook` | Real outbound `http.Client` POST to `target` | none (any reachable http(s) URL) | `delivered` | `failed` |
| `push` | Not yet implemented — FCM/APNs credentials are not configured in this environment | operator-supplied `FCM_SERVER_KEY` / `APNS_*` (not yet wired) | — | `pending_provider_unconfigured` (honest, never fabricated) |
| `in_app` | No external transport; status is caller-supplied (default `pending`) | — | — | — |

See `internal/delivery/` for the SMTP and webhook clients.

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/notification_service
export PORT=8080
go run ./cmd/notification
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
| `SMTP_HOST` | No | — | SMTP server host; email delivery is honestly reported as `failed` when unset |
| `SMTP_PORT` | No | 25 | SMTP server port |
| `SMTP_FROM` | No | notifications@localhost | Envelope/header From address |
| `SMTP_USERNAME` | No | — | SMTP AUTH username (PLAIN auth used only when set) |
| `SMTP_PASSWORD` | No | — | SMTP AUTH password — never hardcode, never commit |

---

*HelixTerminator Notification Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
