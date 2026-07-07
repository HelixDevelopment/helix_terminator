# POSTGRES_PITR.md

## 1. Objective

Perform Point-in-Time Recovery (PITR) of the helix_terminator PostgreSQL primary database to a specific timestamp. This runbook applies to managed PostgreSQL (Cloud SQL, RDS) and self-managed PostgreSQL with WAL archiving.

## 2. Prerequisites

- WAL archiving is enabled (`archive_mode = on`, `archive_command` configured).
- Base backups are taken regularly (e.g., via `pg_basebackup` or managed automated backups).
- You have the target recovery timestamp in UTC (ISO 8601 format).
- You have admin credentials for the database and cloud provider.

## 3. Step-by-Step PITR Procedure

### Step 1: Identify the Target Recovery Time
```bash
TARGET_TIME="2026-07-05T12:00:00Z"
echo "Recovery target: $TARGET_TIME"
```

### Step 2: Stop All Application Writes
```bash
# Scale down mutable services to prevent new writes during recovery
kubectl scale deployment auth-service billing-service collaboration-service --replicas=0 -n production

# Verify no active connections
psql -h $PGHOST -U admin -d helix -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND usename = 'helix_app';"
```

### Step 3: Create a New PITR Instance (Managed)

#### Cloud SQL (GCP)
```bash
gcloud sql instances create helix-postgres-pitr \
  --source-instance=helix-postgres-prod \
  --point-in-time-recovery-time="$TARGET_TIME" \
  --tier=db-custom-4-16384 \
  --region=us-east1
```

#### Amazon RDS
```bash
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier helix-postgres-prod \
  --target-db-instance-identifier helix-postgres-pitr \
  --restore-time "$TARGET_TIME" \
  --db-instance-class db.r5.xlarge
```

### Step 4: PITR for Self-Managed PostgreSQL

```bash
# Step 4a: Prepare a new data directory
PGDATA_NEW="/var/lib/postgresql/pitr"
rm -rf "$PGDATA_NEW"
mkdir -p "$PGDATA_NEW"

# Step 4b: Restore the latest base backup prior to the target time
pg_basebackup -D "$PGDATA_NEW" -Fp -Xs -P -v \
  -h backup-server -U replicator

# Step 4c: Create recovery signal and configure recovery target
cat > "$PGDATA_NEW/postgresql.auto.conf" <<EOF
restore_command = 'cp /wal_archive/%f %p'
recovery_target_time = '$TARGET_TIME'
recovery_target_action = 'promote'
EOF

touch "$PGDATA_NEW/recovery.signal"

# Step 4d: Start PostgreSQL in recovery mode
pg_ctl -D "$PGDATA_NEW" start

# Step 4e: Monitor recovery progress
psql -h localhost -U postgres -c "SELECT pg_last_xact_replay_timestamp();"

# Step 4f: Once recovery completes, PostgreSQL will promote itself to primary
# Verify
psql -h localhost -U postgres -c "SELECT pg_is_in_recovery();"
# Expected: f (false)
```

### Step 5: Verify Data Integrity
```bash
# Connect to the recovered instance
psql -h $PITR_HOST -U admin -d helix -c "SELECT max(updated_at) FROM users;"
psql -h $PITR_HOST -U admin -d helix -c "SELECT count(*) FROM billing_invoices;"

# Run application-level consistency checks
./scripts/testing/db_consistency_check.sh --host "$PITR_HOST"
```

### Step 6: Redirect Application Traffic
```bash
# Update the application connection string to point to the PITR instance
kubectl create secret generic db-credentials \
  --from-literal=uri="postgres://$PITR_USER:$PITR_PASS@$PITR_HOST:5432/helix" \
  -n production --dry-run=client -o yaml | kubectl apply -f -

# Rolling restart of services
kubectl rollout restart deployment/auth-service -n production
kubectl rollout status deployment/auth-service -n production
```

### Step 7: Resume Writes and Monitor
```bash
kubectl scale deployment auth-service billing-service collaboration-service --replicas=3 -n production
kubectl get pods -n production

# Monitor for errors
kubectl logs -l app=auth-service -n production --tail=100 | grep -i "error\|fatal"
```

## 4. Post-Recovery Actions

- Retain the old primary instance (stopped) for 48 hours as a fallback.
- Document the root cause that necessitated PITR.
- Update automated backup verification tests to cover the recovered schema version.
- If the PITR instance becomes the new primary, update Terraform/RDS identifiers and DNS.

## 5. Rollback (If PITR Was Wrong)

If the target time was incorrect:
1. Stop the PITR instance.
2. Revert the secret to point to the original primary (if it was not modified).
3. Restart services.
4. Re-attempt PITR with a corrected timestamp.

## 6. References
- `docs/guides/runbooks/FAILOVER_PROCEDURE.md`
- `infrastructure/terraform/postgres/` — PostgreSQL infrastructure
