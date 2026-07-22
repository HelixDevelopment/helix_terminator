# Feature-tier secret files

This directory holds provider secret **files** that are referenced by PATH from
the gitignored `.env` (never inlined into an env var). It is committed only as
scaffolding — `.gitkeep`, this `README.md`, and any `*.example` templates are
the ONLY tracked files here. Every real secret placed here is git-ignored by the
feature-tier-credentials block in the repo-root `.gitignore`.

Constitution §11.4.10 (credentials-handling mandate) governs this directory:

- Real credentials MUST NEVER be committed. Verify with `git status` before any
  commit — a real secret showing as tracked is a release blocker.
- Restrict permissions: `chmod 700 scripts/testing/secrets` and `chmod 600` on
  each secret file.
- On a suspected leak, rotate the credential at the provider and scrub history.

## Files referenced by the feature tiers

| Env var (path)              | File placed here                     | Tier / provider                    |
|-----------------------------|--------------------------------------|------------------------------------|
| `FCM_SERVICE_ACCOUNT_JSON`  | `fcm_service_account.json`           | Push — Firebase Cloud Messaging v1 |
| `APNS_KEY_PATH`             | `apns_auth_key.p8`                   | Push — Apple Push Notification svc |

Secrets supplied as opaque strings (`STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`,
`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `FCM_SERVER_KEY`, `SMTP_PASSWORD`) go in
the gitignored `.env`, not here.

See `docs/credentials/FEATURE_TIER_SETUP.md` for the full per-tier operator
checklist and each tier's honest default when unset.
