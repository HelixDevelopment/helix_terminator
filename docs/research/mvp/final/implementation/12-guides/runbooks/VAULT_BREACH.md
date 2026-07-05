# VAULT_BREACH.md

## 1. Objective

Respond to a confirmed or suspected compromise of HashiCorp Vault encryption keys, root tokens, or unseal keys in the helix_terminator infrastructure.

## 2. Scope

This runbook covers:
- Detection of unauthorized Vault access or key exfiltration.
- Immediate containment (seal, revoke, audit).
- Root token and unseal key rotation.
- Re-keying and re-encryption of all transit data.
- Verification of integrity and restoration of service.

## 3. Detection

Indicators of compromise:
- Alert: `vault_audit` shows unauthorized `sys/auth`, `sys/policy`, or `transit/decrypt` calls.
- Alert: HSM tamper detection or cloud KMS irregular access.
- Anomalous API request volume from unexpected source IPs.
- Report of leaked unseal key, root token, or recovery key.
- Unexpected secret engine configuration changes.

## 4. Immediate Containment (First 10 Minutes)

```bash
# Step 1: Declare a SEV-1 security incident
slack channel create --name "incident-vault-$(date +%Y%m%d-%H%M)"
pd incident:create --title "Vault Key Compromise" --service "security-prod" --urgency high

# Step 2: Seal Vault immediately to prevent further access
vault operator seal
# If auto-unseal is configured, disable the cloud KMS unseal policy temporarily
# AWS example:
aws kms disable-key --key-id alias/vault-auto-unseal

# Step 3: Revoke all active leases and tokens (emergency blanket revocation)
# WARNING: This will break all active service authentication until re-issued.
vault token revoke -mode path auth/token

# Step 4: Disable all auth methods to prevent re-authentication by the attacker
vault auth disable kubernetes/
vault auth disable approle/
vault auth disable ldap/
# Retain only the root token path for recovery (if root token is secure).

# Step 5: Capture audit logs before they rotate
kubectl cp vault-0:/vault/audit/ /tmp/vault-audit-$(date +%Y%m%d)/
aws s3 cp /tmp/vault-audit-$(date +%Y%m%d)/ s3://helix-security-forensics/vault-audit-$(date +%Y%m%d)/ --recursive
```

## 5. Root Token and Unseal Key Rotation (Next 30 Minutes)

```bash
# Step 6: Unseal Vault using the remaining trusted unseal keys (if any)
# If the unseal key set is compromised, proceed to re-key.

# Step 6a: Generate a new unseal key set (re-key)
vault operator rekey -init -key-shares=5 -key-threshold=3 \
  -pgp-keys="key1.asc,key2.asc,key3.asc,key4.asc,key5.asc"
# Distribute new PGP-encrypted unseal keys to the 5 key holders via secure out-of-band channels.

# Step 6b: Rotate the root token
vault token create -policy=root -ttl=1h > /secure/vault-root-token-new.txt
vault token revoke $(cat /secure/vault-root-token-old.txt)

# Step 7: If auto-unseal is used, rotate the cloud KMS key
# AWS KMS example:
aws kms enable-key-rotation --key-id alias/vault-auto-unseal
# Or create a new KMS key and update Vault configuration:
# Edit the Vault Helm values to reference the new KMS key ID, then:
helm upgrade vault hashicorp/vault -f infrastructure/helm/vault/values.yaml -n vault
kubectl rollout restart statefulset/vault -n vault
```

## 6. Re-Encryption of Transit Data (Next 60 Minutes)

```bash
# Step 8: Rotate all transit encryption keys
for key in $(vault list -format=json transit/keys | jq -r '.[]'); do
  vault write -f "transit/keys/$key/rotate"
done

# Step 9: Re-encrypt all data encrypted with the old key versions
# This is application-dependent. For each service:
#   a. Fetch ciphertext from the database.
#   b. Decrypt with old key version.
#   c. Re-encrypt with new key version.
#   d. Update the database record.

# Example for a single service (auth-service user passwords):
psql -h $PGHOST -U admin -d helix -c \
  "COPY (SELECT id, encrypted_password FROM users) TO STDOUT CSV;" | \
  while IFS=, read -r id ciphertext; do
    plaintext=$(vault write -field=plaintext transit/decrypt/helix-app \
      ciphertext="$ciphertext" key_version=1)
    new_ciphertext=$(vault write -field=ciphertext transit/encrypt/helix-app \
      plaintext="$plaintext")
    psql -h $PGHOST -U admin -d helix -c \
      "UPDATE users SET encrypted_password = '$new_ciphertext' WHERE id = $id;"
  done

# Step 10: Trim old key versions after re-encryption is complete
for key in $(vault list -format=json transit/keys | jq -r '.[]'); do
  vault write "transit/keys/$key/config" min_decryption_version=$(vault read -format=json "transit/keys/$key" | jq -r '.data.latest_version')
done
```

## 7. Restoration of Service (Next 30 Minutes)

```bash
# Step 11: Re-enable auth methods after policy review
vault auth enable kubernetes/
vault auth enable approle/
# Re-configure Kubernetes auth with the new service account issuer:
vault write auth/kubernetes/config \
  token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
  kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443" \
  kubernetes_ca_cert="$(cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt)"

# Step 12: Re-issue AppRole credentials for all services
for svc in auth-service billing-service collaboration-service ai-service; do
  vault write auth/approle/role/$svc policies=$svc token_ttl=1h token_max_ttl=4h
  secret_id=$(vault write -f -field=secret_id auth/approle/role/$svc/secret-id)
  kubectl create secret generic "$svc-vault-credentials" \
    --from-literal=role_id="$svc" \
    --from-literal=secret_id="$secret_id" \
    -n production --dry-run=client -o yaml | kubectl apply -f -
done

# Step 13: Rolling restart of all services to pick up new credentials
for svc in auth-service billing-service collaboration-service ai-service gateway-service health-service; do
  kubectl rollout restart deployment/$svc -n production
  kubectl rollout status deployment/$svc -n production
done

# Step 14: Verify Vault is unsealed and healthy
vault status
vault read sys/health
```

## 8. Post-Incident Actions

- **Forensics**: Preserve all audit logs, HSM tamper evidence, and cloud KMS access logs.
- **Communication**: Notify affected teams that Vault was rotated; no action needed unless they cached old secrets.
- **Policy Review**: Reduce token TTLs, enforce MFA for human auth, and tighten IP allow-lists.
- **Monitoring**: Add anomaly detection on Vault API request volume and unusual auth paths.
- **Post-Mortem**: Schedule within 24 hours. Include security, SRE, platform, and legal teams.

## 9. Verification Checklist

- [ ] Vault is unsealed and all nodes are active.
- [ ] New unseal keys are distributed and old keys are destroyed.
- [ ] New root token is generated and old root token is revoked.
- [ ] All transit keys are rotated.
- [ ] All application data is re-encrypted with new key versions.
- [ ] All services are running and can authenticate to Vault.
- [ ] No unauthorized access in Vault audit logs for 24 hours.
- [ ] HSM / KMS integrity is verified.

## 10. References
- `docs/guides/runbooks/KEY_ROTATION.md`
- `docs/guides/runbooks/SSH_CA_INCIDENT.md`
- `infrastructure/helm/vault/` — Vault Helm configuration
- `infrastructure/terraform/vault/` — Vault infrastructure definitions
