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

`POST /api/v1/notifications` accepts `channel: email|in_app|push|webhook|slack`
and, for `email`/`webhook`/`slack`, a required `target` field (recipient email
address, destination URL, or destination Slack channel ID — e.g. `C0123ABCD`
— respectively). The persisted `status` reflects the REAL delivery outcome —
it is never left permanently `pending`:

| Channel | Delivery mechanism | Requires | Success status | Failure status |
|---|---|---|---|---|
| `email` | Real SMTP send (`net/smtp`) to `target` | `SMTP_HOST` configured | `sent` | `failed` |
| `webhook` | Real outbound `http.Client` POST to `target` | none (any reachable http(s) URL) | `delivered` | `failed` |
| `push` | Real FCM HTTP v1 send to `target` (device registration token) | `FCM_SERVICE_ACCOUNT_JSON` configured | `sent` | `failed` |
| `slack` | Real Slack `chat.postMessage` (via the Herald submodule's Slack channel adapter — compiled in by default, no build tag) to `target` (a Slack channel ID) | `HERALD_SLACK_BOT_TOKEN` configured | `sent` | `failed` |
| `in_app` | No external transport; status is caller-supplied (default `pending`) | — | — | — |

Every channel without its required configuration reports the honest
`pending_provider_unconfigured` status rather than fabricating a delivery
outcome.

See `internal/delivery/` for the SMTP, webhook, push, and Slack clients.

**Slack build precondition.** `internal/delivery/slack_herald.go` imports the
Herald submodule's real Slack channel adapter directly (Constitution
§11.4.74 reuse-first) and compiles by default (no build tag — Constitution
§11.4.197: a wired feature must be active by default). This requires
`submodules/herald`'s own nested git submodules to be initialized —
`git -C submodules/herald submodule update --init --recursive` — which
this repository's project-wide submodule-init mandate (Constitution
§11.4.27/§11.4.36) already expects of every checkout. **Do not run a bare
`go mod tidy` on this service's `go.mod`**: it now technically succeeds,
but it forces `gin-gonic/gin` v1.10.0→v1.12.0 (plus several further
transitive bumps) project-wide, purely because Herald's own `commons`
module requires newer shared-dependency versions — an unrequested,
out-of-scope blast radius. See the hand-written comment directly above the
Herald `require`/`replace` block in `go.mod` for the full rationale; if a
gin bump is ever genuinely wanted, land and test it as its own change.

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
| `FCM_SERVICE_ACCOUNT_JSON` | No | — | Path to a Firebase/GCP service-account JSON key; push delivery is honestly reported as `pending_provider_unconfigured` when unset |
| `FCM_PROJECT_ID` | No | (from service account JSON) | Overrides the project id read from the service account key |
| `HERALD_SLACK_BOT_TOKEN` | No | — | Slack bot token (`xoxb-…`, `chat:write` scope) for the Herald-backed Slack channel adapter (see `internal/delivery/slack.go`), compiled in by default; Slack delivery is honestly reported as `pending_provider_unconfigured` when unset |

---

*HelixTerminator Notification Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
