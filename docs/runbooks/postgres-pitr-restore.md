# PostgreSQL PITR Restore Runbook

**Version:** 1.0.0
**Status:** Active
**RTO:** 4 hours
**RPO:** 15 minutes (WAL archiving)
**Authority:** `CANONICAL_FACTS.md` (CD-5) + `DEPLOYMENT_GUIDE.md`

---

## Backup Retention Policy

| Backup Type | Frequency | Retention | Storage |
|-------------|-----------|-----------|---------|
| Full base backup | Daily | 30 days | S3 (encrypted) |
| WAL archives | Continuous | 7 days | S3 (encrypted) |
| Point-in-time recovery | On-demand | 7 days | S3 + local |
| Cross-region replica | Continuous | N/A | DR region |

---

## Restore Procedure

### Step 1: Identify Recovery Point

```bash
# List available base backups
aws s3 ls s3://helixterminator-backups/postgres/base/

# List WAL archives
aws s3 ls s3://helixterminator-backups/postgres/wal/

# Determine target recovery time
# Format: YYYY-MM-DD HH:MM:SS UTC
TARGET_TIME="2026-07-05 14:30:00"
```

### Step 2: Prepare Restore Environment

```bash
# Create new PostgreSQL instance (do not overwrite primary)
# Use Terraform or AWS console to provision new RDS instance

# Or use local restore for testing
cd /restore
mkdir -p pg_restore
cd pg_restore
```

### Step 3: Download Base Backup

```bash
# Download latest base backup before target time
aws s3 cp s3://helixterminator-backups/postgres/base/base_$(date +%Y%m%d).tar.gz ./

# Extract
tar -xzf base_$(date +%Y%m%d).tar.gz
```

### Step 4: Configure Recovery

```bash
# Create recovery.conf (PostgreSQL 12+) or use postgresql.conf
# For PostgreSQL 17, use postgresql.conf with recovery parameters:

cat >> postgresql.conf << 'EOF'
restore_command = 'aws s3 cp s3://helixterminator-backups/postgres/wal/%f %p'
recovery_target_time = '2026-07-05 14:30:00'
recovery_target_action = 'promote'
EOF

# For PostgreSQL 17, use:
cat >> postgresql.conf << 'EOF'
restore_command = 'aws s3 cp s3://helixterminator-backups/postgres/wal/%f %p'
recovery_target_time = '2026-07-05 14:30:00'
recovery_target_action = 'promote'
EOF
```

### Step 5: Start Recovery

```bash
# Start PostgreSQL in recovery mode
pg_ctl -D /restore/pg_restore start

# Monitor recovery progress
tail -f /restore/pg_restore/log/postgresql-*.log

# Wait for recovery to complete
# PostgreSQL will promote to primary when target time is reached
```

### Step 6: Verify Restore

```bash
# Connect to restored database
psql -h localhost -U postgres -d helixterm_auth

# Verify data at target time
SELECT max(created_at) FROM audit_events;
SELECT count(*) FROM users WHERE created_at <= '2026-07-05 14:30:00';

# Check for data integrity
psql -h localhost -U postgres -d helixterm_auth -c "SELECT pg_database_datvalid(oid) FROM pg_database WHERE datname='helixterm_auth';"
```

---

## Verification Steps

| Check | Command | Expected Result |
|-------|---------|---------------|
| Database reachable | `psql -h localhost -U postgres -c "SELECT 1"` | Returns 1 |
| Data present | `SELECT count(*) FROM users;` | > 0 |
| Timestamp correct | `SELECT max(created_at) FROM audit_events;` | ≤ target time |
| No corruption | `pg_dump --schema-only` | Completes without error |
| WAL applied | `SELECT pg_last_xact_replay_timestamp();` | Close to target time |

---

## Rollback Procedure

If restore fails or data is incorrect:

```bash
# Stop restored instance
pg_ctl -D /restore/pg_restore stop

# Delete restore directory
rm -rf /restore/pg_restore

# Restart from Step 2 with different target time
# Or restore from earlier base backup

# If primary was not touched, no rollback needed
# If primary was modified, restore from cross-region replica
```

### Cross-Region Replica Fallback

```bash
# Promote DR replica to primary
aws rds promote-read-replica \
  --db-instance-identifier helixterminator-dr \
  --region eu-west-1

# Update application connection strings
# See FAILOVER_PROCEDURE.md
```

---

## Testing Schedule

| Test | Frequency | Owner |
|------|-----------|-------|
| Backup integrity | Weekly | Platform SRE |
| Restore drill | Monthly | Platform SRE |
| Cross-region restore | Quarterly | Platform SRE |
| Full DR test | Bi-annually | Platform SRE |

---

*HelixTerminator PostgreSQL PITR Restore Runbook*
