# ADR-010: Microservices over Monolith for System Architecture

## Status
Accepted

## Context
helix_terminator is a multi-tenant platform with distinct bounded contexts: authentication, billing, real-time collaboration, AI inference, audit logging, analytics, and health monitoring. The system must scale elastically, support independent deployment cadences, and tolerate failures in individual subsystems.

## Decision
We chose a **microservices architecture** decomposed by bounded context. A shared platform library (`pkg/`) provides common concerns (telemetry, config, database connection pooling), but each service owns its own data, deployment lifecycle, and API contract.

## Consequences

### Positive
- **Independent scalability**: The AI inference service can scale GPU nodes independently of the lightweight auth service.
- **Team autonomy**: Squads own services end-to-end, enabling parallel development and independent release schedules.
- **Technology heterogeneity**: Go is the default, but Rust is used for the container bridge and Python for ML model serving without dragging the entire codebase into those ecosystems.
- **Fault isolation**: A memory leak in the analytics consumer does not cascade to the billing API.
- **Incremental rewrite**: Services can be refactored or replaced individually without a big-bang migration.

### Negative
- **Operational complexity**: 25+ services require robust observability, service mesh, and deployment automation; operational overhead is higher than a monolith.
- **Distributed systems challenges**: Network partitions, eventual consistency, and distributed transactions require careful design (sagas, outbox pattern, idempotency keys).
- **Latency**: Inter-service RPC adds network hops; caching and colocation strategies are required to mitigate.
- **Testing**: End-to-end integration tests are slower and flakier than monolith integration tests; contract testing (Pact) is mandatory.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Modular Monolith** | Simpler to deploy and test, but couples release cadences and prevents independent scaling. Considered as a stepping-stone during MVP but rejected for production due to scaling and team-parallelism requirements. |
| **Serverless (Functions-as-a-Service)** | Excellent for event-driven bursts, but cold starts, vendor lock-in, and execution time limits make it unsuitable for long-running collaboration sessions and AI inference workloads. Used selectively for webhook handlers. |
| **Service Fabric / Actor Model (e.g., Orleans, Akka)** | Powerful for stateful distributed objects, but adds framework complexity and is less idiomatic in the Go ecosystem. |
| **Single Monolith** | Rejected outright; the breadth of domains and scaling requirements would force over-provisioning and create a single massive failure domain. |

## References
- `services/` — All microservice source directories
- `go.work` — Go workspace linking shared packages
- `test/contracts/` — Pact contract tests
- `infrastructure/helm/helix-platform/` — Composite deployment chart
