# INCIDENT_RESPONSE.md

## 1. Overview

This runbook defines the incident response process for helix_terminator production systems. All engineers with on-call rotation access must follow this procedure.

## 2. PagerDuty Integration

### 2.1 On-Call Schedule
- Primary and secondary on-call rotations are managed in PagerDuty.
- Escalation policy: Primary → Secondary → Engineering Manager → CTO (15-minute intervals).

### 2.2 Alert Routing
| Service | PagerDuty Service Key | Escalation Policy |
|---------|----------------------|-------------------|
| Auth Service | `auth-service-prod` | SRE-Platform |
| Gateway Service | `gateway-service-prod` | SRE-Platform |
| Kafka Cluster | `kafka-prod` | SRE-Data |
| PostgreSQL Primary | `postgres-prod` | SRE-Data |
| AI Inference | `ai-service-prod` | ML-Platform |

### 2.3 Acknowledging an Alert
```bash
# Acknowledge via PagerDuty CLI (pd)
pd incident:ack -i <INCIDENT_ID>

# Or via Slack /pd ack <INCIDENT_ID>
```

## 3. Severity Classification

| Severity | Criteria | Response Time | Communication Channel |
|----------|----------|---------------|----------------------|
| **SEV-1** | Complete service outage; data loss; security breach | 5 min | War room + executive page |
| **SEV-2** | Major feature degraded; significant customer impact | 15 min | War room + status page |
| **SEV-3** | Minor feature degraded; workaround available | 30 min | Slack #incidents |
| **SEV-4** | Cosmetic issue; no customer impact | 4 hours | Jira ticket |
| **SEV-5** | Observation; potential future risk | Next business day | Jira ticket |

## 4. Escalation Paths

```
On-Call Engineer
    ↓ (cannot resolve within SLO)
Secondary On-Call
    ↓ (cannot resolve within 15 min)
Engineering Manager (SRE / Team Lead)
    ↓ (SEV-1 or cross-team impact)
Incident Commander (IC) + CTO notification
    ↓ (security or legal implication)
Legal + Compliance + External Communications
```

## 5. War Room Procedures

### 5.1 Initiating a War Room
For SEV-1 and SEV-2 incidents:
1. Create a dedicated Slack channel: `#incident-<YYYY-MM-DD>-<short-name>`
2. Start a Zoom bridge (link pinned in channel).
3. Designate an **Incident Commander (IC)** — the IC does not debug; they coordinate.
4. Designate a **Scribe** to log all actions in the Slack thread.
5. Post initial status to the public status page if SEV-1/2.

### 5.2 War Room Command Structure
- **Incident Commander (IC)**: Owns timeline, communication, and escalation decisions.
- **Operations Lead (OL)**: Owns technical mitigation and rollback decisions.
- **Communications Lead (CL)**: Owns internal and external stakeholder updates.

### 5.3 Standing Down
1. OL confirms the fix is stable (monitoring green for 30 min).
2. IC declares the incident resolved.
3. Scribe exports the Slack timeline to the incident Jira ticket.
4. Schedule a post-mortem within 24 hours (SEV-1) or 48 hours (SEV-2).

## 6. Step-by-Step Response Flow

```bash
# Step 1: Acknowledge the page
pd incident:ack -i <ID>

# Step 2: Open the monitoring dashboard
open https://grafana.helix.internal/d/incidents

# Step 3: Check recent deployments
helm history helix-platform -n production

# Step 4: Correlate with logs
kubectl logs -l app=<service> -n production --tail=500 | jq -R 'fromjson? | select(.level=="error")'

# Step 5: If root cause is identified and a rollback is safe:
helm rollback helix-platform <PREVIOUS_REVISION> -n production

# Step 6: Update PagerDuty status
pd incident:resolve -i <ID>

# Step 7: File post-mortem Jira ticket with label "post-mortem"
```

## 7. Communication Templates

### Initial Update (within 5 min of SEV-1)
> We are investigating reports of `<symptom>` affecting `<service>`. We will provide updates every 15 minutes. Status: https://status.helix.internal

### Resolution Update
> `<service>` has been restored. We are monitoring for stability. A post-mortem will be published within 24 hours.

## 8. References
- `docs/guides/runbooks/FAILOVER_PROCEDURE.md`
- `docs/guides/runbooks/VAULT_BREACH.md`
- `docs/guides/runbooks/SSH_CA_INCIDENT.md`
