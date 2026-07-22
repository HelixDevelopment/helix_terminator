# firebase_setup.sh

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## Overview

Idempotent, fully-dynamic Firebase project setup for `helix_terminator` (or
any consuming project — the script is decoupled/project-agnostic per
Constitution §11.4.28). It uses the real Firebase CLI to:

1. Select-or-create the target Firebase project.
2. Select-or-create Android / iOS / Web apps under that project (only when
   the operator supplies a package name / bundle id / app name — this
   script never invents one, per §11.4.6 no-guessing).
3. Download each configured app's client SDK config
   (`google-services.json`, `GoogleService-Info.plist`, or a Web config
   JSON) into a gitignored secrets directory.
4. Generate the FCM (Firebase Cloud Messaging) HTTP v1 service-account JSON
   that `services/notification-service/internal/delivery/push.go` reads
   for real push delivery — **when a scriptable path exists** (see
   "FCM service-account key" below for the honest limitation).

This script **is** the regeneration mechanism (§11.4.77) for every
credential/config file it produces — nothing under
`scripts/firebase/secrets/` is meant to be hand-crafted or committed;
everything there is reproducible by re-running this script.

## Prerequisites

- [Firebase CLI](https://firebase.google.com/docs/cli) (`npm install -g
  firebase-tools`), authenticated (`firebase login`, or `FIREBASE_TOKEN`
  set from `firebase login:ci` for non-interactive environments).
- `jq` (JSON parsing of Firebase CLI `--json` output).
- **Optional:** [`gcloud`](https://cloud.google.com/sdk/docs/install)
  (Google Cloud SDK), authenticated (`gcloud auth login`) — only required
  for the automated FCM service-account key generation step. Its absence
  does not block the rest of the script; it produces an honest SKIP with
  exact remediation instructions (see below).

## Usage

```bash
# Select the existing project (or fail with guidance if it doesn't exist):
bash scripts/firebase/firebase_setup.sh

# Create the project if it doesn't exist yet:
FIREBASE_ALLOW_CREATE_PROJECT=1 bash scripts/firebase/firebase_setup.sh

# Also register an Android app and download its config:
FIREBASE_ALLOW_CREATE_PROJECT=1 \
FIREBASE_ALLOW_CREATE_APPS=1 \
FIREBASE_ANDROID_PACKAGE=com.helixdevelopment.helix_terminator \
  bash scripts/firebase/firebase_setup.sh
```

### Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `FIREBASE_PROJECT_ID` | repo dirname, lowercased, `_`→`-` (e.g. `helix-terminator`) | Target Firebase/GCP project id. |
| `FIREBASE_PROJECT_DISPLAY_NAME` | same as `FIREBASE_PROJECT_ID` | Display name if the project is created. |
| `FIREBASE_ALLOW_CREATE_PROJECT` | unset (`0`) | Set `1` to allow creating the project when absent. |
| `FIREBASE_ALLOW_CREATE_APPS` | unset (`0`) | Set `1` to allow creating platform apps when absent. |
| `FIREBASE_ANDROID_PACKAGE` | unset | Android `applicationId`. Registers/selects an Android app when set. |
| `FIREBASE_IOS_BUNDLE_ID` | unset | iOS bundle id. Registers/selects an iOS app when set. |
| `FIREBASE_WEB_APP_NAME` | unset | Web app display name. Registers/selects a Web app when set. |
| `FIREBASE_SETUP_OUTPUT_DIR` | `scripts/firebase/secrets/` | Where every secret/config file is written. Verified gitignored before any write. |
| `FIREBASE_SERVICE_ACCOUNT_KEY_PATH` | `<output-dir>/fcm-service-account.json` | Override the FCM service-account JSON output path. |
| `FIREBASE_TOKEN` | unset | Non-interactive CLI auth token from `firebase login:ci`. Never read/printed/stored by this script — the Firebase CLI itself consumes it. |

**Safe by default.** Without the `*_ALLOW_CREATE_*` flags, the script only
*selects* existing resources and SKIPs with exact guidance when something
is missing — it never surprise-mutates a real Google Cloud account on a
bare invocation.

## Outputs

All under `FIREBASE_SETUP_OUTPUT_DIR` (default `scripts/firebase/secrets/`,
gitignored, directory mode `700`, files mode `600`):

- `fcm-service-account.json` — FCM HTTP v1 service-account key. Point
  `services/notification-service`'s `FCM_SERVICE_ACCOUNT_JSON` env var at
  this path to enable real push delivery.
- `<platform>-<identifier>.json` / `.plist` — client SDK config per
  registered app.

## FCM service-account key: the honest limitation

The Firebase CLI has **no command** to generate or download a
service-account key (verified 2026-07-22 against `firebase --help`'s full
command list — `apps:*` covers client SDK config only, `login:ci` issues a
CI auth token, not a service-account credential). There are exactly two
genuine paths to obtain one:

1. **`gcloud` (automated by this script when present):**
   `gcloud iam service-accounts keys create` against the project's
   auto-provisioned `firebase-adminsdk-*` service account — the same
   account Firebase's Console "Generate New Private Key" button targets.
   Firebase grants this account Cloud Messaging send permission
   (`cloudmessaging.messages.create`) automatically; the script does not
   create a new IAM principal or role binding, only an additional key for
   an account that already has the permission.
2. **Firebase Console (manual, UI-only, not scriptable):** Project
   Settings → Service accounts tab → **Generate New Private Key**. Console
   URL: `https://console.firebase.google.com/project/<PROJECT_ID>/settings/serviceaccounts/adminsdk`.

When `gcloud` is absent, the script prints both remediation paths verbatim
(with the exact Console URL and exact target file path) and exits with a
`skipped_no_gcloud` summary line — an honest SKIP (Constitution §11.4.3),
never a fabricated success.

## Edge cases / internal behaviour

- **stdout/stderr separation is load-bearing.** The Firebase CLI writes its
  human-readable progress spinner to **stderr** even when `--json` is
  passed. The script's first live run (2026-07-22) initially merged
  `2>&1` into the captured auth-check variable, which corrupted the JSON
  payload and made every downstream `jq` parse fail even though the
  underlying `firebase projects:list` call had actually succeeded. Fixed
  by capturing stdout and stderr into separate variables/files — this
  regression is now called out explicitly in the script's own comments so
  it is never reintroduced.
- **Gitignore preflight.** Before writing anything, the script creates a
  probe file under the output directory and verifies `git check-ignore`
  reports it as ignored — refusing to proceed (rather than writing a
  credential into a trackable path) if the directory is not covered by
  `.gitignore`.
- **Idempotency.** Re-running with the same inputs is a no-op for every
  already-satisfied step (existing project selected, existing apps
  selected, existing service-account key left untouched) — verified by two
  consecutive live runs during this script's initial validation.
- **App identifiers are never guessed.** With no `FIREBASE_ANDROID_PACKAGE`
  / `FIREBASE_IOS_BUNDLE_ID` / `FIREBASE_WEB_APP_NAME` set, the
  corresponding platform step SKIPs cleanly — inventing a placeholder
  package/bundle id would create a semi-permanent, hard-to-reverse
  App-Store-facing identifier without operator confirmation (Constitution
  §11.4.6 / §11.4.101).

## Related scripts

- `services/notification-service/internal/delivery/push.go` — the real FCM
  HTTP v1 + APNs-via-FCM delivery client that consumes this script's
  `fcm-service-account.json` output via `FCM_SERVICE_ACCOUNT_JSON`.
- `scripts/rotate-secrets.sh` — general secret-rotation entry point (this
  script's service-account key is one of the secret classes it should
  eventually cover — tracked as phase-2 work, see
  `docs/CONTINUATION.md`).

## Last verified

2026-07-22 — live-run verified twice against the real, authenticated
Firebase CLI (project `helix-terminator` selected on both runs; FCM
service-account step honestly SKIPPED with `gcloud` absent from the host).
