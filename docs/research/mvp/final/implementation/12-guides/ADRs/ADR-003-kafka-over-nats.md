# ADR-003: Kafka + RabbitMQ over NATS for Messaging

## Status
Accepted

## Context
The helix_terminator platform requires a messaging backbone that supports both high-throughput event streaming (audit logs, telemetry, inter-service events) and reliable task queueing (job dispatch, background work, RPC-style request/reply). A single broker was evaluated but no single option covered both patterns optimally.

## Decision
We adopted a **dual-broker strategy**:
- **Apache Kafka** for high-throughput, durable event streaming and log-based messaging.
- **RabbitMQ** for reliable task queueing, request/reply patterns, and complex routing.

NATS (and NATS Streaming / JetStream) was evaluated as a unified alternative but rejected.

## Consequences

### Positive
- **Best-of-breed fit**: Kafka excels at append-only log streams with replay and retention; RabbitMQ excels at routing, priority queues, and dead-letter handling.
- **Ecosystem maturity**: Both have battle-tested operators, monitoring integrations (Prometheus exporters), and client libraries in Go.
- **Operational isolation**: Streaming backpressure and task queueing do not share the same broker, preventing one workload from starving the other.

### Negative
- **Operational complexity**: Two brokers to deploy, monitor, upgrade, and tune.
- **Developer cognitive load**: Engineers must know which broker to use for which pattern; misuse can lead to latency or durability issues.
- **Cost**: Double the infrastructure footprint compared to a single broker.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **NATS (Core + JetStream)** | Lightweight and fast, but JetStream’s persistence and clustering model was less mature than Kafka at the time of evaluation. NATS Core lacks the durability guarantees required for audit and billing events. |
| **Kafka alone** | Poor fit for request/reply and per-message TTL/dead-letter patterns; queue semantics are awkward compared to AMQP. |
| **RabbitMQ alone** | Poor fit for high-throughput log streaming; streams plugin was experimental and Kafka’s ecosystem for stream processing (e.g., Kafka Streams, ksqlDB) is richer. |
| **Pulsar** | Strong on paper, but operational tooling and community size were smaller than Kafka’s, increasing risk. |

## References
- `infrastructure/helm/kafka/` — Kafka Helm charts
- `infrastructure/helm/rabbitmq/` — RabbitMQ Helm charts
- `services/audit-service/` — Primary Kafka consumer
- `services/collaboration-service/` — Primary RabbitMQ consumer
