# Feature-tier credential setup

**Revision:** 1
**Last modified:** 2026-07-22T11:44:33Z

Operator checklist for the four credential-gated feature tiers. Each tier is
**seam-scaffolded**: the code reads its credentials from the environment
(Constitution §11.4.10 — never hardcoded, never logged) and, when they are
absent, runs an **honest dev default** rather than a fabricated-active state
(Constitution §11.4 anti-bluff covenant). Supplying a credential ARMS the tier;
arming performs **no** external API call at startup.

All values are placeholders in
`infrastructure/docker/compose/.env.example`. Copy that file to the gitignored
`.env` and fill only the tiers you enable. Secret **files** (the FCM
service-account JSON, the APNs `.p8` key) live under
`scripts/testing/secrets/` (git-ignored) and are referenced by PATH; opaque
string secrets go directly in `.env`. NO real secret is committed.

## How activation works, per tier

### 1. Billing — Stripe (`billing-service`)

| Item | Value |
|---|---|
| Secret(s) | `STRIPE_SECRET_KEY` (arms the gateway), `STRIPE_WEBHOOK_SECRET` (optional) |
| Stored in | gitignored `.env` (opaque strings) |
| Reads it | `internal/payment.StripeConfigFromEnv` → `payment.NewGateway` in `cmd/billing-service/main.go` |
| Activation | `STRIPE_SECRET_KEY` present ⇒ gateway `Enabled()`. With `STRIPE_WEBHOOK_SECRET` also set ⇒ `WebhookVerificationReady()`. Startup logs `payment mode=…` (a log-safe state word, never the secret). |
| Honest default when unset | Gateway **disabled** — `mode=disabled (no STRIPE_SECRET_KEY — internal dev default)`. billing-service runs with no live payment provider. |
| Honest boundary | Arming makes **no** Stripe API call. A real charge/refund/webbook-verify client is a future change; the seam only records that credentials are present. |

### 2. Cloud LLM — OpenAI **or** Anthropic (`ai-service`)

| Item | Value |
|---|---|
| Secret(s) | `OPENAI_API_KEY` **or** `ANTHROPIC_API_KEY` (set ONE) |
| Optional | `AI_CLOUD_MODEL`, `AI_CLOUD_BASE_URL`, `LLM_MONTHLY_COST_CEILING_USD` |
| Stored in | gitignored `.env` (opaque strings) |
| Reads it | provider switch in `cmd/ai-service/main.go`, reusing `internal/llmclient.NewGenericClient` (one generic OpenAI-compatible adapter for all tiers) |
| Activation | `OPENAI_API_KEY` set ⇒ **openai** tier (default model `gpt-4o-mini` @ `https://api.openai.com/v1/chat/completions`). Else `ANTHROPIC_API_KEY` set ⇒ **anthropic** tier (default model `claude-3-5-haiku-latest` @ `https://api.anthropic.com/v1/chat/completions`, its OpenAI-compatible endpoint). `AI_CLOUD_MODEL` / `AI_CLOUD_BASE_URL` override the per-provider defaults. Startup logs `cloud LLM provider=…`. |
| Honest default when unset | **Local HelixLLM** tier — `AI_LOCAL_PROVIDER_BASE_URL` / `AI_LOCAL_PROVIDER_MODEL` (defaults `http://127.0.0.1:18434/v1/chat/completions`, `qwen2.5-1.5b-instruct`). |
| Honest boundary | `LLM_MONTHLY_COST_CEILING_USD` is parsed, validated (non-negative), and **logged for visibility, NOT enforced** — no request path meters spend against it yet. Selecting a tier makes no LLM call at startup; the first real completion runs at request time through the handler. |

### 3. Push — FCM **or** APNs (`notification-service`)

| Item | Value |
|---|---|
| Secret(s) — FCM | `FCM_SERVICE_ACCOUNT_JSON` (path to the HTTP v1 service-account file) **or** `FCM_SERVER_KEY` (legacy string) |
| Secret(s) — APNs | ALL of `APNS_KEY_PATH` (path to the `.p8`) + `APNS_KEY_ID` + `APNS_TEAM_ID` + `APNS_BUNDLE_ID` |
| Stored in | file secrets under `scripts/testing/secrets/` (git-ignored); identifier strings in `.env` |
| Reads it | `internal/delivery.PushConfigFromEnv` → `delivery.NewPushSenderWithConfig`, wired in `handler.New` |
| Activation | A **complete** FCM set OR a **complete** APNs set ⇒ the push sender is armed (FCM takes precedence if both present). A partial set is treated as **not configured** (never half-armed). |
| Honest default when unset | Unconfigured push sender (`NewPushSender()`); `Send()` returns `ErrPushProviderNotConfigured`; the handler persists `pending_provider_unconfigured`. |
| **Honest boundary (important)** | Even when credentials ARE detected, the real FCM HTTP v1 / APNs HTTP/2 delivery client is **not yet implemented**. An armed sender's `Send()` returns `ErrPushProviderNotImplemented` — **no push is ever actually sent**, and the handler still persists `pending_provider_unconfigured`. Detecting credentials is NEVER reported as a delivered push (Constitution §11.4 anti-bluff). Building the real delivery client is a tracked follow-up. |

### 4. Email — SMTP (`notification-service`)

| Item | Value |
|---|---|
| Secret(s) | `SMTP_HOST` (enables the tier), `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD` |
| Stored in | gitignored `.env` |
| Reads it | `internal/delivery.SMTPConfigFromEnv` → `delivery.NewEmailSender`, wired in `handler.New` |
| Activation | `SMTP_HOST` present ⇒ real SMTP delivery via the Go stdlib `net/smtp`. PLAIN auth is used only when `SMTP_USERNAME` is set (MailHog/Mailpit-class dev sinks need no auth). |
| Honest default when unset | Email honestly reported as not configured — the handler sets `failed` rather than fabricating a `sent` status. |
| Honest boundary | Success means the configured SMTP server **accepted** the message, not that the recipient received it — mailbox/sink confirmation comes from the integration tests' real MailHog/Mailpit inbox. This tier (unlike push) delivers for real. |

## Anti-bluff summary

- No secret is committed — every slot in `.env.example` is a REDACTED,
  commented placeholder; file secrets are git-ignored under
  `scripts/testing/secrets/`.
- Every "enabled" state is driven by real credential presence read from the
  environment; no tier silently self-activates.
- Two deliberate honest boundaries: **push delivery** detects credentials but
  does not yet send (returns `ErrPushProviderNotImplemented`), and the
  **cloud-LLM cost ceiling** is surfaced/logged but not enforced. Both are
  logged/documented, never claimed as working.
