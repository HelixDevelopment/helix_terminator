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

---

*HelixTerminator Notification Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
