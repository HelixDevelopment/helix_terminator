#!/usr/bin/env bash
# firebase_setup.sh
#
# Purpose:
#   Idempotent, fully-dynamic Firebase project setup for this repo (or any
#   consuming project — see "Reusability" below). Uses the Firebase CLI to
#   select-or-create the Firebase project, select-or-create platform apps
#   (Android/iOS/Web) when their identifiers are provided, download each
#   app's client SDK config, and — where a full automation path exists —
#   generate the FCM (Firebase Cloud Messaging) HTTP v1 service-account JSON
#   the notification-service Go delivery client
#   (services/notification-service/internal/delivery/push.go) reads for
#   real push delivery.
#
#   This IS the regeneration mechanism for every secret/config file it
#   produces (Constitution §11.4.77) — nothing it writes is hand-crafted;
#   every output is reproducible by re-running this script.
#
# Usage:
#   bash scripts/firebase/firebase_setup.sh
#
#   Safe by default: selecting an EXISTING project/app is always performed;
#   CREATING a new project or new platform apps requires an explicit opt-in
#   env var (see Inputs) so this script never surprise-mutates an
#   operator's real Google Cloud account on a bare invocation.
#
# Inputs (environment variables — Constitution §11.4.28: project id + app
#   ids are ALWAYS parameterised via env, NEVER hardcoded in this script):
#   FIREBASE_PROJECT_ID            Firebase/GCP project id to target.
#                                   Default: this repo's directory name,
#                                   lowercased with '_' -> '-' (Firebase
#                                   project ids disallow underscores) —
#                                   e.g. "helix_terminator" -> "helix-terminator".
#                                   Mirrors the §11.4.151 prefix-resolution
#                                   pattern (derive from repo root, never a
#                                   literal baked into source).
#   FIREBASE_PROJECT_DISPLAY_NAME  Display name if the project is created.
#                                   Default: same as FIREBASE_PROJECT_ID.
#   FIREBASE_ALLOW_CREATE_PROJECT  Set to "1" to allow this script to CREATE
#                                   the project via `firebase projects:create`
#                                   when it does not already exist. Default:
#                                   unset (select-only — SKIPs with guidance
#                                   if the project is missing).
#   FIREBASE_ALLOW_CREATE_APPS     Set to "1" to allow this script to CREATE
#                                   Android/iOS/Web apps via `firebase
#                                   apps:create` when the identifiers below
#                                   are set and no matching app exists yet.
#                                   Default: unset.
#   FIREBASE_ANDROID_PACKAGE       Android applicationId (e.g.
#                                   "com.example.app"). Optional — when set
#                                   (and FIREBASE_ALLOW_CREATE_APPS=1) an
#                                   Android app is selected-or-created.
#   FIREBASE_IOS_BUNDLE_ID         iOS bundle id. Optional, same gating as
#                                   FIREBASE_ANDROID_PACKAGE.
#   FIREBASE_WEB_APP_NAME          Web app display name. Optional, same
#                                   gating as FIREBASE_ANDROID_PACKAGE.
#   FIREBASE_SETUP_OUTPUT_DIR      Directory every secret/config file is
#                                   written to. Default:
#                                   scripts/firebase/secrets/ (GITIGNORED —
#                                   see .gitignore; verified by this
#                                   script's own preflight check, §11.4.10).
#   FIREBASE_SERVICE_ACCOUNT_KEY_PATH
#                                   Override the FCM service-account JSON
#                                   output path. Default:
#                                   <output-dir>/fcm-service-account.json.
#   FIREBASE_TOKEN                 (Read-only, never printed.) If your
#                                   environment authenticates the CLI via a
#                                   CI token (`firebase login:ci` output)
#                                   rather than an interactive `firebase
#                                   login`, export FIREBASE_TOKEN before
#                                   running this script — the Firebase CLI
#                                   reads it automatically. This script
#                                   never reads, prints, or stores its
#                                   value; it only checks whether the CLI
#                                   ends up authenticated (via
#                                   `firebase projects:list`).
#
# Outputs (all under FIREBASE_SETUP_OUTPUT_DIR, gitignored, chmod 600):
#   fcm-service-account.json       FCM HTTP v1 service-account key (only
#                                   when a scriptable generation path is
#                                   available — see "FCM service-account key"
#                                   below for the honest-SKIP behaviour when
#                                   it is not).
#   <platform>-<appId>.json / .plist
#                                   Client SDK config per registered app
#                                   (google-services.json for Android,
#                                   GoogleService-Info.plist for iOS, a JSON
#                                   config object for Web), only for apps
#                                   this script selected or created.
#
# Side-effects:
#   With FIREBASE_ALLOW_CREATE_PROJECT=1 and the target project absent:
#     creates a REAL Google Cloud Platform project + adds Firebase
#     resources to it (`firebase projects:create`). Reversible (the project
#     can be deleted from the Firebase/GCP console or via
#     `firebase projects:delete`); no billing account is required for the
#     Spark (free) plan, which is sufficient for Cloud Messaging.
#   With FIREBASE_ALLOW_CREATE_APPS=1 and a platform identifier set:
#     registers a REAL platform app under the project (`firebase apps:create`).
#   FCM service-account key generation (when gcloud is available — see
#     below) creates a REAL new IAM key for the project's existing
#     auto-provisioned "firebase-adminsdk-*" service account
#     (`gcloud iam service-accounts keys create`). This does NOT create a
#     new IAM principal or grant any new role — it only mints an additional
#     credential for an account Firebase already provisioned with Cloud
#     Messaging send permission, and the key is revocable at any time from
#     the Google Cloud Console.
#
# Dependencies:
#   firebase (Firebase CLI, https://firebase.google.com/docs/cli — this
#     script hard-fails with install guidance if absent), jq (JSON
#     parsing of Firebase CLI --json output), bash, coreutils. gcloud
#     (Google Cloud SDK) is OPTIONAL — required only for the FCM
#     service-account key generation step; its absence is an honest,
#     clearly-reported SKIP with an exact remediation path, never a
#     silent no-op or a fabricated success.
#
# Reusability (Constitution §11.4.28 — decoupled, project-agnostic):
#   Every project-identifying value is environment-sourced with a
#   dynamically-derived (never hardcoded literal) default. Any consuming
#   project can invoke this script unmodified.
#
# Cross-references:
#   docs/scripts/firebase_setup.md   companion user guide (§11.4.18)
#   services/notification-service/internal/delivery/push.go
#     reads FCM_SERVICE_ACCOUNT_JSON (point it at this script's output) to
#     perform real FCM HTTP v1 + APNs-via-FCM delivery.
#   .gitignore                       scripts/firebase/secrets/ pattern
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." >/dev/null 2>&1 && pwd)"

log() { echo "[firebase_setup] $*"; }
err() { echo "[firebase_setup] ERROR: $*" >&2; }
warn() { echo "[firebase_setup] WARN: $*" >&2; }

# ---------------------------------------------------------------------------
# Dynamic, never-hardcoded project id default (§11.4.28 / §11.4.151 pattern):
# derive from THIS repo's own directory name, not a literal baked into the
# script, so the same script works unmodified for any consuming project.
# ---------------------------------------------------------------------------
DEFAULT_PROJECT_ID="$(basename "$REPO_ROOT" | tr '[:upper:]' '[:lower:]' | tr '_' '-')"
PROJECT_ID="${FIREBASE_PROJECT_ID:-$DEFAULT_PROJECT_ID}"
PROJECT_DISPLAY_NAME="${FIREBASE_PROJECT_DISPLAY_NAME:-$PROJECT_ID}"
ALLOW_CREATE_PROJECT="${FIREBASE_ALLOW_CREATE_PROJECT:-0}"
ALLOW_CREATE_APPS="${FIREBASE_ALLOW_CREATE_APPS:-0}"
ANDROID_PACKAGE="${FIREBASE_ANDROID_PACKAGE:-}"
IOS_BUNDLE_ID="${FIREBASE_IOS_BUNDLE_ID:-}"
WEB_APP_NAME="${FIREBASE_WEB_APP_NAME:-}"
OUTPUT_DIR="${FIREBASE_SETUP_OUTPUT_DIR:-$SCRIPT_DIR/secrets}"
SERVICE_ACCOUNT_KEY_PATH="${FIREBASE_SERVICE_ACCOUNT_KEY_PATH:-$OUTPUT_DIR/fcm-service-account.json}"

CREATED_PROJECT=0
PROJECT_ALREADY_EXISTED=0
SA_KEY_STATUS="skipped"
APPS_SUMMARY=()

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------
if ! command -v firebase >/dev/null 2>&1; then
  err "Firebase CLI not found on PATH."
  err "Install: npm install -g firebase-tools   (https://firebase.google.com/docs/cli)"
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  err "jq not found on PATH (required to parse 'firebase ... --json' output)."
  err "Install jq via your package manager, then re-run."
  exit 1
fi

# §11.4.10 / §11.4.10.A: verify the secret output directory is actually
# gitignored BEFORE writing anything into it — refuse to proceed rather
# than risk a credential landing in a trackable path.
mkdir -p "$OUTPUT_DIR"
chmod 700 "$OUTPUT_DIR"
PROBE_FILE="$OUTPUT_DIR/.gitignore_probe"
: > "$PROBE_FILE"
if git -C "$REPO_ROOT" check-ignore -q "$PROBE_FILE" 2>/dev/null; then
  rm -f "$PROBE_FILE"
else
  rm -f "$PROBE_FILE"
  err "$OUTPUT_DIR is NOT covered by .gitignore — refusing to write any credential there."
  err "Add a pattern covering it to $REPO_ROOT/.gitignore (e.g. 'scripts/firebase/secrets/') and re-run."
  exit 1
fi

# ---------------------------------------------------------------------------
# Auth check — doubles as the project-list probe. Never prints tokens.
# ---------------------------------------------------------------------------
log "Checking Firebase CLI authentication..."
# IMPORTANT: capture stdout and stderr SEPARATELY. The Firebase CLI writes
# its human-readable progress spinner to stderr even with --json — merging
# streams (2>&1) into the captured variable corrupts the JSON payload and
# makes every downstream `jq` parse fail even though the call itself
# succeeded. (Forensic note: this exact bug was caught and fixed during
# this script's own first live run, 2026-07-22 — captured here so it is
# never reintroduced.)
FB_AUTH_STDERR="$(mktemp)"
trap 'rm -f "$FB_AUTH_STDERR"' EXIT
if ! PROJECTS_JSON="$(firebase projects:list --json 2>"$FB_AUTH_STDERR")"; then
  err "Firebase CLI is not authenticated (or the auth check failed)."
  err "CLI stderr output (no secrets are ever included in this message):"
  sed 's/^/[firebase_setup]   /' "$FB_AUTH_STDERR" >&2
  err "Remediation: run 'firebase login' interactively, OR in a non-interactive"
  err "environment generate a CI token with 'firebase login:ci' and export it as"
  err "FIREBASE_TOKEN before re-running this script."
  exit 1
fi
if ! echo "$PROJECTS_JSON" | jq -e '.status == "success"' >/dev/null 2>&1; then
  err "Firebase CLI auth check did not report success."
  exit 1
fi
log "Firebase CLI is authenticated."

# ---------------------------------------------------------------------------
# Step 1: select-or-create the project
# ---------------------------------------------------------------------------
EXISTING_PROJECT="$(echo "$PROJECTS_JSON" | jq -r --arg id "$PROJECT_ID" '.result[] | select(.projectId == $id) | .projectId')"
if [ -n "$EXISTING_PROJECT" ]; then
  log "Project '$PROJECT_ID' already exists — selecting it (idempotent no-op)."
  PROJECT_ALREADY_EXISTED=1
else
  if [ "$ALLOW_CREATE_PROJECT" = "1" ]; then
    log "Project '$PROJECT_ID' not found — creating it (display name: '$PROJECT_DISPLAY_NAME')..."
    firebase projects:create "$PROJECT_ID" --display-name "$PROJECT_DISPLAY_NAME" --non-interactive
    CREATED_PROJECT=1
    log "Project '$PROJECT_ID' created."
  else
    err "Project '$PROJECT_ID' does not exist and FIREBASE_ALLOW_CREATE_PROJECT is not set to 1."
    err "SKIPPING project creation (safe default — this script never creates cloud"
    err "resources implicitly)."
    err "To create it: FIREBASE_ALLOW_CREATE_PROJECT=1 bash scripts/firebase/firebase_setup.sh"
    exit 1
  fi
fi

# ---------------------------------------------------------------------------
# Step 2: select-or-create platform apps (only for identifiers provided)
# ---------------------------------------------------------------------------
select_or_create_app() {
  local platform="$1" identifier="$2" filter_field="$3"
  local existing_app_id
  existing_app_id="$(firebase apps:list "$platform" -P "$PROJECT_ID" --json 2>/dev/null \
    | jq -r --arg id "$identifier" --arg field "$filter_field" '.result[] | select(.[$field] == $id) | .appId' \
    | head -1)"
  if [ -n "$existing_app_id" ]; then
    log "$platform app '$identifier' already exists (appId=$existing_app_id) — selecting it."
    echo "$existing_app_id"
    return 0
  fi
  if [ "$ALLOW_CREATE_APPS" != "1" ]; then
    warn "$platform app '$identifier' not found and FIREBASE_ALLOW_CREATE_APPS is not set to 1 — SKIPPING creation."
    return 1
  fi
  log "Creating $platform app '$identifier'..."
  local created_json
  case "$platform" in
    ANDROID) created_json="$(firebase apps:create ANDROID "$identifier" -a "$identifier" -P "$PROJECT_ID" --json)" ;;
    IOS)     created_json="$(firebase apps:create IOS "$identifier" -b "$identifier" -P "$PROJECT_ID" --json)" ;;
    WEB)     created_json="$(firebase apps:create WEB "$identifier" -P "$PROJECT_ID" --json)" ;;
    *) err "unknown platform $platform"; return 1 ;;
  esac
  local new_app_id
  new_app_id="$(echo "$created_json" | jq -r '.result.appId // empty')"
  if [ -z "$new_app_id" ]; then
    err "Failed to determine the new appId from 'firebase apps:create' output for $platform app '$identifier'."
    return 1
  fi
  log "$platform app '$identifier' created (appId=$new_app_id)."
  echo "$new_app_id"
}

download_sdkconfig() {
  local platform="$1" app_id="$2" out_basename="$3"
  local ext="json"
  [ "$platform" = "IOS" ] && ext="plist"
  local out_path="$OUTPUT_DIR/${out_basename}.${ext}"
  if firebase apps:sdkconfig "$platform" "$app_id" -P "$PROJECT_ID" -o "$out_path" >/dev/null 2>&1; then
    chmod 600 "$out_path"
    log "Downloaded $platform SDK config -> $out_path"
    APPS_SUMMARY+=("$platform appId=$app_id config=$out_path")
  else
    warn "Failed to download SDK config for $platform app $app_id."
  fi
}

if [ -n "$ANDROID_PACKAGE" ]; then
  if app_id="$(select_or_create_app ANDROID "$ANDROID_PACKAGE" packageName)"; then
    download_sdkconfig ANDROID "$app_id" "android-${ANDROID_PACKAGE}"
  fi
else
  log "FIREBASE_ANDROID_PACKAGE not set — skipping Android app (no package name to register; this is a genuine operator decision, not guessed — see docs/scripts/firebase_setup.md)."
fi

if [ -n "$IOS_BUNDLE_ID" ]; then
  if app_id="$(select_or_create_app IOS "$IOS_BUNDLE_ID" bundleId)"; then
    download_sdkconfig IOS "$app_id" "ios-${IOS_BUNDLE_ID}"
  fi
else
  log "FIREBASE_IOS_BUNDLE_ID not set — skipping iOS app (no bundle id to register)."
fi

if [ -n "$WEB_APP_NAME" ]; then
  if app_id="$(select_or_create_app WEB "$WEB_APP_NAME" displayName)"; then
    download_sdkconfig WEB "$app_id" "web-${WEB_APP_NAME}"
  fi
else
  log "FIREBASE_WEB_APP_NAME not set — skipping Web app (no app name to register)."
fi

# ---------------------------------------------------------------------------
# Step 3: FCM HTTP v1 service-account JSON
#
# The Firebase CLI has NO command to generate/download a service-account
# key (verified 2026-07-22 against `firebase --help`'s full command list —
# `apps:*` covers client SDK config only, `login:ci` issues a CI auth
# token, not a service-account key). The two genuine paths are:
#   (A) gcloud (Google Cloud SDK): `gcloud iam service-accounts keys create`
#       against the project's auto-provisioned "firebase-adminsdk-*"
#       service account, which Firebase already grants Cloud Messaging
#       send permission (cloudmessaging.messages.create) by default.
#   (B) Firebase Console (manual, UI-only, not scriptable):
#       Project Settings > Service accounts > Generate New Private Key.
# ---------------------------------------------------------------------------
if [ -f "$SERVICE_ACCOUNT_KEY_PATH" ]; then
  log "FCM service-account JSON already present at $SERVICE_ACCOUNT_KEY_PATH (idempotent no-op)."
  log "Delete it and re-run to force regeneration."
  SA_KEY_STATUS="already_present"
elif ! command -v gcloud >/dev/null 2>&1; then
  SA_KEY_STATUS="skipped_no_gcloud"
  warn "gcloud (Google Cloud SDK) not found on PATH — cannot automate FCM service-account key generation."
  warn "BLOCKER: the Firebase CLI has no equivalent command; a service-account JSON"
  warn "can only be produced via gcloud or the Firebase Console."
  warn ""
  warn "Remediation option A (no install required):"
  warn "  1. Open: https://console.firebase.google.com/project/$PROJECT_ID/settings/serviceaccounts/adminsdk"
  warn "  2. Click 'Generate New Private Key', confirm, and save the downloaded file to:"
  warn "     $SERVICE_ACCOUNT_KEY_PATH"
  warn "  3. chmod 600 '$SERVICE_ACCOUNT_KEY_PATH'"
  warn ""
  warn "Remediation option B (enables full automation on future runs):"
  warn "  1. Install gcloud: https://cloud.google.com/sdk/docs/install"
  warn "  2. gcloud auth login"
  warn "  3. Re-run this script — it will detect gcloud and generate the key automatically."
else
  log "gcloud detected — locating the auto-provisioned firebase-adminsdk service account for '$PROJECT_ID'..."
  SA_EMAIL="$(gcloud iam service-accounts list --project="$PROJECT_ID" --format=json 2>/dev/null \
    | jq -r '.[] | select(.email | startswith("firebase-adminsdk")) | .email' | head -1)"
  if [ -z "$SA_EMAIL" ]; then
    SA_KEY_STATUS="skipped_no_service_account_found"
    warn "No firebase-adminsdk-* service account found for project '$PROJECT_ID' via gcloud."
    warn "This usually means gcloud is not authenticated for this project ('gcloud auth login'"
    warn "then 'gcloud config set project $PROJECT_ID'), or the project is too new for the Admin"
    warn "SDK service account to have provisioned yet (retry in a minute). Falling back to the"
    warn "Firebase Console path — see the gcloud-absent guidance above for the exact URL."
  else
    log "Found service account: $SA_EMAIL — creating a new key..."
    if gcloud iam service-accounts keys create "$SERVICE_ACCOUNT_KEY_PATH" \
        --iam-account="$SA_EMAIL" --project="$PROJECT_ID" >/dev/null 2>&1; then
      chmod 600 "$SERVICE_ACCOUNT_KEY_PATH"
      SA_KEY_STATUS="created"
      log "FCM service-account JSON written to $SERVICE_ACCOUNT_KEY_PATH (mode 600)."
    else
      SA_KEY_STATUS="failed"
      err "gcloud iam service-accounts keys create failed. Re-run with 'gcloud' output visible"
      err "(remove the redirect in this script, or run the gcloud command directly) to diagnose."
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Summary (project id / app ids / file paths only — NEVER secret content,
# Constitution §11.4.10)
# ---------------------------------------------------------------------------
echo
log "=== Summary ==="
log "Project ID:            $PROJECT_ID"
log "Project already existed: $([ "$PROJECT_ALREADY_EXISTED" = "1" ] && echo yes || echo no)"
log "Project created now:     $([ "$CREATED_PROJECT" = "1" ] && echo yes || echo no)"
log "FCM service-account key: $SA_KEY_STATUS"
if [ "$SA_KEY_STATUS" = "created" ] || [ "$SA_KEY_STATUS" = "already_present" ]; then
  log "  -> export FCM_SERVICE_ACCOUNT_JSON=$SERVICE_ACCOUNT_KEY_PATH"
fi
if [ "${#APPS_SUMMARY[@]}" -gt 0 ]; then
  log "Apps configured:"
  for line in "${APPS_SUMMARY[@]}"; do
    log "  - $line"
  done
else
  log "Apps configured:        none (no FIREBASE_*_PACKAGE/BUNDLE_ID/APP_NAME set)"
fi
log "Output directory:       $OUTPUT_DIR (gitignored)"
log "Done."
