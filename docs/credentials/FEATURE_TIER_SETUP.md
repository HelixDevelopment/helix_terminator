# Feature-tier credential setup — operator checklist

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## Scope of this document

Every feature tier described below is **already implemented and merged
on `main`** — real Stripe payment calls, real FCM/APNs push delivery,
real SMTP email delivery, and a real local-LLM client. This document
does **not** implement, stub, wire, or change any of that code. It is
**operator-facing config guidance only**: which environment variable to
set, which credential file it expects, and how to confirm — from real,
cited runtime behaviour — that a tier actually activated once you supply
credentials.

Every variable named below was verified by reading the real
`os.Getenv` call that consumes it (Constitution §11.4.6 no-guessing —
nothing here is invented). Each citation is `file:line` in this
checkout. Where a tier's activation is observable only through a
notification/subscription **status value** rather than a startup log
line, that is stated explicitly — it is not a gap in this document, it
is the real, current behaviour of the code.

| Tier | Service | Status |
|---|---|---|
| [Push notifications](#push-notifications-fcm--apns-via-fcm) | notification-service | Implemented |
| [Payments](#payments-stripe) | billing-service | Implemented |
| [Email](#email-smtp) | notification-service | Implemented |
| [AI / LLM](#ai--llm) | ai-service | **Local tier** implemented; **cloud tier NOT implemented** (see below) |

## A known gap this document does not close

`infrastructure/docker/compose/docker-compose.yml` does not yet pass
`STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` / `FCM_SERVICE_ACCOUNT_JSON`
/ `FCM_PROJECT_ID` / `SMTP_*` / `AI_LOCAL_PROVIDER_*` through to each
service's `environment:` block (the same gap `docs/guides/BILLING.md`
already documents for Stripe specifically, under "Environment variables
/ key provisioning"). Populating
`infrastructure/docker/compose/.env.example`-derived values in a local
`.env` is therefore **necessary but not yet sufficient** for a
`docker compose up` deployment to pick them up — the compose-level
passthrough (mirroring the existing
`JWT_PUBLIC_KEY: ${JWT_PUBLIC_KEY:-}` pattern already used for
billing-service) is a tracked follow-up, not something this document
implements. **Running a service binary directly** (`go run
./cmd/<service>` or the built binary) with the variables exported in
your shell, or sourced from a local file under `scripts/testing/secrets/`,
picks them up immediately — that path does not depend on the
compose-passthrough gap.

---

## Push notifications (FCM + APNs-via-FCM)

**Real implementation:**
`services/notification-service/internal/delivery/push.go` — a from-scratch
FCM HTTP v1 client (OAuth2 JWT-bearer service-account flow, no vendored
Firebase SDK) that also delivers to iOS devices via FCM's built-in APNs
bridge.

### Environment variables

| Variable | Required | Read at | Purpose |
|---|---|---|---|
| `FCM_SERVICE_ACCOUNT_JSON` | Yes, to enable push | `push.go:135` (`PushConfigFromEnv`) | Filesystem path to a Firebase/GCP service-account JSON key with Cloud Messaging send permission. |
| `FCM_PROJECT_ID` | No | `push.go:141` | Overrides the `project_id` embedded in the service-account JSON. Leave unset to use the key's own project. |

### Which secret file, where

`FCM_SERVICE_ACCOUNT_JSON` must point at a real Google/Firebase
service-account key JSON — the "Generate new private key" download from
**Firebase Console → Project Settings → Service Accounts**, or
`gcloud iam service-accounts keys create`.

The project already ships an automation script for this:
`scripts/firebase/firebase_setup.sh`. Run it; by default it writes the
key to `scripts/firebase/secrets/fcm-service-account.json` (override the
output directory with `FIREBASE_SETUP_OUTPUT_DIR`, or the exact key path
with `FIREBASE_SERVICE_ACCOUNT_KEY_PATH`). That directory is **already**
gitignored (`scripts/firebase/secrets/` in the root `.gitignore`) —
point `FCM_SERVICE_ACCOUNT_JSON` at whatever path the script prints.

If you intend to deliver to **iOS** devices, Firebase's APNs bridge
additionally requires you to upload an **APNs Authentication Key**
(a `.p8` file, plus its Key ID and your Apple Developer Team ID) once,
manually, in **Firebase Console → Project Settings → Cloud Messaging →
Apple app configuration**. `push.go` never reads the `.p8` file itself
(Firebase's own infrastructure holds it and performs the APNs bridge —
see `push.go`'s package doc comment) — but if you keep a local copy of
that key for reference, store it under `scripts/testing/secrets/` (or
anywhere outside the tree) and never commit it; the root `.gitignore`
now covers `*.p8` / `*.p12` project-wide as a defense-in-depth measure.

### Honest default when unset

`FCM_SERVICE_ACCOUNT_JSON` unset → `notification-service` starts with
an **unconfigured** `PushSender` (`delivery.NewPushSender()`, the
zero-credential state). Every push notification's `Send` call returns
`ErrPushProviderNotConfigured`, and the notification row's `Status` is
honestly set to `"pending_provider_unconfigured"`
(`handler.go:250-253`) — never a fabricated `"sent"`.

Set-but-broken (the variable is set but the file cannot be read or
parsed as a valid Google service-account key) is deliberately **not**
collapsed into the same "not configured" bucket — it is a real
misconfiguration and surfaces at startup as:

```
[notify] push (FCM) configuration error, falling back to honest not-configured state: <error>
```

(`handler.go:52`) — the service still starts (falls back to the
unconfigured sender) rather than crash-looping, but the log line names
exactly what is broken.

### How to know it activated

There is **no explicit "push provider configured" startup log line** in
the current code (unlike Stripe below) — this is the real, current
behaviour, not an omission in this document. The positive signals are:

1. **Absence** of the `[notify] push (FCM) configuration error...` log
   line at startup (its presence means the variable is set but broken).
2. A real push notification's `status` field becomes `"sent"`
   (`handler.go:268-270`) rather than `"pending_provider_unconfigured"`
   after `deliverPush` runs — this is the definitive runtime signal
   that FCM's HTTP v1 endpoint actually accepted the message
   (`push.go:466-534`, `PushSender.Send`).

---

## Payments (Stripe)

**Real implementation:**
`services/billing-service/internal/billing/stripe_provider.go` (real
`github.com/stripe/stripe-go/v86` client calls — subscription
create/update/cancel + webhook signature verification). Full detail:
[`docs/guides/BILLING.md`](../guides/BILLING.md).

### Environment variables

| Variable | Required | Read at | Purpose |
|---|---|---|---|
| `STRIPE_SECRET_KEY` | No — honest `501` when absent | `env.go:41` (`NewProviderFromEnv`, constant `EnvStripeSecretKey` at `env.go:13`) | Stripe API secret key (`sk_test_...` / `sk_live_...`). |
| `STRIPE_WEBHOOK_SECRET` | No — webhook verification fails closed when absent | `env.go:45` (constant `EnvStripeWebhookSecret` at `env.go:14`) | Stripe webhook endpoint signing secret (`whsec_...`). |

### Which secret, where

Both are plain string secrets from the Stripe Dashboard, not files —
export them as shell environment variables, or place them in a local
file under `scripts/testing/secrets/` and `source` it before running the
service locally (see `scripts/testing/secrets/README.md`). Full
walkthrough (creating a Stripe test-mode account, copying the secret
key, creating a recurring Price, configuring the webhook endpoint):
[`docs/guides/BILLING.md` § "Stripe setup (test mode)"](../guides/BILLING.md#stripe-setup-test-mode).

### Honest default when unset

`STRIPE_SECRET_KEY` unset → `billing.NewProviderFromEnv()` returns
`(nil, nil)` — not an error. Every subscription-lifecycle-mutating
endpoint (`POST /api/v1/subscriptions`, `PUT .../:id`,
`POST .../:id/cancel`) honestly responds `501 Not Implemented` — never a
fabricated `"active"` status.

`STRIPE_WEBHOOK_SECRET` may be left empty even when `STRIPE_SECRET_KEY`
is set: subscription create/update/cancel work normally, but
`StripeProvider.VerifyWebhook` always fails closed and rejects every
inbound webhook payload (`stripe_provider.go:268-270`) — there is no
"verify without a secret" mode, since that would mean trusting an
unverified payload.

### How to know it activated

`internal/server.New` reads `STRIPE_SECRET_KEY` exactly once at startup
and logs which state the process is in (`server.go:85-92`):

```
billing-service: payments provider "stripe" configured — subscription lifecycle calls are REAL
```

or, honestly, when unset:

```
billing-service: no payments provider configured (STRIPE_SECRET_KEY unset) — subscription-lifecycle-mutating endpoints will respond 501 Not Implemented
```

Beyond the log line: a real `POST /api/v1/subscriptions` call returns
`201` with `status` set to the **real** Stripe-reported status (e.g.
`"active"`, `"incomplete"`, `"trialing"`) plus `provider` and
`externalSubscriptionId` in the response body — never a hardcoded
`"active"`.

---

## Email (SMTP)

**Real implementation:**
`services/notification-service/internal/delivery/email.go` — real
outbound SMTP delivery via the Go standard library `net/smtp` (no
third-party mail library).

### Environment variables

| Variable | Required | Read at | Default when unset (if the tier is otherwise enabled) |
|---|---|---|---|
| `SMTP_HOST` | Yes, to enable email | `email.go:55` (`SMTPConfigFromEnv`) | — (unset ⇒ tier disabled entirely) |
| `SMTP_PORT` | No | `email.go:59` | `"25"` |
| `SMTP_FROM` | No | `email.go:63` | `"notifications@localhost"` |
| `SMTP_USERNAME` | No | `email.go:71` | empty (PLAIN auth is used only when a username is set — MailHog/Mailpit-class dev sinks need no auth) |
| `SMTP_PASSWORD` | No | `email.go:72` | empty |

### Which secret, where

`SMTP_USERNAME` / `SMTP_PASSWORD` are the credentials your SMTP relay
issues (e.g. an SMTP relay API key from SendGrid/Mailgun/Postmark, or
your own mail server's account). For local development, point
`SMTP_HOST` at a local dev sink (Mailhog, Mailpit, or equivalent) that
requires no auth and leave `SMTP_USERNAME` / `SMTP_PASSWORD` unset.
Store real relay credentials the same way as Stripe's above — a shell
export, or a local file under `scripts/testing/secrets/`.

### Honest default when unset

`SMTP_HOST` unset → `notification-service`'s `Handler.emailSender` stays
`nil` (`handler.go:40-42` never sets it). Every email notification's
`status` is honestly set to `"failed"` (`handler.go:203-205`) — no
delivery attempt is made, and the status is never fabricated as `"sent"`.

### How to know it activated

**There is no startup log line for SMTP** (unlike push and Stripe above
— this is the real current behaviour of the code, stated honestly
rather than invented). The only observable signal is runtime: create an
email-channel notification and inspect its `status`. `"sent"`
(`handler.go:211-213`, set only after the configured SMTP server
actually accepted the message) means SMTP is genuinely configured and
working; `"failed"` means either it is unconfigured (`SMTP_HOST` unset)
or a real delivery error occurred (both cases collapse to the same
`"failed"` status — the two are distinguished at the source-log level
via the error `smtp not configured: SMTP_HOST is unset` from
`email.go:133`, which is returned to the caller but not itself logged
by this handler; a future improvement could split the two).

---

## AI / LLM

**Real implementation:**
`services/ai-service/cmd/ai-service/main.go` +
`services/ai-service/internal/llmclient/llmclient.go` — a real HTTP
client (via the `vasic-digital/LLMProvider` generic OpenAI-compatible
adapter) that calls a **local** llama.cpp-compatible LLM server. Full
detail: [`docs/guides/AI_SERVICE.md`](../guides/AI_SERVICE.md).

### Environment variables — local tier (the only tier that exists)

| Variable | Required | Read at | Default when unset |
|---|---|---|---|
| `AI_LOCAL_PROVIDER_BASE_URL` | No | `main.go:90` | `http://127.0.0.1:18434/v1/chat/completions` |
| `AI_LOCAL_PROVIDER_MODEL` | No | `main.go:94` | `qwen2.5-1.5b-instruct` |

Both already have working code-level defaults matching the project's
local HelixLLM llama.cpp smoke-test convention — neither variable is
required for `ai-service` to start and serve real completions against a
local model server.

### Honest statement: there is NO cloud-LLM tier to configure

Unlike push/Stripe/email above, **there is no second, cloud-hosted LLM
tier already implemented that merely lacks operator config.** Per
Constitution §11.4.6 (no-guessing — "if a tier's var can't be found,
say so honestly rather than invent"), a repo-wide search for
`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or any other cloud-provider
credential inside `services/ai-service` found **no matches**. `main.go`
states this explicitly:

> "Local HelixLLM tier only — the cloud tier (OpenAI/Anthropic API
> keys) is OPERATOR-BLOCKED and deliberately not wired" (`main.go:98-99`)

`llmclient.go`'s package doc comment repeats the same fact
(`llmclient.go:5-7`), and
[`docs/guides/AI_SERVICE.md` § "Local-only posture — why the cloud tier
is off"](../guides/AI_SERVICE.md#local-only-posture--why-the-cloud-tier-is-off)
documents it in full, including that the underlying `LLMProvider`
library the local client is built on ships first-class `openai` /
`anthropic` / `claude` adapters that are simply never invoked here. See
also `docs/CONTINUATION.md` for the tracked follow-up item.

**Consequence for this document:** there is no `CLOUD_LLM_*` /
`OPENAI_API_KEY` / `ANTHROPIC_API_KEY` variable added to
`infrastructure/docker/compose/.env.example` — adding one would invent
a value the code does not read, which is exactly the fabrication this
project's anti-bluff covenant forbids. When a cloud tier is
implemented, its operator-config additions belong in a follow-up to
this document (and to `.env.example`), not here.

### How to know the local tier activated

`ai-service` always logs which local LLM endpoint it is actually using,
whether default or overridden (`main.go:101`):

```
ai-service: local LLM provider base_url=<url> model=<model>
```

This confirms which endpoint the process will call — it does not by
itself confirm the endpoint is reachable; a real `POST` to the
service's request-creation endpoint against a running local model
server is the positive end-to-end confirmation (see
`docs/guides/AI_SERVICE.md` for the full request/response contract and
failure-mode table).

---

## See also

- [`docs/guides/BILLING.md`](../guides/BILLING.md) — full Stripe
  integration design, API changes, webhook configuration, and test
  suite instructions.
- [`docs/guides/AI_SERVICE.md`](../guides/AI_SERVICE.md) — full
  local-LLM integration design and the local-only posture rationale.
- `scripts/firebase/firebase_setup.sh` — FCM/APNs service-account
  provisioning automation (`--help` for full usage).
- [`scripts/testing/secrets/README.md`](../../scripts/testing/secrets/README.md)
  — where to stage local credential files safely (gitignored).
- [`infrastructure/docker/compose/.env.example`](../../infrastructure/docker/compose/.env.example)
  — the tracked placeholder template these variables are documented
  against.
- `docs/CONTINUATION.md` — live project state, including the tracked
  cloud-LLM and compose-passthrough follow-up items.
