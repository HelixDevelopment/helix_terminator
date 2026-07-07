# ADR-002: Go over Rust/Node.js for Backend Microservices

## Status
Accepted

## Context
helix_terminator is a polyglot backend with ~25 microservices. We needed a primary service language that balances developer velocity, runtime performance, operational simplicity, and hiring pipeline. Services span REST/gRPC APIs, event consumers, background workers, and infrastructure automation hooks.

## Decision
We chose **Go** as the primary backend language. Rust is used for performance-critical components (e.g., the container-bridge service). Node.js is used only for legacy analytics dashboards.

## Consequences

### Positive
- **Fast compilation**: Go builds produce static binaries in seconds, enabling rapid CI/CD iteration.
- **Goroutine concurrency**: Lightweight threads make high-concurrency I/O (gRPC, Kafka consumers) straightforward without async/await complexity.
- **Operational simplicity**: Static binaries with minimal runtime dependencies simplify Docker images and reduce attack surface.
- **Strong standard library**: `net/http`, `encoding/json`, and `database/sql` reduce external dependency sprawl.
- **Hiring and onboarding**: Go is widely known in the SRE and backend community, shortening ramp-up time.

### Negative
- **Generics verbosity**: Pre-Go 1.18 code required boilerplate for generic containers; modern Go mitigates this but legacy patterns remain in some services.
- **Error handling**: Explicit `if err != nil` checks are verbose compared to exception-based languages.
- **Runtime performance**: Go’s GC and lack of zero-cost abstractions make it slower than Rust for CPU-bound or latency-critical paths (hence the Rust carve-out).

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Rust** | Excellent for systems-level work, but compile times are long, the learning curve is steep, and the async ecosystem was less mature at project inception. Reserved for specific services where memory safety and performance are paramount. |
| **Node.js / TypeScript** | Rapid for I/O-bound prototypes, but single-threaded event loop becomes a bottleneck under CPU load, and npm dependency trees introduce supply-chain risk. Retained only for existing analytics dashboards. |
| **Java / Kotlin (JVM)** | Mature ecosystem, but JVM startup times and memory footprint conflict with our goal of fast-scaling, small-container microservices. |
| **Python** | Great for scripting and ML glue, but GIL and runtime performance make it unsuitable for high-throughput request paths. |

## References
- `services/` — Go microservices source tree
- `go.work` — Go workspace definition
- `.golangci.yml` — Linting and style rules
