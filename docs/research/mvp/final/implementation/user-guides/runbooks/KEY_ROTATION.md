# KEY_ROTATION.md

## 1. Objective

Rotate secrets and encryption keys used by helix_terminator services with minimal downtime. Covers both manual and automated rotation paths.

## 2. Scope

| Key / Secret | Rotation Owner | Method |
|--------------|----------------|--------|
| PostgreSQL application password | SRE-Data | Manual |
| Kafka SCRAM credentials | SRE-Data | Manual |
| Vault transit encryption key | Security | Automated (auto-unseal) |
| JWT Ed25519 signing key | Auth Team | Manual (dual-key) |
| API Gateway TLS key | SRE-Platform | Automated (cert-manager) |
| Service account tokens | Platform | Automated (SPIRE) |
| Terraform state encryption key | SRE-Platform | Manual |

## 3. Manual Rotation: PostgreSQL Application Password

```bash
# Step 1: Generate a new password
NEW_PASS=$(openssl rand -base64 32)

# Step 2: Create the new user/role in PostgreSQL (zero-downtime via connection pooler)
psql -h $PGHOST -U admin -d helix -c "CREATE ROLE helix_app_new WITH LOGIN PASSWORD '$NEW_PASS';"
psql -h $PGHOST -U admin -d helix -c "GRANT helix_app TO helix_app_new;"

# Step 3: Update the Kubernetes secret
kubectl create secret generic db-credentials \
  --from-literal=password="$NEW_PASS" \
  -n production --dry-run=client -o yaml | kubectl apply -f -

# Step 4: Rolling restart of services to pick up the new secret
kubectl rollout restart deployment/auth-service -n production
kubectl rollout restart deployment/billing-service -n production
kubectl rollout status deployment/auth-service -n production

# Step 5: Verify connectivity
kubectl exec -it deployment/auth-service -n production -- \
  psql "$DATABASE_URL" -c "SELECT 1;"

# Step 6: Drop the old role after a 24-hour grace period
psql -h $PGHOST -U admin -d helix -c "DROP ROLE helix_app;"
```

## 4. Manual Rotation: JWT Ed25519 Signing Key (Dual-Key)

```bash
# Step 1: Generate a new Ed25519 key pair
openssl genpkey -algorithm Ed25519 -out jwt-signing-key-new.pem
openssl pkey -in jwt-signing-key-new.pem -pubout -out jwt-signing-key-new.pub.pem

# Step 2: Upload the new public key to the verification service
# The auth service must accept BOTH old and new public keys during the transition window.
kubectl create secret generic jwt-signing-keys \
  --from-file=key-new.pem=jwt-signing-key-new.pem \
  --from-file=key-old.pem=jwt-signing-key-current.pem \
  -n production --dry-run=client -o yaml | kubectl apply -f -

# Step 3: Rolling restart of auth service
kubectl rollout restart deployment/auth-service -n production
kubectl rollout status deployment/auth-service -n production

# Step 4: Verify that newly issued tokens use the new key (check kid header)
curl -s https://auth.helix.internal/.well-known/jwks.json | jq '.keys[] | select(.kid=="new")'

# Step 5: After 7-day grace period (or max token TTL), remove the old key
kubectl create secret generic jwt-signing-keys \
  --from-file=key.pem=jwt-signing-key-new.pem \
  -n production --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/auth-service -n production
```

## 5. Automated Rotation: Vault Transit Key

```bash
# Vault transit keys support automatic rotation via the API.
# This is typically triggered by a CI/CD pipeline or cron job.

# Step 1: Rotate the key
vault write -f transit/keys/helix-app/rotate

# Step 2: Verify the new key version
vault read transit/keys/helix-app | jq '.data.keys | keys'

# Step 3: (Optional) Auto-unseal and rewrap old ciphertext
vault write transit/keys/helix-app/config \
  auto_rotate_period="720h"  # 30 days

# Step 4: Update the key version annotation in service configs
kubectl annotate secret vault-transit-config -n production \
  "vault.hashicorp.com/key-version=$(vault read -format=json transit/keys/helix-app | jq -r '.data.latest_version')" \
  --overwrite
```

## 6. Automated Rotation: cert-manager TLS Certificates

```bash
# cert-manager handles automated rotation via Certificate resources.
# Force an early rotation if needed:

kubectl annotate certificate api-gateway-tls -n production \
  cert-manager.io/trigger-cert-renewal=true --overwrite

# Verify the new certificate
kubectl get certificate api-gateway-tls -n production -o json | \
  jq -r '.status.conditions[] | select(.type=="Ready") | .status'

kubectl get secret api-gateway-tls -n production -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates
```

## 7. Verification Checklist

- [ ] New secret/key is active in the target namespace.
- [ ] Services have restarted and are healthy.
- [ ] No authentication errors in logs (`grep -i "auth\|permission\|denied"`).
- [ ] End-to-end smoke tests pass.
- [ ] Old secret/key is revoked or deleted after the grace period.
- [ ] Incident ticket is updated with rotation timestamp and key IDs.

## 8. References
- `docs/guides/runbooks/CERTIFICATE_ROTATION.md`
- `docs/guides/runbooks/VAULT_BREACH.md`
- `infrastructure/terraform/vault/` — Vault infrastructure
