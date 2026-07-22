# billing-service — Real Payment-Provider Integration (Stripe)

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## The bluff this document's fix closes

**Forensic anchor (Constitution §11.4 anti-bluff covenant).** Before
this fix, `internal/handler.CreateSubscription` persisted a new
subscription row with `Status: "active"` **unconditionally** — no
payment processor was ever contacted. A caller got back a 201 Created
and an `"active"` subscription regardless of whether any money had
moved, any card had been verified, or any processor had agreed to
anything at all. That is a textbook §11.4 PASS-bluff: the row claimed
the organization had a working paid subscription when nothing backed
that claim.

This fix closes it structurally, not cosmetically:

- A pluggable **`billing.PaymentProvider`** interface
  (`internal/billing/provider.go`) is now the *only* way a
  subscription's lifecycle (create / update / cancel) can be mutated.
- **`billing.StripeProvider`** (`internal/billing/stripe_provider.go`)
  is a real implementation backed by the official
  [Stripe Go SDK v86](https://github.com/stripe/stripe-go) — every
  call it makes is a genuine HTTP request to Stripe's API.
- When no provider is configured, every subscription-lifecycle-mutating
  endpoint responds **`501 Not Implemented`** with
  `{"error": "payments provider not configured"}` — **never** a
  fabricated success, on either code path.
- Every persisted subscription row now carries `provider`,
  `external_subscription_id`, and `external_customer_id` (migration
  `002_payment_provider`) — proof-of-real-call baked into the schema
  itself, not just the API response.

## Table of contents

- [Architecture](#architecture)
- [The `PaymentProvider` interface](#the-paymentprovider-interface)
- [The honest feature-flag](#the-honest-feature-flag)
- [API changes](#api-changes)
- [Stripe setup (test mode)](#stripe-setup-test-mode)
- [Environment variables / key provisioning](#environment-variables--key-provisioning)
- [Webhook configuration](#webhook-configuration)
- [Running the tests](#running-the-tests)
- [Honest boundaries / what this does NOT do](#honest-boundaries--what-this-does-not-do)
- [Sources verified](#sources-verified)

## Architecture

```
internal/billing/
  provider.go                          PaymentProvider interface + shared types
  stripe_provider.go                   StripeProvider — REAL Stripe Go SDK calls
  env.go                                NewProviderFromEnv() — the honest feature-flag
  env_test.go                          unit tests
  stripe_provider_test.go              unit tests (fake stripeClient + REAL webhook crypto)
  stripe_provider_integration_test.go  //go:build integration — REAL Stripe test API

internal/handler/handler.go             wires PaymentProvider into every subscription endpoint
internal/server/server.go               constructs the provider from env at startup, mounts
                                         POST /api/v1/webhooks/stripe (outside JWT auth)
internal/repository/repository.go       persists provider/external_subscription_id/
                                         external_customer_id; GetLatestExternalCustomerID
                                         reuses a tenant's existing processor customer
migrations/002_payment_provider.{up,down}.sql
```

`billing.PaymentProvider` is deliberately processor-agnostic. Stripe is
**one** implementation; a future change can add Paddle, Braintree, or
any other processor by implementing the same four-method interface —
`internal/handler` never imports the Stripe SDK directly.

## The `PaymentProvider` interface

```go
type PaymentProvider interface {
    Name() string
    CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (*SubscriptionResult, error)
    UpdateSubscription(ctx context.Context, in UpdateSubscriptionInput) (*SubscriptionResult, error)
    CancelSubscription(ctx context.Context, in CancelSubscriptionInput) (*SubscriptionResult, error)
    VerifyWebhook(payload []byte, signatureHeader string) (*WebhookEvent, error)
}
```

`SubscriptionResult.Status` always carries the **real** status string
the processor returned (`"active"`, `"incomplete"`, `"past_due"`,
`"canceled"`, …). The handler layer persists this value verbatim — it
never assumes "the call succeeded" implies `"active"`. A processor can
accept a create call and still return a non-active status (e.g.
`"incomplete"` when a payment action is required); reporting that
honestly is the entire point of this package.

### `StripeProvider` design decisions

- **Collection method: `send_invoice`, 30-day terms.** Stripe finalizes
  and emails an invoice rather than attempting to auto-charge a card on
  file. This lets a subscription become genuinely `"active"` via the
  API alone, without first requiring a separate client-side
  card-collection flow (Stripe Elements/Checkout) that billing-service
  does not implement. It also matches this service's existing
  `invoices` table/endpoints, which already model invoice-based billing
  rather than instant-charge-card billing. To switch to
  `charge_automatically`, extend `CreateSubscriptionInput` with a
  Stripe `PaymentMethod` id and thread it into
  `StripeProvider.CreateSubscription`'s `SubscriptionCreateParams`.
- **Customer reuse, never duplication.** `CreateSubscription` creates a
  Stripe Customer only when `ExistingCustomerID` is empty.
  `internal/handler.CreateSubscription` looks up the caller's org's
  most recent Stripe customer id (`Repository.GetLatestExternalCustomerID`)
  before calling the provider, so a second subscription for the same
  org reuses the same Stripe Customer.
- **Idempotency.** Every Stripe-mutating call carries a
  processor-native idempotency key derived from the client-supplied
  `Idempotency-Key` HTTP header (or a fresh UUID when the client
  supplies none) — a retried request can never create two Stripe
  customers or two Stripe subscriptions for the same logical attempt.
- **Plan-change replaces the price, never appends a second item.**
  Stripe's subscription-item update API adds a *new* item when no item
  `id` is supplied — `UpdateSubscription` retrieves the subscription
  first to discover its existing item id, then targets that id
  explicitly. See the doc comment on `StripeProvider.UpdateSubscription`
  for the captured evidence that motivated this (a naive
  price-only update would have silently double-billed).
- **Webhook API-version mismatch is not fail-closed.** Stripe's default
  `webhook.ConstructEvent` additionally rejects any event whose
  `api_version` does not exactly match the SDK's compiled-in version.
  `StripeProvider.VerifyWebhook` uses
  `ConstructEventWithOptions(..., IgnoreAPIVersionMismatch: true)` — the
  **signature** check (the actual trust boundary) still fails closed;
  the API-version check is a compatibility hint billing-service does
  not want to turn into a self-inflicted webhook outage over, since it
  does not control the Stripe account's platform-wide default API
  version. See the doc comment on `VerifyWebhook` for the captured
  error text this decision is based on.

## The honest feature-flag

`internal/billing.NewProviderFromEnv()`:

| `STRIPE_SECRET_KEY` | Result |
|---|---|
| unset / empty | `(nil, nil)` — **not** an error. Every subscription-lifecycle-mutating endpoint responds `501 Not Implemented`. |
| set | `(*StripeProvider, nil)` — every subsequent call this process makes is REAL, against whatever kind of key was supplied (`sk_test_...` or `sk_live_...` — this package never inspects the prefix; that judgment belongs to whoever provisions the environment). |

`internal/server.New` calls this exactly once at startup (mirroring the
existing `JWT_PUBLIC_KEY` env-read pattern) and logs which state the
process is in:

```
billing-service: payments provider "stripe" configured — subscription lifecycle calls are REAL
```
or
```
billing-service: no payments provider configured (STRIPE_SECRET_KEY unset) — subscription-lifecycle-mutating endpoints will respond 501 Not Implemented
```

`STRIPE_WEBHOOK_SECRET` may be left empty even when `STRIPE_SECRET_KEY`
is set — subscription create/update/cancel then work normally, but
`StripeProvider.VerifyWebhook` always fails closed (rejects every
payload) until it is also configured. There is no "verify without a
secret" mode — that would mean trusting an unverified payload.

## API changes

### `POST /api/v1/subscriptions`

```json
{
  "planId": "3fa85f64-...-46656b0e-0002",
  "stripePriceId": "price_1AbCdEfGhIjKlMnO"
}
```

- `planId` — unchanged: billing-service's own internal plan reference
  (a UUID), attributed to the caller's authenticated org (T14 — never
  from client input).
- `stripePriceId` — **new, required whenever a provider is
  configured**. The Stripe Price object id the subscription is created
  against. Validated at the business-logic layer (not a static binding
  tag) because "required" here is conditional on runtime provider
  configuration.
- With no provider configured: `501 {"error": "payments provider not configured"}`.
- With a provider configured but no `stripePriceId`: `400 {"error": "stripePriceId is required when a payment provider is configured"}`.
- On success: `201` with `status` set to the **real** Stripe status,
  plus `provider` and `externalSubscriptionId` in the response body.

### `PUT /api/v1/subscriptions/:id`

```json
{ "planId": "...", "stripePriceId": "price_..." }
```
or
```json
{ "status": "canceled" }
```

- `status`'s allowed values are now **`canceled` / `expired` only** —
  `"active"` was **removed** from the allowed set. Reactivating a
  subscription is a real processor-side event (a resumed subscription,
  a paid invoice) that must be *learned from the processor*, never
  asserted directly by a bare PUT with zero processor involvement —
  that PUT-status-to-active path was the same class of bluff this whole
  fix closes, just at the update endpoint instead of create.
- `planId` + `stripePriceId` together change price. If the target
  subscription is processor-backed (`externalSubscriptionId != ""`),
  this calls the configured provider's `UpdateSubscription` for real
  and persists the real returned status; with no provider configured,
  `501`. If the subscription was never processor-backed, the change is
  local-only bookkeeping (no processor truth to diverge from).
- `planId` and `status` **cannot** be supplied in the same request
  (`400`) — this removes an ordering ambiguity about which value would
  "win".

### `POST /api/v1/subscriptions/:id/cancel`

Unchanged request shape. If the subscription is processor-backed, this
now calls the configured provider's `CancelSubscription` for real and
persists the **real** returned status (not a hardcoded `"canceled"`
literal) — with no provider configured, `501`. If the subscription was
never processor-backed, cancellation remains local-only.

### `POST /api/v1/webhooks/stripe` (new)

Mounted **outside** the JWT `authMiddleware` group — Stripe
authenticates a webhook delivery via its own `Stripe-Signature` header
scheme, never a bearer token. Verifies the payload via
`PaymentProvider.VerifyWebhook`; for `customer.subscription.updated`
and `customer.subscription.deleted` events, reconciles the matching
local subscription row's status to what Stripe now reports (closing
the gap where a processor-initiated change — e.g. a failed payment
auto-canceling a subscription — would otherwise never reach this
service's own records). Other event types are acknowledged (`200`)
without action.

## Stripe setup (test mode)

1. Create a free Stripe account (or use an existing one) —
   <https://dashboard.stripe.com/register>.
2. Switch the dashboard to **Test mode** (toggle top-right).
3. **Developers → API keys** — copy the **Secret key**
   (`sk_test_...`). This is `STRIPE_SECRET_KEY`.
4. **Product catalog → Add product** — create a product with a
   **recurring** Price (any interval/amount). Copy the Price id
   (`price_...`). This is the value the integration tests need as
   `STRIPE_TEST_PRICE_ID`, and the value real API callers pass as
   `stripePriceId`.
5. **Developers → Webhooks → Add endpoint** (only needed if you intend
   to exercise webhook delivery) — see
   [Webhook configuration](#webhook-configuration) below.

No card details, no real money, and no live-mode data are involved at
any point in this walkthrough — Stripe's test mode is a fully
functional sandbox against the real API.

## Environment variables / key provisioning

| Variable | Required | Purpose |
|---|---|---|
| `STRIPE_SECRET_KEY` | No (honest 501 when absent) | Stripe API secret key (`sk_test_...` / `sk_live_...`). |
| `STRIPE_WEBHOOK_SECRET` | No (webhook verification fails closed when absent) | Stripe webhook endpoint signing secret (`whsec_...`). |
| `STRIPE_TEST_PRICE_ID` | Test-only | A real Stripe **test-mode** recurring Price id — used by the integration/stress/chaos test suites (never by production code) to create real test-mode subscriptions. |

**Constitution §11.4.10 — these are secrets and MUST NEVER be committed.**

- Export them as shell environment variables (`export STRIPE_SECRET_KEY=...`)
  for local runs, or via your deployment platform's secret-injection
  mechanism (Kubernetes `Secret`, Docker `--env-file` pointed at a
  gitignored file, etc.) in every other environment.
- billing-service's `.gitignore` inheritance (Constitution §11.4.30)
  already excludes `.env` / `.env.*` project-wide — never add a
  `STRIPE_SECRET_KEY=sk_test_...` line to any file tracked by git,
  including test fixtures or example configs. `.env.example` (if you
  add one) MUST contain only a placeholder, e.g.
  `STRIPE_SECRET_KEY=sk_test_replace_me`.
- This service's `infrastructure/docker/compose/docker-compose.yml` /
  `.env.example` do not yet declare `STRIPE_SECRET_KEY` /
  `STRIPE_WEBHOOK_SECRET` — wiring the compose-level env passthrough
  (mirroring the existing `JWT_PUBLIC_KEY: ${JWT_PUBLIC_KEY:-}`
  pattern) is out of this document's scope (`services/billing-service/**`
  only) and is a follow-up item for whoever owns the infra compose
  file.
- Before storing any key an operator supplies, audit for prior
  accidental leaks per Constitution §11.4.10.A
  (`git ls-files | xargs grep -l <value>` and
  `git log -S<value> --all --source --remotes`) BEFORE persisting it
  anywhere, even in a gitignored file, and surface any finding to the
  operator before proceeding.
- If a key is ever suspected leaked, rotate it immediately from the
  Stripe dashboard (**Developers → API keys → Roll key**) — the old key
  is invalidated the moment you do.

## Webhook configuration

1. **Developers → Webhooks → Add endpoint** in the Stripe dashboard.
2. **Endpoint URL**: `https://<your-billing-service-host>/api/v1/webhooks/stripe`
   (this route is unauthenticated at the HTTP layer — Stripe's payload
   signature is the trust boundary, not a bearer token).
3. **Events to send**: at minimum `customer.subscription.updated` and
   `customer.subscription.deleted` (the two event types
   `internal/handler.StripeWebhook` currently reconciles).
4. After creating the endpoint, copy its **Signing secret**
   (`whsec_...`) into `STRIPE_WEBHOOK_SECRET`.
5. **Local development** — Stripe cannot reach `localhost` directly.
   Use the [Stripe CLI](https://docs.stripe.com/stripe-cli) to forward
   real webhook deliveries to a local process:
   ```bash
   stripe login
   stripe listen --forward-to localhost:8087/api/v1/webhooks/stripe
   ```
   The CLI prints a `whsec_...` value scoped to that forwarding
   session — export it as `STRIPE_WEBHOOK_SECRET` for local testing.
   `stripe trigger customer.subscription.updated` then sends a real,
   correctly-signed test event through the forward.

## Running the tests

### Unit tests (always run, no external dependencies)

```bash
cd services/billing-service
GOWORK=off GOMAXPROCS=2 go test -p 2 ./...
```

Covers: `internal/billing` (provider construction, parameter
construction against a fake `stripeClient`, and a **real cryptographic**
webhook-signature round-trip using the Stripe SDK's own
`webhook.ComputeSignature` — no crypto is faked), `internal/handler`
(the honest-501 gate, ownership/identity checks, mutual-exclusion
validation — several of these tests also spin up a REAL disposable
PostgreSQL container via rootless podman to prove persistence, per
Constitution §11.4.27), `internal/repository`, `internal/model`,
`migrations`.

### The honest-501 proof (no key set → 501)

Captured directly from this repository (`internal/handler/handler_test.go`,
`TestCreateSubscription_NoProvider_Returns501` +
`TestCreateSubscription_NoProvider_NeverFabricatesActive`):

```
$ GOWORK=off GOMAXPROCS=2 go test -p 2 -run TestCreateSubscription_NoProvider -v ./internal/handler/...
=== RUN   TestCreateSubscription_NoProvider_Returns501
--- PASS: TestCreateSubscription_NoProvider_Returns501 (0.00s)
=== RUN   TestCreateSubscription_NoProvider_NeverFabricatesActive
--- PASS: TestCreateSubscription_NoProvider_NeverFabricatesActive (0.00s)
PASS
ok  	github.com/helixdevelopment/billing-service/internal/handler	0.010s
```

The server-level equivalent (`internal/server`, real HTTP server, real
Postgres, `STRIPE_SECRET_KEY` unset) logs the same honest state at
startup:

```
billing-service: no payments provider configured (STRIPE_SECRET_KEY unset) — subscription-lifecycle-mutating endpoints will respond 501 Not Implemented
```

### Integration tests (real Stripe test API — build tag `integration`)

Structured to SKIP honestly (§11.4.3) when no real Stripe test
credentials are provisioned, and to run for real the moment they are:

```bash
cd services/billing-service
export STRIPE_SECRET_KEY="sk_test_..."
export STRIPE_TEST_PRICE_ID="price_..."
export STRIPE_WEBHOOK_SECRET="whsec_..."          # optional, only needed for webhook flows
GOWORK=off GOMAXPROCS=2 go test -tags integration -p 2 -v -run TestStripeProvider_Integration ./internal/billing/...
```

Drives a REAL create → update-price-guard → cancel cycle against the
real Stripe test-mode API (`TestStripeProvider_Integration_FullSubscriptionLifecycle`),
proves a genuinely-invalid price is rejected by the real API
(`TestStripeProvider_Integration_CreateSubscription_InvalidPriceRejected`),
and proves customer reuse creates two distinct real subscriptions under
one real customer (`TestStripeProvider_Integration_ReusesExistingCustomer`).

The T12/T14 cross-tenant-isolation integration suite
(`internal/server`, `-tags integration`, requires `DATABASE_URL`) also
has one subtest gated the same way
(`CreateSubscription_ignores_client-supplied_orgId...`) — the other
subtests in that suite (Update/Cancel ownership checks) exercise rows
seeded directly via the repository with no processor involved, so they
run unconditionally.

### Stress / chaos tests (build tags `stress` / `chaos`)

Per Constitution §11.4.27(A), these NEVER substitute a fake payment
provider — every subtest that needs a genuinely-created subscription
requires the SAME real Stripe test credentials as the integration
suite (`STRIPE_SECRET_KEY` + `STRIPE_TEST_PRICE_ID`) and SKIPs
individually, with an honest reason, when they are absent:

```bash
export STRIPE_SECRET_KEY="sk_test_..."
export STRIPE_TEST_PRICE_ID="price_..."
GOWORK=off GOMAXPROCS=2 go test -race -tags stress -p 2 -run TestStress -v -timeout 120s ./internal/handler/...
GOWORK=off GOMAXPROCS=2 go test -race -tags chaos  -p 2 -run TestChaos  -v -timeout 120s ./internal/handler/...
```

## Honest boundaries / what this does NOT do

- **No client-side card collection flow.** `send_invoice` collection
  means real card/payment-method collection (Stripe Elements/Checkout,
  3-D Secure/SCA) is not implemented by this fix. Adding it is a
  distinct, larger change (frontend + `charge_automatically` +
  `payment_intent` handling) explicitly out of scope here.
  `Operator-blocked`-class follow-up, not silently claimed as done.
- **Webhook reconciliation covers two event types.** Only
  `customer.subscription.updated` / `customer.subscription.deleted`
  are reconciled today. Invoice-payment events
  (`invoice.paid` / `invoice.payment_failed`) are received (verified,
  acknowledged `200`) but not yet reconciled into the `invoices` table
  — a real, bounded follow-up.
- **Plan catalog / Price mapping is caller-supplied, not
  service-managed.** billing-service's `billing_plans` table is not
  queried by any handler (pre-existing, before this fix) — `planId`
  remains an opaque internal reference, and the caller supplies the
  Stripe `stripePriceId` directly. A future change could add a
  `billing_plans.stripe_price_id` column and resolve it server-side;
  this fix deliberately did not invent that mapping to avoid scope
  creep beyond "make the payment provider real."

## Sources verified

**2026-07-22:**

- <https://github.com/stripe/stripe-go> — current release (v86.1.1),
  install command, `stripe.NewClient` / `V1Customers` / `V1Subscriptions`
  usage, supported Go version policy.
- <https://github.com/stripe/stripe-go/wiki/Migration-guide-for-Stripe-Client> —
  `stripe.Client` vs the legacy `client.API` pattern, method-naming
  (`Create`/`Retrieve`/`Update`/`Delete`), `context.Context` as the
  first argument, version-namespaced services (`V1Subscriptions`).
- <https://pkg.go.dev/github.com/stripe/stripe-go/v86/webhook> —
  `webhook.ConstructEvent` / `ConstructEventWithOptions` /
  `ComputeSignature` signatures, default tolerance, error types.
- <https://proxy.golang.org/github.com/stripe/stripe-go/> (`@v/list`,
  `@latest` per major) — confirmed v86.1.1 is the current major/version
  as of this date (v82–v86 all queried; v86.1.1 released 2026-07-16).
- <https://docs.stripe.com/billing/subscriptions/change-price#changing> —
  cited (via `stripe-go`'s own generated doc comments on
  `SubscriptionUpdateItemParams.ID`) for the "omitting an item id ADDS
  a second item instead of replacing the price" semantics
  `StripeProvider.UpdateSubscription` guards against.
- Source code fetched directly from
  `raw.githubusercontent.com/stripe/stripe-go/v86.1.1/{subscription.go,subscription_service.go,customer.go,params.go,stripe_client.go,event.go,webhook/client.go}`
  via the GitHub API/raw content — struct fields, service method
  signatures, and `Params.SetIdempotencyKey` verified directly against
  the actual v86.1.1 source, not assumed from training data.
