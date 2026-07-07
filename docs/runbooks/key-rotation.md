# Key Rotation Runbook

**Version:** 1.0.0
**Status:** Active
**Frequency:** 90 days (encryption keys), on-demand (compromised keys)
**Authority:** `CANONICAL_FACTS.md` (CD-7) + `SECURITY_RUNBOOK.md`

---

## Rotation Schedule

| Key Type | Lifetime | Rotation Frequency | Owner |
|----------|----------|-------------------|-------|
| Encryption keys (AES-256-GCM) | 90 days | 60 days before expiry | Security Team |
| JWT signing key (Ed25519) | 90 days | Scheduled or on-demand | Auth Team |
| Database encryption key | 90 days | 60 days before expiry | Platform SRE |
| API key secrets | 90 days | 60 days before expiry | Security Team |
| SSH host keys | 365 days | 90 days before expiry | Platform SRE |
| HSM key material | Per policy | Per HSM policy | Security Team |

---

## Pre-Rotation Checklist

- [ ] Verify current key status and expiry
- [ ] Confirm no active incidents
- [ ] Notify on-call rotation
- [ ] Verify backup key availability
- [ ] Confirm rollback plan documented
- [ ] Schedule maintenance window (if customer-facing)

---

## Key Generation Procedure

### Encryption Key (AES-256-GCM)

```bash
# Generate 256-bit key
openssl rand -base64 32 > new-encryption-key.txt

# Verify key strength
wc -c new-encryption-key.txt  # Should be ~45 bytes (base64 of 32 bytes)
```

### JWT Signing Key (Ed25519)

```bash
# Generate new Ed25519 keypair
openssl genpkey -algorithm Ed25519 -out new-jwt-private.pem
openssl pkey -in new-jwt-private.pem -pubout -out new-jwt-public.pem

# Verify keypair
openssl pkey -in new-jwt-private.pem -check
```

### Database Encryption Key

```bash
# Generate key for pgcrypto
openssl rand -hex 32 > new-db-encryption-key.txt

# Update PostgreSQL
psql $DATABASE_URL -c "SELECT pgp_sym_encrypt('test', '$(cat new-db-encryption-key.txt)');"
```

---

## Distribution Procedure

### Step 1: Update Kubernetes Secrets

```bash
# Create new secret
kubectl create secret generic encryption-key-new \
  --from-file=key=new-encryption-key.txt \
  -n helixterminator --dry-run=client -o yaml | kubectl apply -f -

# For JWT keys
kubectl create secret generic jwt-signing-key-new \
  --from-file=private=new-jwt-private.pem \
  --from-file=public=new-jwt-public.pem \
  -n helixterminator --dry-run=client -o yaml | kubectl apply -f -
```

### Step 2: Rolling Update

```bash
# Update deployment to reference new secret
kubectl patch deployment <service> -n helixterminator \
  --type='json' -p='[{"op": "add", "path": "/spec/template/spec/volumes/-", "value":{"name":"new-key","secret":{"secretName":"encryption-key-new"}}}]'

# Rolling restart
kubectl rollout restart deployment <service> -n helixterminator
kubectl rollout status deployment <service> -n helixterminator
```

### Step 3: JWKS Update (for JWT keys)

```bash
# Add new key to JWKS with future kid
# Keep old key for token TTL overlap (15 minutes)

# Wait for all old tokens to expire
sleep 900  # 15 minutes

# Remove old key from JWKS
curl -X DELETE https://auth.helixterminator.io/.well-known/jwks.json \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"kid": "old-key-id"}'
```

---

## Verification Steps

| Check | Command | Expected Result |
|-------|---------|---------------|
| Key loaded | `kubectl exec <pod> -- cat /secrets/key` | New key content |
| JWT valid | `jwt decode <token>` | Signed with new kid |
| Encryption works | Service-specific test | Data encrypts/decrypts |
| DB encryption | `psql -c "SELECT pgp_sym_decrypt(...)"` | Decrypts correctly |
| No errors | `kubectl logs <pod>` | No key-related errors |
| API healthy | `curl -sf https://api.helixterminator.io/healthz` | HTTP 200 |

---

## Rollback Procedure

If rotation causes issues:

```bash
# Restore previous secret
kubectl apply -f /backup/old-key-secret.yaml

# Restart affected services
kubectl rollout restart deployment <service> -n helixterminator

# Verify restoration
kubectl get pods -n helixterminator
kubectl logs <pod> | grep -i "key"
```

---

## Compromised Key Response

If a key is suspected compromised:

1. **Immediate containment**
   ```bash
   # Revoke all tokens (for JWT)
   curl -X POST https://auth.helixterminator.io/api/v1/auth/revoke-all \
     -H "Authorization: Bearer $ADMIN_TOKEN"
   
   # Rotate key immediately (follow procedure above)
   ```

2. **Investigate scope**
   - Check audit logs for key usage
   - Review access patterns
   - Identify potentially affected data

3. **Notify stakeholders**
   - Security Lead (immediate)
   - CTO (if P0)
   - Legal (if customer data involved)

4. **Document and remediate**
   - Write incident report
   - Update key management procedures
   - Review access controls

---

*HelixTerminator Key Rotation Runbook*
