# FAILOVER_PROCEDURE.md

## 1. Objective

Execute a cross-region disaster recovery (DR) failover of the helix_terminator platform from **us-east-1** (primary) to **eu-west-1** (secondary). This runbook assumes:
- eu-west-1 infrastructure is pre-provisioned and in a "warm standby" state.
- Database replication and Kafka MirrorMaker 2 are active.
- Terraform state and Helm values are version-controlled.

## 2. Pre-Flight Checks

Run these commands from the bastion host or CI/CD pipeline with `kubectl` and `terraform` access.

```bash
# Step 0: Verify you are targeting the correct cluster
kubectl config current-context
# Expected: arn:aws:eks:us-east-1:111111111111:cluster/helix-prod

# Step 0b: Verify eu-west-1 cluster is reachable
kubectl --context=arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr get nodes
```

## 3. The 7-Step Failover Runbook

### Step 1: Declare the Disaster
```bash
# Create the incident channel and page the DR team
slack channel create --name "dr-failover-$(date +%Y%m%d-%H%M)"
pd incident:create --title "DR Failover us-east-1 -> eu-west-1" --service "dr-prod"
```

### Step 2: Stop Writes in Primary (us-east-1)
```bash
# Scale down all mutable services in us-east-1 to prevent split-brain writes
kubectl config use-context arn:aws:eks:us-east-1:111111111111:cluster/helix-prod

for svc in auth-service billing-service collaboration-service ai-service; do
  kubectl scale deployment "$svc" --replicas=0 -n production
done

# Verify zero active pods
kubectl get deployments -n production | grep -E 'auth-service|billing-service|collaboration-service|ai-service'
```

### Step 3: Promote PostgreSQL Replica in eu-west-1
```bash
# Connect to the eu-west-1 Cloud SQL / RDS instance and promote it
# For RDS:
aws rds --region eu-west-1 promote-read-replica \
  --db-instance-identifier helix-postgres-prod-dr

# Wait for promotion
aws rds --region eu-west-1 wait db-instance-available \
  --db-instance-identifier helix-postgres-prod-dr

# Update the application connection string secret in eu-west-1
kubectl --context=arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr \
  set secret db-credentials \
  --from-literal=uri="postgres://helix-postgres-prod-dr.XXXXXXXX.eu-west-1.rds.amazonaws.com:5432/helix" \
  -n production --dry-run=client -o yaml | kubectl apply -f -
```

### Step 4: Promote Kafka in eu-west-1
```bash
# Stop MirrorMaker 2 in us-east-1 (prevents reverse replication)
kubectl --context=arn:aws:eks:us-east-1:111111111111:cluster/helix-prod \
  scale deployment kafka-mirror-maker-2 --replicas=0 -n data

# In eu-west-1, reconfigure brokers to become the active cluster
# Update the Kafka CR (if using Strimzi) or broker config
kubectl --context=arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr \
  patch kafka helix-kafka -n data --type=merge -p '
  {"spec":{"kafka":{"config":{"min.insync.replicas":2}}}}'

# Verify topic leadership has migrated
kubectl --context=arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr \
  exec -it kafka-broker-0 -n data -- \
  kafka-topics.sh --bootstrap-server localhost:9092 --describe
```

### Step 5: Redirect Traffic
```bash
# Update the global load balancer / DNS to point to eu-west-1
# Cloudflare example:
cfcli --zone helix.internal dns update \
  --name api.helix.internal --content <eu-west-1-nlb-ip> --type A

# If using Route 53:
aws route53 change-resource-record-sets \
  --hosted-zone-id ZXXXXXXXXXXXXX \
  --change-batch file://eu-west-1-alias.json

# Verify DNS propagation
dig +short api.helix.internal
```

### Step 6: Scale Up Services in eu-west-1
```bash
kubectl --context=arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr \
  config use-context arn:aws:eks:eu-west-1:111111111111:cluster/helix-prod-dr

# Scale up all production services
for svc in auth-service billing-service collaboration-service ai-service gateway-service health-service; do
  kubectl scale deployment "$svc" --replicas=3 -n production
done

# Verify readiness
kubectl get pods -n production
kubectl rollout status deployment/gateway-service -n production
```

### Step 7: Validate and Communicate
```bash
# Run smoke tests
./scripts/testing/smoke_test.sh --endpoint https://api.helix.internal --region eu-west-1

# Verify critical user journeys (login, billing, collaboration)
./scripts/testing/e2e_critical_path.sh --region eu-west-1

# Update status page and resolve PagerDuty incident
pd incident:resolve -i <DR_INCIDENT_ID>
slack post --channel "#incidents" --message "DR failover to eu-west-1 complete. All systems nominal."
```

## 4. Post-Failover Actions

- **Do not** scale us-east-1 back up until the root cause is fully understood.
- Monitor replication lag from eu-west-1 back to us-east-1 (when it becomes the new DR target).
- Schedule a post-mortem within 24 hours.
- Update Terraform to reflect the new primary region if the failover is permanent.

## 5. Rollback (Failback to us-east-1)

When us-east-1 is restored:
1. Reverse Step 2: scale down eu-west-1 mutable services.
2. Reverse Step 3: demote eu-west-1 PostgreSQL to read replica; promote us-east-1.
3. Reverse Step 4: restart MirrorMaker 2 toward us-east-1.
4. Reverse Step 5: update DNS to us-east-1.
5. Reverse Step 6: scale up us-east-1 services.
6. Validate with smoke tests.

## 6. References
- `docs/guides/runbooks/POSTGRES_PITR.md`
- `docs/guides/runbooks/KAFKA_RECOVERY.md`
- `infrastructure/terraform/dr/` — DR infrastructure definitions
