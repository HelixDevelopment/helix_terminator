# Certificate Rotation Runbook

**Version:** 1.0.0  
**Status:** Active  
**Frequency:** 90 days (TLS), 30 days (mTLS), 8 hours (SSH certs)  
**Authority:** `CANONICAL_FACTS.md` (CD-7) + `SECURITY_RUNBOOK.md`

---

## Rotation Schedule

| Certificate Type | Lifetime | Rotation Frequency | Owner |
|----------------|----------|-------------------|-------|
| TLS (edge) | 90 days | 60 days before expiry | Platform SRE |
| mTLS (service mesh) | 30 days | 14 days before expiry | Security Team |
| SSH CA (user certs) | 8 hours | Automatic | PKI Service |
| SSH CA (host certs) | 90 days | 60 days before expiry | PKI Service |
| JWT signing | 90 days | On-demand or scheduled | Auth Team |
| Code signing | 365 days | 90 days before expiry | DevOps |

---

## Pre-Rotation Checklist

- [ ] Verify current certificate expiry dates
- [ ] Confirm no active incidents
- [ ] Notify on-call rotation
- [ ] Verify backup CA/private key availability
- [ ] Confirm rollback plan documented

---

## TLS Certificate Rotation

### Step 1: Generate New Certificate

```bash
# Generate new private key
openssl genpkey -algorithm RSA -out new-tls-key.pem -pkeyopt rsa_keygen_bits:4096

# Create CSR
openssl req -new -key new-tls-key.pem -out new-tls-csr.pem \
  -subj "/CN=*.helixterminator.io/O=HelixDevelopment/C=US" \
  -addext "subjectAltName=DNS:*.helixterminator.io,DNS:api.helixterminator.io"

# Sign with internal CA (or submit to public CA)
openssl x509 -req -in new-tls-csr.pem -CA ca.pem -CAkey ca-key.pem \
  -out new-tls-cert.pem -days 90 -sha384
```

### Step 2: Deploy to Kubernetes

```bash
# Create new secret
kubectl create secret tls new-tls-cert \
  --cert=new-tls-cert.pem --key=new-tls-key.pem \
  -n helixterminator --dry-run=client -o yaml | kubectl apply -f -

# Update ingress to reference new secret
kubectl patch ingress helixterm-ingress -n helixterminator \
  --type='json' -p='[{"op": "replace", "path": "/spec/tls/0/secretName", "value":"new-tls-cert"}]'

# Verify rollout
kubectl rollout status deployment/gateway -n helixterminator
```

### Step 3: Verification

```bash
# Check certificate chain
openssl s_client -connect api.helixterminator.io:443 -servername api.helixterminator.io

# Verify expiry date
openssl x509 -in new-tls-cert.pem -noout -dates

# Test API health
curl -sf https://api.helixterminator.io/healthz
```

### Step 4: Cleanup

```bash
# After 24h confirmation period, delete old secret
kubectl delete secret old-tls-cert -n helixterminator
```

---

## mTLS Certificate Rotation

### Step 1: SPIRE SVID Rotation

SPIRE automatically rotates SVIDs. To force rotation:

```bash
# Restart SPIRE agent
kubectl rollout restart daemonset spire-agent -n spire

# Verify new SVIDs
kubectl exec -n helixterminator deployment/<service> -- \
  cat /var/run/secrets/spire/svid.pem | openssl x509 -noout -dates
```

### Step 2: Istio Certificate Rotation

```bash
# Restart Istio proxy to pick up new certs
kubectl rollout restart deployment/<service> -n helixterminator

# Verify mTLS
istioctl authn tls-check <pod>.<namespace>
```

---

## SSH CA Certificate Rotation

### Step 1: Generate New CA Keypair

```bash
# Generate new Ed25519 CA key
ssh-keygen -t ed25519 -f new-ssh-ca -C "HelixTerminator SSH CA $(date +%Y-%m-%d)"

# Sign new CA with old CA (cross-certification)
ssh-keygen -s old-ssh-ca -I "cross-cert" -n host -h new-ssh-ca.pub
```

### Step 2: Deploy to PKI Service

```bash
# Update Kubernetes secret
kubectl create secret generic ssh-ca-new \
  --from-file=ca-key=new-ssh-ca --from-file=ca-pub=new-ssh-ca.pub \
  -n helixterminator --dry-run=client -o yaml | kubectl apply -f -

# Restart PKI service
kubectl rollout restart deployment/pki-service -n helixterminator
```

### Step 3: Update SSH Proxy

```bash
# Update trusted CA list
kubectl patch configmap ssh-config -n helixterminator \
  --type='merge' -p='{"data":{"trusted_user_ca_keys":"new-ssh-ca.pub"}}'

# Restart SSH Proxy
kubectl rollout restart deployment/ssh-proxy-service -n helixterminator
```

### Step 4: Revoke Old CA

```bash
# After 7-day overlap period, revoke old CA
curl -X POST https://api.helixterminator.io/api/v1/pki/ca/revoke \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"ca_id": "old-ca-id", "reason": "superseded"}'
```

---

## Verification Steps

| Check | Command | Expected Result |
|-------|---------|---------------|
| TLS expiry | `openssl x509 -in cert.pem -noout -dates` | NotBefore < now < NotAfter |
| mTLS working | `istioctl authn tls-check pod.namespace` | OK: mTLS |
| SSH CA valid | `ssh-keygen -L -f cert.pub` | Valid, signed by correct CA |
| API reachable | `curl -sf https://api.helixterminator.io/healthz` | HTTP 200 |
| Service mesh | `kubectl exec -it <pod> -- curl -sf http://<service>:8080/healthz` | HTTP 200 |

---

## Rollback Procedure

If rotation causes issues:

```bash
# Restore previous secret
kubectl apply -f /backup/old-cert-secret.yaml

# Restart affected services
kubectl rollout restart deployment/<service> -n helixterminator

# Verify restoration
kubectl get pods -n helixterminator
```

---

*HelixTerminator Certificate Rotation Runbook*
