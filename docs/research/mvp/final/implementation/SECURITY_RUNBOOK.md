# HelixTerminator — Security Runbook

**Version:** 1.0.0
**Status:** Draft
**Date:** 2026-07-05
**Classification:** Internal — Authorized Personnel Only
**Authority:** `CANONICAL_FACTS.md` (CD-7, CD-8, CD-10) + `SERVICE_REGISTRY.md`

---

## Incident Severity Levels

| Level | Description | Response Time | Escalation |
|-------|-------------|---------------|------------|
| **P0 — Critical** | Active breach, data exfiltration, RCE | 15 minutes | CEO, Legal, SOC |
| **P1 — High** | Authentication bypass, privilege escalation | 30 minutes | CTO, Security Lead |
| **P2 — Medium** | Vulnerability disclosure, suspicious activity | 2 hours | Engineering Manager |
| **P3 — Low** | Policy violation, misconfiguration | 24 hours | Team Lead |
| **P4 — Informational** | Audit finding, compliance gap | 72 hours | Compliance Officer |

---

## On-Call Rotation

| Role | Primary | Secondary | Escalation |
|------|---------|-----------|------------|
| Security Lead | @security-lead | @security-engineer | @cto |
| Platform SRE | @sre-primary | @sre-secondary | @cto |
| Incident Commander | @ic-primary | @ic-secondary | @ceo |

PagerDuty rotation: `security-oncall` schedule.
Slack channel: `#incidents`

---

## Incident Response Procedures

### Phase 1: Detection & Triage (0–15 min)

1. **Alert received** (PagerDuty / Falco / WAF / manual report)
2. **Acknowledge** within SLA
3. **Classify** severity using table above
4. **Create incident channel** in Slack: `#incident-YYYY-MM-DD-<brief>`
5. **Notify** on-call rotation

### Phase 2: Containment (15–60 min)

1. **Isolate affected systems**
   ```bash
   # Scale down compromised service
   kubectl scale deployment <service> --replicas=0 -n helixterminator

   # Block IP at WAF
   aws wafv2 update-ip-set --name blocked-ips --addresses <ip>/32

   # Revoke compromised tokens
   curl -X POST https://auth.helixterminator.io/api/v1/auth/revoke \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{"user_id": "<uuid>", "reason": "incident"}'
   ```

2. **Preserve evidence**
   - Snapshot logs: `kubectl logs -n helixterminator <pod> --previous > /evidence/<incident>/logs.txt`
   - Snapshot DB: `pg_dump --data-only > /evidence/<incident>/db.sql`
   - Capture network: `tcpdump -w /evidence/<incident>/traffic.pcap`

3. **Document timeline** in incident channel

### Phase 3: Eradication (1–4 hours)

1. **Identify root cause**
   - Review logs in Loki
   - Review traces in Jaeger
   - Review Falco alerts
   - Review WAF logs

2. **Apply fix**
   - Patch vulnerability
   - Rotate credentials
   - Update firewall rules
   - Deploy fix via CI/CD

3. **Verify fix**
   - Re-run security tests
   - Re-scan with Trivy
   - Verify no regression

### Phase 4: Recovery (4–24 hours)

1. **Restore service**
   ```bash
   kubectl scale deployment <service> --replicas=3 -n helixterminator
   ```

2. **Monitor for recurrence**
   - Watch metrics for 24 hours
   - Enable enhanced logging
   - Set temporary alerts

3. **Communicate**
   - Internal: Post-mortem in incident channel
   - External: Customer notification if required (GDPR 72h rule)
   - Legal: Breach notification if required

### Phase 5: Post-Incident (24–72 hours)

1. **Write post-mortem**
   - Timeline
   - Root cause
   - Impact assessment
   - Lessons learned
   - Action items

2. **Update runbooks**
   - Document new detection rules
   - Update response procedures
   - Share with team

3. **Close incident**
   - Archive Slack channel
   - Update incident tracker
   - Schedule follow-up review

---

## Security Procedures

### Certificate Rotation

**Frequency:** 90 days for TLS, 30 days for mTLS, 8 hours for SSH certs

```bash
# 1. Generate new certificate
openssl req -new -newkey ed25519 -keyout new-key.pem -out new-csr.pem -nodes

# 2. Sign with CA
openssl x509 -req -in new-csr.pem -CA ca.pem -CAkey ca-key.pem -out new-cert.pem -days 90

# 3. Deploy to Kubernetes
kubectl create secret tls new-cert --cert=new-cert.pem --key=new-key.pem -n helixterminator

# 4. Update deployment
kubectl rollout restart deployment/<service> -n helixterminator

# 5. Verify
openssl s_client -connect api.helixterminator.io:443 -servername api.helixterminator.io
```

### Key Rotation

**Frequency:** 90 days for encryption keys, on-demand for compromised keys

```bash
# 1. Generate new key
openssl rand -base64 32 > new-key.txt

# 2. Re-encrypt data with new key
# (service-specific procedure)

# 3. Update Kubernetes secret
kubectl create secret generic encryption-key --from-file=key=new-key.txt -n helixterminator

# 4. Verify
# (service-specific verification)
```

### JWT Secret Rotation

```bash
# 1. Generate new Ed25519 keypair
openssl genpkey -algorithm Ed25519 -out new-jwt-private.pem
openssl pkey -in new-jwt-private.pem -pubout -out new-jwt-public.pem

# 2. Update JWKS endpoint
# Add new key with future `kid`, keep old key

# 3. Wait for token TTL (15 minutes)

# 4. Remove old key from JWKS
```

### Password Hash Migration

```bash
# 1. Update hashing parameters
# Argon2id: time_cost=3, memory_cost=65536, parallelism=4

# 2. Re-hash on next login
# Store new hash, invalidate old hash

# 3. Force password reset for at-risk accounts
```

---

## Compliance Procedures

### SOC 2 Type II

| Control | Evidence | Frequency |
|---------|----------|-----------|
| Access control | Audit logs, RBAC reviews | Monthly |
| Change management | PR history, deployment logs | Continuous |
| Encryption | Key rotation logs, cipher audits | Quarterly |
| Monitoring | Alert logs, incident response | Continuous |

### GDPR

| Requirement | Procedure |
|-------------|-----------|
| Right to erasure | Crypto-shred (DEK destruction) |
| Data portability | Export API |
| Breach notification | 72-hour SLA |
| DPO contact | dpo@helixdevelopment.io |

### FedRAMP Moderate

| Control | Implementation |
|---------|----------------|
| AC-2 | RBAC with 6 roles |
| AC-3 | Least privilege enforcement |
| AU-6 | Real-time audit analysis |
| CM-2 | Baseline configuration |
| SC-8 | TLS 1.3, mTLS |

---

## Contact Information

| Role | Contact | Method |
|------|---------|--------|
| Security Lead | security@helixdevelopment.io | Email, PagerDuty |
| Incident Response | #incidents Slack | Slack, PagerDuty |
| Legal | legal@helixdevelopment.io | Email |
| DPO | dpo@helixdevelopment.io | Email |

---

*HelixTerminator Security Runbook*
*Consolidated from: 09-security-zero-trust/README.md, docs/runbooks/*
