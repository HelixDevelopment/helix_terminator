# Backend Challenges

> Coding challenges for backend developers working on HelixTerminator services.

## Challenge 1: Concurrent Session Manager

**Difficulty:** Hard
**Service:** `terminal-service`
**Time:** 4 hours

Implement a concurrent session manager that:
- Handles 10,000+ simultaneous SSH terminal sessions
- Gracefully handles connection drops and reconnections
- Enforces per-user session limits (max 5 concurrent)
- Implements proper resource cleanup on timeout

### Acceptance Criteria
- [ ] All sessions are isolated; no cross-session data leakage
- [ ] Session limit enforced atomically
- [ ] Zero goroutine leaks under load testing
- [ ] Reconnection restores terminal state within 200ms

### Evaluation Rubric
| Criteria | Weight |
|----------|--------|
| Correctness | 40% |
| Concurrency safety | 30% |
| Performance | 20% |
| Code clarity | 10% |

---

## Challenge 2: Zero-Downtime Migration

**Difficulty:** Hard
**Service:** `gateway-service`
**Time:** 6 hours

Design and implement a zero-downtime configuration migration system that:
- Migrates routing rules from v1 to v2 schema
- Supports rollback within 30 seconds
- Validates new schema before applying
- Emits audit events for all changes

### Acceptance Criteria
- [ ] No dropped requests during migration
- [ ] Rollback restores exact previous state
- [ ] Invalid schemas are rejected before any changes
- [ ] Audit trail is complete and queryable

---

## Challenge 3: Distributed Rate Limiter

**Difficulty:** Medium
**Service:** `auth-service`
**Time:** 3 hours

Build a distributed rate limiter using Redis that:
- Supports sliding window algorithm
- Handles Redis failover gracefully (degrade to local memory)
- Returns accurate remaining quota headers
- Is testable without a real Redis instance

### Acceptance Criteria
- [ ] Rate limits are consistent across multiple gateway instances
- [ ] Redis unavailability triggers graceful degradation
- [ ] Headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- [ ] Unit tests cover all edge cases (clock skew, key expiry, etc.)

---

## Challenge 4: Secure Credential Rotation

**Difficulty:** Hard
**Service:** `vault-service`
**Time:** 5 hours

Implement an automatic credential rotation system that:
- Rotates database credentials every 24 hours
- Ensures zero-downtime rotation (dual-active window)
- Encrypts credentials at rest with envelope encryption
- Provides an emergency manual rotation API

### Acceptance Criteria
- [ ] Rotation occurs without service interruption
- [ ] Old credentials expire with a configurable grace period
- [ ] All encryption uses AES-256-GCM with HSM-backed keys
- [ ] Audit log captures every rotation event

---

## Challenge 5: Event Sourced Audit Trail

**Difficulty:** Medium
**Service:** `audit-service`
**Time:** 3 hours

Refactor the audit service to use event sourcing:
- Store all events in an append-only log
- Support temporal queries ("what did the system look like at time T?")
- Implement snapshotting for performance
- Ensure idempotent event processing

### Acceptance Criteria
- [ ] No mutable state in the primary store
- [ ] Snapshots are taken automatically every 1000 events
- [ ] Temporal queries return consistent results
- [ ] Duplicate events are handled idempotently

---

## Challenge 6: gRPC Interceptor Chain

**Difficulty:** Medium
**Service:** All
**Time:** 2 hours

Write a reusable gRPC interceptor chain that provides:
- Request/response logging with configurable verbosity
- Automatic OpenTelemetry trace propagation
- Panic recovery with structured error reporting
- Metrics emission (latency histograms, request counts)

### Acceptance Criteria
- [ ] Interceptors are composable and order-independent where possible
- [ ] Panics are recovered and logged without crashing the server
- [ ] Traces include service name, method name, and outcome
- [ ] Metrics are compatible with Prometheus exposition format

---

## Challenge 7: Multi-Tenant Data Isolation

**Difficulty:** Hard
**Service:** `workspace-service`
**Time:** 4 hours

Ensure strict data isolation between tenants:
- Row-level security policies in PostgreSQL
- Tenant context propagation through all service layers
- Prevention of cross-tenant query injection
- Validation tests that prove isolation

### Acceptance Criteria
- [ ] A tenant can never access another tenant's data
- [ ] Tenant context is propagated via gRPC metadata
- [ ] SQL queries are automatically scoped to the tenant
- [ ] Fuzz tests attempt and fail to bypass isolation

---

## Challenge 8: Circuit Breaker with Half-Open State

**Difficulty:** Medium
**Service:** `gateway-service`
**Time:** 2 hours

Implement a circuit breaker for upstream service calls:
- States: Closed, Open, Half-Open
- Configurable failure threshold and timeout
- Automatic health probe in Half-Open state
- Metrics and alerting hooks

### Acceptance Criteria
- [ ] Circuit opens after N consecutive failures
- [ ] Half-Open allows a single probe request
- [ ] Successful probe closes the circuit
- [ ] Metrics expose state transitions

---

## Challenge 9: Optimistic Locking for Collaborative Editing

**Difficulty:** Hard
**Service:** `collaboration-service`
**Time:** 5 hours

Implement optimistic locking for real-time collaborative editing:
- Version vectors for conflict detection
- Automatic merge for non-conflicting changes
- Manual conflict resolution UI hook
- Preservation of edit history

### Acceptance Criteria
- [ ] Concurrent edits to different regions merge automatically
- [ ] Conflicting edits surface a conflict resolution token
- [ ] Version vectors are monotonic and comparable
- [ ] Edit history is immutable and queryable

---

## Challenge 10: Custom Metrics Pipeline

**Difficulty:** Medium
**Service:** `analytics-service`
**Time:** 3 hours

Build a custom metrics aggregation pipeline:
- Collect metrics from all services via OTLP
- Aggregate into 1-minute, 5-minute, and 1-hour windows
- Support real-time and batch query modes
- Export to Prometheus and optional S3 parquet

### Acceptance Criteria
- [ ] Aggregation is accurate and handles late-arriving data
- [ ] Query latency < 100ms for real-time mode
- [ ] Batch queries can scan 30 days in < 5 seconds
- [ ] Data is retained with configurable TTL

---

## Submission Guidelines

1. Fork the repository and create a branch: `challenge/<your-name>-<challenge-number>`
2. Write your solution in Go 1.22+
3. Include comprehensive tests (`go test -race -coverprofile=coverage.out ./...`)
4. Open a draft PR for review
5. Tag `@helix-backend-reviewers` for feedback

## Scoring

- **Pass:** All acceptance criteria met, tests > 80% coverage, no race conditions
- **Merit:** Pass + performance benchmarks exceed baseline by 20%
- **Distinction:** Merit + innovative approach or reusable library contribution
