# SSH_CA_INCIDENT.md

## 1. Objective

Respond to a compromise of the SSH Certificate Authority (CA) used to sign host and user certificates for helix_terminator bastion and node access.

## 2. Scope

This runbook covers:
- Detection of CA private key exfiltration or unauthorized certificate issuance.
- Immediate revocation of all existing certificates.
- Rotation of the CA key pair.
- Re-issuance of host and user certificates.
- Verification of infrastructure integrity.

## 3. Detection

Indicators of compromise:
- Alert: `vault_audit` shows unexpected `pki/issue/ssh` calls.
- Alert: HSM tamper detection triggered.
- Anomalous SSH connections in `auth.log` / CloudTrail from unknown IPs.
- Report from security team of leaked CA private key material.

## 4. Immediate Response (First 15 Minutes)

```bash
# Step 1: Declare a SEV-1 security incident
slack channel create --name "incident-ssh-ca-$(date +%Y%m%d-%H%M)"
pd incident:create --title "SSH CA Compromise" --service "security-prod" --urgency high

# Step 2: Revoke ALL existing SSH certificates via Vault
# This marks all previously issued certificates as invalid.
vault write ssh/roles/helix-host/revoke-all
vault write ssh/roles/helix-user/revoke-all

# Step 3: If Vault itself is compromised, seal it immediately
vault operator seal

# Step 4: Disable SSH CA signing endpoints
vault policy write ssh-ca-lockdown - <<EOF
path "ssh/*" {
  capabilities = ["deny"]
}
EOF
vault auth tune -default-policy=ssh-ca-lockdown ssh/

# Step 5: Rotate the CA key pair (generate new keys offline if HSM is suspect)
# If using Vault PKI:
vault write ssh/config/ca generate_signing_key=true

# If using an offline HSM, generate new key pair on the HSM and import the public key.
```

## 5. Key Rotation and Re-Issuance (Next 30 Minutes)

```bash
# Step 6: Generate a new SSH CA key pair (if not using Vault auto-generation)
ssh-keygen -t ed25519 -f /secure/ssh_ca_new -C "helix-ssh-ca-$(date +%Y%m%d)"

# Step 7: Update the trusted CA public key on ALL hosts
# This must be pushed via configuration management (Ansible / Terraform / cloud-init).
for host in $(cat /etc/helix/bastion_hosts.txt); do
  ssh "$host" "sudo tee /etc/ssh/trusted-user-ca-keys.pem" < /secure/ssh_ca_new.pub
  ssh "$host" "sudo systemctl restart sshd"
done

# Step 8: Re-issue host certificates for all infrastructure nodes
for host in $(cat /etc/helix/all_hosts.txt); do
  ssh-keygen -s /secure/ssh_ca_new -I "$host" -h -n "$host" -V +52w /etc/ssh/ssh_host_ed25519_key.pub
  scp /etc/ssh/ssh_host_ed25519_key-cert.pub "$host:/etc/ssh/"
  ssh "$host" "sudo systemctl restart sshd"
done

# Step 9: Re-issue user certificates for all authorized personnel
for user in $(cat /etc/helix/authorized_users.txt); do
  ssh-keygen -s /secure/ssh_ca_new -I "$user" -n "$user" -V +1w "/home/$user/.ssh/id_ed25519.pub"
  scp "/home/$user/.ssh/id_ed25519-cert.pub" "$user@bastion.helix.internal:/home/$user/.ssh/"
done
```

## 6. Infrastructure Verification (Next 60 Minutes)

```bash
# Step 10: Verify no unauthorized certificates are trusted
for host in $(cat /etc/helix/all_hosts.txt); do
  ssh "$host" "sudo ssh-keygen -L -f /etc/ssh/ssh_host_ed25519_key-cert.pub"
done

# Step 11: Audit all recent SSH connections
aws logs filter-log-events \
  --log-group-name /var/log/auth.log \
  --filter-pattern "Accepted certificate" \
  --start-time $(date -d '1 hour ago' +%s)000

# Step 12: Verify Vault audit logs for any further unauthorized signing
vault audit list
vault read sys/audit/file
# Inspect the audit log file for unexpected `pki/issue/ssh` entries.

# Step 13: Confirm HSM integrity (if applicable)
# Follow vendor-specific HSM tamper-evidence verification procedures.
```

## 7. Post-Incident Actions

- **Forensics**: Preserve all logs, disk images of affected systems, and HSM audit trails.
- **Communication**: Notify affected users that old SSH certificates are revoked; distribute new certificates securely.
- **Policy Review**: Shorten certificate TTLs (e.g., from 52 weeks to 1 week for users, 4 weeks for hosts).
- **Monitoring**: Add alerting for unexpected certificate issuance rates.
- **Post-Mortem**: Schedule within 24 hours. Include security, SRE, and legal teams.

## 8. Verification Checklist

- [ ] Old CA public key is removed from all `/etc/ssh/trusted-user-ca-keys.pem` files.
- [ ] New CA public key is present and `sshd` has restarted on all hosts.
- [ ] All host certificates are signed by the new CA.
- [ ] All active users have new certificates with short TTLs.
- [ ] Vault SSH signing endpoint is re-enabled only after policy review.
- [ ] No unauthorized SSH connections observed in logs for 24 hours.

## 9. References
- `docs/guides/runbooks/KEY_ROTATION.md`
- `docs/guides/runbooks/VAULT_BREACH.md`
- `infrastructure/security/ssh/` — SSH CA configuration
