# ADR-006: CRDTs over Operational Transformation for Real-Time Collaboration

## Status
Accepted

## Context
The collaboration service in helix_terminator supports real-time shared editing of project documents, diagrams, and configuration. Multiple users may edit concurrently with intermittent connectivity and variable latency. We need a conflict resolution mechanism that is eventually consistent, partition-tolerant, and does not require a central coordination server for every edit.

## Decision
We chose **Conflict-free Replicated Data Types (CRDTs)** over **Operational Transformation (OT)** for real-time collaboration state synchronization.

## Consequences

### Positive
- **No central sequencer**: CRDTs converge without a single server ordering operations, improving availability and reducing latency for global users.
- **Partition tolerance**: Users can continue editing offline; local changes merge automatically when connectivity resumes.
- **Simpler correctness**: CRDT merge functions are mathematically guaranteed to converge; OT requires complex transformation functions that are error-prone.
- **Scalability**: State can be replicated across regional caches and edge nodes without strong consistency requirements.

### Negative
- **Memory overhead**: Some CRDT structures (e.g., sequence CRDTs with tombstones) grow unbounded; garbage collection and compression are required.
- **Initial complexity**: Implementing or integrating a robust CRDT library (e.g., Yjs, Automerge) requires upfront investment.
- **Undo/redo semantics**: Global undo is more complex in CRDTs because operations are not totally ordered; user-visible undo is scoped to local intent.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Operational Transformation (OT)** | The classic choice (e.g., Google Docs), but requires a central server to transform operations against each other in real time. This creates a single point of latency and failure; complex transformation functions are notoriously difficult to implement correctly. |
| **Event Sourcing with Central Aggregate** | Provides strong consistency but reintroduces the central bottleneck and complicates offline support. |
| **Last-Write-Wins (LWW) Registers** | Too simplistic; concurrent edits to structured documents would silently lose data. |
| **Delta CRDTs (pure state-based)** | Considered as a variant; we chose operation-based CRDTs with delta-state anti-entropy for bandwidth efficiency. |

## References
- `services/collaboration-service/` — Real-time collaboration backend
- `clients/flutter/lib/crdt/` — Flutter CRDT integration layer
- `docs/guides/runbooks/KAFKA_RECOVERY.md` — Event bus recovery related to collaboration sync
