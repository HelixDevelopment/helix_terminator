# Billing Service

HelixTerminator microservice — Subscription/seat management, Stripe integration, invoicing, usage metering, trials, dunning

## Features
- Subscription and seat management
- Stripe integration for payments
- Usage metering and billing
- Trial management
- Dunning (failed payment recovery)
- Invoice generation and delivery

## Module Path

`helixterminator.io/services/billing`

## Database

PostgreSQL helixterm_billing

## Upstream Dependencies

org, user, notification, audit

## API Endpoints

- `GET` `/api/v1/billing/subscription` — Get subscription
- `POST` `/api/v1/billing/subscription` — Create subscription
- `PUT` `/api/v1/billing/subscription` — Update subscription
- `DELETE` `/api/v1/billing/subscription` — Cancel subscription
- `GET` `/api/v1/billing/invoices` — List invoices
- `GET` `/api/v1/billing/invoices/{invoiceId}` — Get invoice
- `POST` `/api/v1/billing/payment-method` — Add payment method
- `DELETE` `/api/v1/billing/payment-method/{methodId}` — Remove payment method
- `GET` `/api/v1/billing/usage` — Get usage metering
- `POST` `/api/v1/billing/trial` — Start trial

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/billing_service
export PORT=8080
go run ./cmd/billing
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
| `JWT_PUBLIC_KEY` | No (401 on all `/api/v1/*` when unset) | — | base64 Ed25519 public key, same key `auth-service`/`gateway-service` use |
| `STRIPE_SECRET_KEY` | No (honest `501` on subscription-lifecycle endpoints when unset) | — | Stripe API secret key (`sk_test_...`/`sk_live_...`) — see `docs/guides/BILLING.md` |
| `STRIPE_WEBHOOK_SECRET` | No (webhook verification fails closed when unset) | — | Stripe webhook endpoint signing secret (`whsec_...`) — see `docs/guides/BILLING.md` |

## Payments / Stripe integration

Subscription create/update/cancel are backed by a pluggable
`billing.PaymentProvider` (`internal/billing/`), with a real Stripe Go
SDK implementation. With no `STRIPE_SECRET_KEY` configured, the
subscription-lifecycle-mutating endpoints honestly respond `501 Not
Implemented` — never a fabricated success. Full architecture, Stripe
setup, webhook configuration, and test instructions:
**`docs/guides/BILLING.md`**.

---

*HelixTerminator Billing Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
