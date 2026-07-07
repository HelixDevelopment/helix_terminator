# Incident Response Runbook

**Version:** 1.0.0
**Status:** Active
**Severity Levels:** P0 (Critical) → P4 (Informational)
**Authority:** `CANONICAL_FACTS.md` + `SECURITY_RUNBOOK.md`

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

## Escalation Procedures

### Escalation Chain

```
On-Call Engineer → Team Lead → Engineering Manager → CTO → CEO
        ↓
Security Lead (for security incidents)
        ↓
Legal (for breach, regulatory, or contractual issues)
```

### Escalation Criteria

| Condition | Escalate To | Timeframe |
|-----------|-------------|-----------|
| P0 incident | Security Lead + CTO + CEO | Immediate |
| Customer data involved | Legal + DPO | Within 1 hour |
| Regulatory breach | Compliance Officer + Legal | Within 2 hours |
| No progress after 30 min | Next level in chain | 30 minutes |
| Service unavailable >1 hour | CTO | 1 hour |

---

## Communication Templates

### P0 — Critical Incident (Internal)

```
🚨 P0 INCIDENT — [Brief Title]

Time: [YYYY-MM-DD HH:MM UTC]
Severity: P0 — Critical
Status: Investigating
Impact: [Service/Region/Data affected]

Description:
[What happened, what we know, what we don't know]

Actions Taken:
- [Action 1]
- [Action 2]

Next Update: [Time + 15 min]
Incident Channel: #incident-YYYY-MM-DD-[brief]
```

### P0 — Customer Notification

```
Subject: [HelixTerminator] Service Disruption — [Service Name]

We are currently investigating a service disruption affecting [service].

Impact: [Description of customer impact]
Started: [Time UTC]
Status: Investigating

We will provide an update within 30 minutes.

For real-time status: https://status.helixterminator.io
```

### Status Update Template

```
⏳ UPDATE — [Time UTC]

Status: [Investigating / Identified / Monitoring / Resolved]

What we know:
- [Fact 1]
- [Fact 2]

What we're doing:
- [Action 1]
- [Action 2]

Next update: [Time]
```

### Resolution Template

```
✅ RESOLVED — [Time UTC]

Incident: [Brief Title]
Duration: [Start] → [End] ([Duration])

Root Cause:
[Description]

Resolution:
[What was done to fix it]

Impact:
[What was affected and for how long]

Post-mortem:
[Link to post-mortem doc, scheduled for within 72 hours]
```

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

## Communication Channels

| Audience | Channel | Frequency |
|----------|---------|-----------|
| Response team | #incident-YYYY-MM-DD | Real-time |
| Engineering | #incidents | Updates every 30 min |
| Leadership | #leadership-alerts | P0/P1 only |
| Customers | status page + email | As needed |
| Regulators | Legal + DPO | GDPR 72h rule |

---

*HelixTerminator Incident Response Runbook*
