# 11 — Performance Analysis

**Status:** `Draft`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-4) + `SERVICE_REGISTRY.md`

---

## Overview

HelixTerminator is engineered for sub-100ms terminal latency, 60fps UI rendering, and horizontal scalability to 10,000+ concurrent SSH sessions per region. Performance is not an afterthought — it is a first-class requirement with explicit SLOs, load-test benchmarks, and gap analysis.

---

## SLOs (Service Level Objectives)

| Service | Availability | Latency (p99) | Error Rate | Throughput |
|---------|-------------|---------------|------------|------------|
| API Gateway | 99.99% | 50ms | 0.01% | 10,000 req/s |
| Auth Service | 99.99% | 100ms | 0.01% | 5,000 req/s |
| Vault Service | 99.99% | 50ms | 0.001% | 2,000 req/s |
| SSH Proxy | 99.9% | 200ms | 0.1% | 10,000 sessions |
| Terminal Session | 99.9% | 100ms | 0.1% | 5,000 concurrent |
| SFTP | 99.9% | 500ms | 0.1% | 1,000 transfers/s |
| Port Forward | 99.5% | 50ms | 0.5% | 5,000 tunnels |
| Collaboration | 99.9% | 100ms | 0.1% | 100 participants/session |
| AI/Autocomplete | 99.5% | 500ms | 1% | 100 req/s |
| Audit | 99.99% | 200ms | 0.001% | 50,000 events/s |
| Recording | 99.5% | 2s | 0.5% | 100 recordings/min |

---

## Key Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Terminal keystroke latency | < 16ms | Client-side, 95th percentile |
| SSH connection establish | < 500ms | End-to-end, including auth |
| Session share propagation | < 100ms | CRDT sync to all participants |
| AI suggestion latency | < 500ms | From keystroke to UI render |
| Vault unlock | < 200ms | Client-side Argon2id + AES-GCM |
| Audit event ingestion | < 50ms | Kafka → PostgreSQL |
| Recording segment assembly | < 5s | Kafka → S3, per minute segment |
| Cold start (mobile) | < 2s | App launch to interactive |
| Cold start (desktop) | < 1s | App launch to interactive |
| Frame rate | 60fps | All platforms, all screens |
| Memory footprint (idle) | < 150MB desktop, < 80MB mobile | |

---

## Load Test Benchmarks

### k6 Scenarios

| Scenario | VUs | Duration | Target |
|----------|-----|----------|--------|
| Login stress | 1,000 | 5m | 99th percentile < 200ms |
| SSH session storm | 5,000 | 10m | 99th percentile < 500ms |
| SFTP bulk transfer | 500 | 10m | 100MB/s aggregate throughput |
| Vault sync burst | 2,000 | 5m | 99th percentile < 300ms |
| AI autocomplete flood | 1,000 | 5m | 99th percentile < 1s |
| Audit event flood | 10,000 | 10m | Zero dropped events |

### Stress Test Results

> **DEFERRED:** Real load-test results are tooling-only in source doc 09. Actual stress data to 2-3× baseline, soak/chaos results, and cost/perf tradeoffs are not yet authored.

---

## Bottleneck Analysis

| Component | Bottleneck | Mitigation |
|-----------|-----------|------------|
| PostgreSQL (audit) | Write throughput | Partitioning by month, BRIN indexes, async batch insert |
| Kafka (session events) | Consumer lag | Parallel consumers, partition by session_id hash |
| Redis (scrollback) | Memory growth | TTL eviction, compression, cluster sharding |
| Flutter (terminal) | GPU memory (Impeller) | Texture atlas pooling, glyph caching |
| SSH Proxy | Connection table size | Connection pooling, graceful handoff on scale-down |

---

## Danger Zones

1. **Audit sink throughput:** At 50,000 events/s, PostgreSQL partitioning and batching must be tuned. Risk: events dropped during spikes.
2. **CRDT sync latency:** Under high-latency networks (>200ms RTT), collaborative buffer sync may degrade. Risk: user-visible desync.
3. **AI model inference:** On-device models (<50MB) are fast but limited. Cloud inference adds latency. Risk: inconsistent UX.
4. **Certificate rotation:** CA rotation must not invalidate active sessions. Risk: mass disconnection.

---

## Diagrams

| Diagram | Source |
|---------|--------|
| Performance Architecture | `diagrams/mermaid/09_performance_architecture.mmd` |
| Data Flow | `diagrams/mermaid/05_data_flow.mmd` |

---

## Cross-References

- [02 — System Architecture](../02-system-architecture/) — Resilience matrix, failure classes
- [03 — Service Catalog](../03-service-catalog/) — Per-service throughput requirements
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — Observability, SLO dashboards
- [12 — Product Roadmap](../12-product-roadmap/) — Performance benchmarks per phase

---

*Section 11 — Performance Analysis*  
*Consolidated from: 09_performance_analysis.md, CANONICAL_FACTS.md (CD-4)*
