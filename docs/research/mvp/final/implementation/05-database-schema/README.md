# 05 — Database Schema

**Status:** `Complete`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-4: PostgreSQL 17.2) + `SERVICE_REGISTRY.md`  

---

## Overview

HelixTerminator uses a **database-per-service** microservices architecture with PostgreSQL 17.2. Each of the 23 services with dedicated databases owns its own PostgreSQL instance (or schema in development). Two services (API Gateway, Health/Monitoring) are stateless and have no dedicated database.

| Statistic | Count |
|-----------|-------|
| Services with dedicated DB | 23 |
| Total `CREATE TABLE` statements | 120 |
| Total indexes | 261 |
| Partition tables | 24 (quarterly/monthly) |
| RLS policies | 60+ |

---

## Database-per-Service Map

| Service | Database | File |
|---------|----------|------|
| Auth Service | `helixterm_auth` | `auth_db.sql` |
| User Service | `helixterm_users` | `user_db.sql` |
| Vault Service | `helixterm_vault` | `vault_db.sql` |
| Host Service | `helixterm_hosts` | `host_db.sql` |
| SSH Proxy Service | `helixterm_ssh_proxy` | `session_db.sql` |
| Terminal Session Service | `helixterm_terminal` | `session_db.sql` |
| SFTP Service | `helixterm_sftp` | `session_db.sql` |
| Port Forwarding Service | `helixterm_port_forward` | `session_db.sql` |
| Snippet Service | `helixterm_snippets` | `snippet_db.sql` |
| Keychain Service | `helixterm_keychain` | `keychain_db.sql` |
| Workspace Service | `helixterm_workspaces` | `workspace_db.sql` |
| Collaboration Service | `helixterm_collab` | `collab_db.sql` |
| Notification Service | `helixterm_notifications` | `notification_db.sql` |
| Audit Service | `helixterm_audit` | `audit_db.sql` |
| Analytics Service | `helixterm_analytics` | `analytics_db.sql` |
| AI Service | `helixterm_ai` | `ai_db.sql` |
| Session Recording Service | `helixterm_recordings` | `recording_db.sql` |
| PKI Service | `helixterm_pki` | `pki_db.sql` |
| Organization/Team Service | `helixterm_org` | `org_db.sql` |
| Billing Service | `helixterm_billing` | `billing_db.sql` |
| Configuration Service | `helixterm_config` | `config_db.sql` |
| Container Registry Bridge | `helixterm_container_bridge` | `container_bridge_db.sql` |
| HelixTrack Integration | `helixterm_helixtrack_bridge` | `helixtrack_db.sql` |

---

## Schema Conventions

- All primary keys are `UUID` using `gen_random_uuid()` (pgcrypto).
- All timestamps are `TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()`.
- Soft delete: `deleted_at TIMESTAMP WITH TIME ZONE` — `NULL` means not deleted.
- Partial indexes on `deleted_at IS NULL` for all soft-delete tables.
- `ENUM`-like status columns use `VARCHAR` with `CHECK` constraints.
- `JSONB` for flexible metadata fields with GIN indexes.
- `BRIN` indexes on all `created_at` / `occurred_at` for append-heavy tables.
- Every multi-tenant table has `ROW LEVEL SECURITY` + `FORCE ROW LEVEL SECURITY` + policy.

---

## Row-Level Security (RLS)

Every table carrying tenant-scoping data is protected by PostgreSQL RLS. Two session variables carry request-scoped context:

| Variable | Set by | Scopes |
|----------|--------|--------|
| `app.current_org` | Every request handler within an org | Tables with `org_id` |
| `app.current_user_id` | Every authenticated request | `auth_db` tables (no org_id) |

Connection wiring uses `SET LOCAL` per transaction (safe under PgBouncer transaction-pooling):

```go
func WithOrgScope(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, fn func(pgx.Tx) error) error {
    tx, err := pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)
    if _, err := tx.Exec(ctx, "SELECT set_config('app.current_org', $1, true)", orgID.String()); err != nil {
        return err
    }
    if err := fn(tx); err != nil { return err }
    return tx.Commit(ctx)
}
```

---

## Artifact Files

| File | Description |
|------|-------------|
| `auth_db.sql` | Users, sessions, MFA, SSO, API keys, login history |
| `vault_db.sql` | Vaults, items, versions, sync states, audit events |
| `host_db.sql` | Hosts, groups, labels, fingerprints, connection history, jump chains |
| `session_db.sql` | SSH sessions, events, recordings, SFTP sessions/transfers, port-forward rules/connections |
| `snippet_db.sql` | Categories, snippets, executions, results |
| `keychain_db.sql` | SSH keys, deployments, usage log, certificate store |
| `workspace_db.sql` | Workspaces, snapshots, sessions, templates |
| `collab_db.sql` | Collaboration sessions, participants, events |
| `org_db.sql` | Organizations, members, teams, roles, invitations |
| `audit_db.sql` | Audit events, hash chain, exports, PII keys |
| `pki_db.sql` | CA keys, certificates, revocations, CRL entries |
| `notification_db.sql` | Notifications, templates, digests |
| `analytics_db.sql` | Metrics, dashboards, SLO data |
| `ai_db.sql` | Suggestions, models, feedback |
| `recording_db.sql` | Recording metadata, transcripts |
| `billing_db.sql` | Subscriptions, invoices, payments |
| `config_db.sql` | Feature flags, operational parameters |
| `container_bridge_db.sql` | Clusters, pods, registries |
| `helixtrack_db.sql` | Links, sync states |

---

## Cross-References

- [03 — Service Catalog](../03-service-catalog/) — Which service owns which database
- [04 — API Specification](../04-api-specification/) — REST endpoints backed by these schemas
- [09 — Security — Zero Trust](../09-security-zero-trust/) — RLS, audit hash chain, PII encryption
- [16 — References](../16-references/) — Canonical version pins (CD-4: PostgreSQL 17.2)

---

*Section 05 — Database Schema*  
*Consolidated from: 07_api_and_database.md §17, CANONICAL_FACTS.md (CD-4)*
