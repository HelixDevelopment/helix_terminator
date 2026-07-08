# JWT Key Provisioning — auth-service

**Revision:** 2
**Last modified:** 2026-07-08T00:00:00Z

## Problem this closes (T15, production blocker)

`auth-service` used to generate a fresh, ephemeral Ed25519 keypair on
every process start (`crypto.NewJWTManager()` → `ed25519.GenerateKey`,
with a `// TODO: load from KMS or mounted secret in production` left in
`main.go`). A JWT it issued was only ever verifiable inside that one
process:

- `gateway-service` and `billing-service` independently validate bearer
  tokens against a `JWT_PUBLIC_KEY` they read from their own
  environment (`services/gateway-service/internal/server/server.go`,
  `services/billing-service/internal/server/server.go`) — with no key
  provisioned at all, they could never validate a real token.
- Even `auth-service` validating its own tokens broke across a simple
  pod restart, since the signing key changed every time.

Net effect: the entire JWT auth chain failed closed (401) in a real
deployment.

## The fix

`auth-service` now resolves its Ed25519 signing key via
`internal/server/server.go`'s `loadJWTManager`, in this order:

1. **`JWT_PRIVATE_KEY`** (base64, `encoding/base64` **standard**
   encoding, exactly 64 raw bytes = `ed25519.PrivateKeySize`) — the
   persisted, production key. If present, `auth-service` derives its
   public key from it. If a paired **`JWT_PUBLIC_KEY`** is *also* set,
   it is decoded the same way (32 raw bytes = `ed25519.PublicKeySize`,
   the same encoding `gateway-service`/`billing-service` already use)
   and MUST byte-for-byte match the public key derived from
   `JWT_PRIVATE_KEY` — a mismatched pair is a fail-closed configuration
   error (`auth-service` refuses to start), not a warning.
2. **`ENVIRONMENT=production` with no `JWT_PRIVATE_KEY`** — refuses to
   start with a clear, descriptive fatal error rather than silently
   falling back to a per-process ephemeral key nothing else could ever
   validate.
3. **Neither set** — dev/test fallback: generates a fresh ephemeral
   Ed25519 keypair exactly as before, but now logs a loud
   `WARNING: ephemeral JWT key — ...` line naming the exact
   consequence (tokens won't validate across restarts or against
   `gateway-service`/`billing-service`) and pointing back at this
   document. This is the path the existing test suite and any ad-hoc
   `go run ./cmd/auth-service` still take.

The token claim shape (`userId`/`orgId`/EdDSA signing) is unchanged.

Proof: `services/auth-service/internal/crypto/crypto_key_provisioning_test.go`
and `services/auth-service/internal/server/server_jwt_key_test.go`
exercise all four paths with real cryptographic verification (never
committing real key material — every key used in tests is generated
fresh, in-test, and discarded).

## Generating an Ed25519 keypair

Use the Go toolchain already required to build `auth-service` — this
prints base64 (standard encoding) private + public key material, the
exact format `crypto.NewJWTManagerFromKey` (auth-service) and the
`JWT_PUBLIC_KEY` decode path (`gateway-service`, `billing-service`)
expect:

```bash
cat <<'EOF' > /tmp/gen_jwt_keypair.go
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("JWT_PRIVATE_KEY=" + base64.StdEncoding.EncodeToString(priv))
	fmt.Println("JWT_PUBLIC_KEY=" + base64.StdEncoding.EncodeToString(pub))
}
EOF
go run /tmp/gen_jwt_keypair.go
rm /tmp/gen_jwt_keypair.go
```

Treat the `JWT_PRIVATE_KEY=...` line as a secret from the moment it is
printed: never paste it into a shell history file, ticket, chat
message, or any tracked file (Constitution §11.4.10). Pipe it directly
into `kubectl create secret` (below) or a secrets manager, then discard
the terminal scrollback.

## Creating the `helix-jwt-keys` Secret

`infrastructure/kubernetes/base/services/{auth,gateway,billing}-service/deployment.yaml`
reference a Kubernetes Secret named **`helix-jwt-keys`** (placeholder
name — no literal key value is committed in any manifest):

```bash
kubectl create secret generic helix-jwt-keys \
  --namespace helixterminator \
  --from-literal=JWT_PRIVATE_KEY='<value printed above>' \
  --from-literal=JWT_PUBLIC_KEY='<value printed above>'
```

| Env var | Consumed by | Purpose |
|---|---|---|
| `JWT_PRIVATE_KEY` | `auth-service` only | signs issued tokens |
| `JWT_PUBLIC_KEY` | `auth-service` (optional consistency check), `gateway-service`, `billing-service` | validates tokens |

`auth-service`'s deployment also sets `ENVIRONMENT=production` so that,
if the `helix-jwt-keys` Secret is ever missing or misconfigured, the
pod fails to start with a clear error instead of silently minting
ephemeral, unusable tokens. (In practice, Kubernetes itself already
refuses to start a container whose `secretKeyRef` points at a
nonexistent Secret/key — `ENVIRONMENT=production` is defense in depth
for any non-Kubernetes deployment target of this same binary.)

## Docker Compose

`infrastructure/docker/compose/docker-compose.yml` wires the same three
env vars for `auth-service` (`ENVIRONMENT`, `JWT_PRIVATE_KEY`,
`JWT_PUBLIC_KEY`) and `gateway-service` / `billing-service`
(`JWT_PUBLIC_KEY` only), each sourced via the compose file's existing
`${VAR:-default}` interpolation convention — never a literal value in
the tracked YAML.

**Default local bring-up (no `.env`, or a `.env` without these keys
set):** `ENVIRONMENT` defaults to `development` and the JWT vars default
to empty, so `auth-service` takes its loud-warned ephemeral-keypair dev
path (see "The fix" above) — the stack boots and is usable for solo
local development, but tokens will not validate across an `auth-service`
restart, nor against `gateway-service` / `billing-service` (they have no
public key to check against, either). This mirrors the compose file's
existing "keyless" default for other secrets (e.g. `GF_SECURITY_ADMIN_PASSWORD`).

**Real / shared compose deployment (staging-like, multiple developers,
or anything where JWT validation must actually work end-to-end):**

1. Generate a keypair with the snippet in "Generating an Ed25519
   keypair" above.
2. Create (or edit) `infrastructure/docker/compose/.env` — copy
   `infrastructure/docker/compose/.env.example` if you do not have one
   yet — and set:

   ```dotenv
   ENVIRONMENT=production
   JWT_PRIVATE_KEY=<value printed above>
   JWT_PUBLIC_KEY=<value printed above>
   ```

3. Bring the stack up (`docker compose up -d` /
   `podman-compose up -d`, per Constitution §11.4.161 rootless preferred)
   — `auth-service` now signs with the persisted key, refuses to start
   if `JWT_PRIVATE_KEY` is malformed or mismatched with a set
   `JWT_PUBLIC_KEY`, and `gateway-service` / `billing-service` validate
   against the same public key.

`infrastructure/docker/compose/.env` is **never committed** — it is
covered by the repo's root `.env` gitignore pattern (`.gitignore` line
`.env`, which matches at any depth including this directory) per
Constitution §11.4.30 / §11.4.10. Only `.env.example` (placeholders,
commented out) is tracked.

## Rotation

The JWT signing key is already listed in the 90-day rotation schedule
in `docs/runbooks/key-rotation.md`. Rotating it means: generate a new
keypair (above), update the `helix-jwt-keys` Secret, and roll
`auth-service` + `gateway-service` + `billing-service` together (a
window during which tokens signed by the old key still validate is
achievable by temporarily provisioning both public keys and unioning
them at the validation layer — not implemented today; see "Future
hardening" below).

## Future hardening — operator decision needed

A raw Secret-mounted private key (this fix) is a floor, not a ceiling.
Real hardening — signing via a cloud KMS asymmetric-sign API or a
HashiCorp Vault Transit engine, so the private key material never
exists in a pod's environment at all — is explicitly **not**
implemented by this fix. Per Constitution §11.4.101 (autonomous-safe
default now, defer high-blast-radius irreversible-integration choices)
and §11.4.112 (don't guess a vendor-specific integration nobody asked
for), which KMS/HSM provider to standardize on is an **operator
decision**, not something this fix should force. Track KMS integration
as a follow-up work item once that decision is made.
