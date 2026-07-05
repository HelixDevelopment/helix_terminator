# ADR-004: PostgreSQL over CockroachDB for Primary Datastore

## Status
Accepted

## Context
helix_terminator needs a relational primary datastore for transactional workloads: user accounts, billing records, project metadata, and configuration. Requirements include ACID compliance, strong consistency, rich query capabilities, and mature operational tooling.

## Decision
We chose **PostgreSQL** (via managed Cloud SQL on GCP and RDS on AWS) as the primary relational datastore. CockroachDB is not used for primary OLTP.

## Consequences

### Positive
- **Mature ecosystem**: PostgreSQL has decades of tooling: pg_dump, pg_basebackup, WAL archiving, logical replication, and a vast extension library (PostGIS, pg_stat_statements).
- **Operational familiarity**: The team has deep prior experience with PostgreSQL tuning, query optimization, and incident response.
- **Managed offerings**: Cloud SQL and RDS provide automated backups, PITR, and high-availability failover with minimal operational burden.
- **Cost efficiency**: For our workload profile, PostgreSQL is cheaper than CockroachDB’s per-node licensing and infrastructure overhead.
- **JSON support**: Native `jsonb` allows flexible schemaless columns where strict relational modeling is premature.

### Negative
- **Horizontal scaling limits**: Write scaling requires application-level sharding or read-replicas; PostgreSQL does not auto-shard.
- **Multi-region writes**: Synchronous multi-region replication adds latency; CockroachDB handles this more gracefully.
- **Cloud lock-in**: Managed PostgreSQL ties us to Cloud SQL/RDS primitives, though logical replication mitigates migration risk.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **CockroachDB** | Excellent for global, strongly consistent distributed SQL, but adds operational complexity (multi-node consensus, certificate rotation, cluster rebalancing) and higher cost. Our workload does not require planet-scale writes; PostgreSQL with read replicas is sufficient. |
| **MySQL / MariaDB** | Viable, but PostgreSQL’s advanced indexing (GIN, BRIN), window functions, and extension ecosystem are stronger fits for our query patterns. |
| **Spanner** | Managed global consistency, but GCP-only and expensive; rejected to preserve multi-cloud portability. |
| **DynamoDB / Cassandra** | Scalable NoSQL options, but lack ACID transactions across documents and complex JOIN capabilities required by our domain model. |

## References
- `infrastructure/terraform/postgres/` — PostgreSQL infrastructure definitions
- `docs/guides/runbooks/POSTGRES_PITR.md` — Recovery procedures
- `services/auth-service/` — Primary PostgreSQL client
