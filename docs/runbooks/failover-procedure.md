# Failover Procedure Runbook

**Version:** 1.0.0
**Status:** Active
**Trigger:** Primary region failure, data corruption, security incident
**Authority:** `CANONICAL_FACTS.md` (CD-5, CD-6) + `DEPLOYMENT_GUIDE.md`

---

## Failover Triggers

| Trigger | Severity | Action |
|---------|----------|--------|
| Primary region unavailable >5 min | P0 | Initiate DR failover |
| Database corruption detected | P0 | Initiate DR failover |
| Security incident requiring isolation | P0 | Initiate DR failover |
| Primary region degraded >30 min | P1 | Evaluate partial failover |
| Network partition >15 min | P1 | Evaluate split-brain risk |

---

## Step-by-Step Failover Procedure

### Phase 1: Assessment (0–5 min)

1. **Confirm primary region failure**
   ```bash
   # Check primary health
   kubectl --context=primary get nodes
   # If unreachable, confirm with cloud provider dashboard
   ```

2. **Verify DR region health**
   ```bash
   kubectl --context=dr get nodes
   kubectl --context=dr get pods -n helixterminator
   ```

3. **Classify incident severity**
   - P0: Complete primary region failure
   - P1: Degraded but some services functional

### Phase 2: DR Promotion (5–15 min)

1. **Promote DR database**
   ```bash
   cd infrastructure/terraform/environments/dr
   terraform apply -var="failover=true"
   ```

2. **Update DNS to point to DR**
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z123456789 \
     --change-batch file://dr-failover.json
   ```

3. **Scale DR services to full capacity**
   ```bash
   kubectl --context=dr scale deployment --all --replicas=3 -n helixterminator
   ```

4. **Verify DR services**
   ```bash
   kubectl --context=dr get pods -n helixterminator
   for svc in gateway auth user vault host ssh-proxy terminal; do
     curl -sf https://dr-api.helixterminator.io/healthz || echo "FAIL: $svc"
   done
   ```

### Phase 3: Notification (15–20 min)

1. **Post in incident channel**
   ```
   #incidents: DR failover initiated for [reason]. DR region now active.
   ```

2. **Notify stakeholders**
   - Customer success (if customer-facing impact)
   - Executive team (if P0)
   - Compliance officer (if regulatory impact)

3. **Update status page**
   ```bash
   # Update status page via API
   curl -X POST https://status.helixterminator.io/incidents \
     -d '{"status": "investigating", "impact": "major"}'
   ```

### Phase 4: Monitoring (20 min–ongoing)

1. **Monitor DR health**
   ```bash
   # Watch DR metrics
   kubectl --context=dr top pods -n helixterminator
   
   # Check error rates
   curl http://prometheus-dr:9090/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m])
   ```

2. **Watch for data divergence**
   - Compare DR and primary DB lag
   - Check Kafka consumer offsets
   - Verify no split-brain writes

---

## Verification Checklist

- [ ] DR database is writable and replicating
- [ ] All 25 services are running in DR
- [ ] DNS resolves to DR load balancer
- [ ] API health checks pass (200)
- [ ] WebSocket connections work
- [ ] Kafka consumers are processing
- [ ] No 5xx errors in DR metrics
- [ ] SSL certificate valid for DR endpoint

---

## Rollback Procedure

### When Primary Recovers

1. **Assess primary health**
   ```bash
   kubectl --context=primary get nodes
   kubectl --context=primary get pods -n helixterminator
   ```

2. **Sync data from DR to primary**
   ```bash
   # PostgreSQL replication catch-up
   psql $PRIMARY_DB -c "SELECT pg_is_in_recovery();"
   
   # Kafka mirror-maker
   kafka-mirror-maker.sh --consumer.config dr-consumer.properties \
     --producer.config primary-producer.properties --whitelist ".*"
   ```

3. **Update DNS back to primary**
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z123456789 \
     --change-batch file://primary-restore.json
   ```

4. **Verify primary**
   ```bash
   for svc in gateway auth user vault host ssh-proxy terminal; do
     curl -sf https://api.helixterminator.io/healthz || echo "FAIL: $svc"
   done
   ```

5. **Scale DR down to standby**
   ```bash
   kubectl --context=dr scale deployment --all --replicas=1 -n helixterminator
   ```

6. **Close incident**
   - Update status page
   - Post resolution in incident channel
   - Schedule post-mortem

---

*HelixTerminator Failover Procedure Runbook*
