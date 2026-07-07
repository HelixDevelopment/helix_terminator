# 06 — Client Specification

**Status:** `Draft`  
**Module:** A (Secure Terminal Platform)  
**Authority:** `CANONICAL_FACTS.md` (CD-4: Flutter 3.24)  

---

## Overview

HelixTerminator's client is a cross-platform Flutter/Dart application targeting 6 platforms: Web (WASM), macOS, Windows, Linux, iOS, and Android. It follows a strict BLoC pattern with unidirectional data flow.

| Platform | Target | Notes |
|----------|--------|-------|
| Web (WASM) | Primary | Browser-based terminal via WebAssembly |
| macOS | Primary | Native desktop with Secure Enclave key storage |
| Windows | Primary | Native desktop with DPAPI key storage |
| Linux | Primary | Native desktop with kernel keyring |
| iOS | Primary | Mobile with Secure Enclave, biometric auth |
| Android | Primary | Mobile with Android Keystore, biometric auth |

---

## Architecture

### Layered BLoC Pattern

```
┌─────────────────────────────────────────┐
│         Presentation Layer              │
│    Flutter Widgets → BLoC/Cubit       │
├─────────────────────────────────────────┤
│           Domain Layer                  │
│    Use Cases → Domain Entities          │
├─────────────────────────────────────────┤
│            Data Layer                   │
│  Repositories → Remote API / Local DB   │
├─────────────────────────────────────────┤
│        Infrastructure Layer           │
│   Platform Channels / Secure Storage    │
└─────────────────────────────────────────┘
```

**Core Principle:** UI widgets never call data sources directly. Every widget observes a BLoC. Every BLoC calls a repository interface. Concrete implementations are injected via `get_it`.

### Project Structure

```
lib/
├── core/
│   ├── di/                    # Dependency injection (get_it + injectable)
│   ├── navigation/            # go_router configuration
│   ├── network/               # Dio client + interceptors
│   ├── storage/               # Drift database + secure storage
│   ├── error/                 # Failures and exceptions
│   └── utils/
├── features/
│   ├── auth/
│   ├── vault/
│   ├── hosts/
│   ├── terminal/
│   ├── ssh_session/
│   ├── sftp/
│   ├── port_forwarding/
│   ├── workspace/
│   ├── snippets/
│   ├── keychain/
│   ├── collaboration/
│   ├── ai_autocomplete/
│   ├── settings/
│   └── organizations/
└── main.dart
```

---

## Terminal Emulator

- Custom terminal emulator built on `xterm.js` principles, ported to Dart/Flutter
- Supports xterm-256color, truecolor, sixel graphics
- Scrollback buffer: configurable (default 10,000 lines)
- Font rendering: Impeller on iOS/macOS, Skia elsewhere
- Input latency target: < 16ms keystroke-to-screen

---

## Offline Mode

- SQLite (Drift) local database caches hosts, vault metadata, snippets
- Vault items encrypted locally with AES-256-GCM
- Sync queue for pending operations; reconciled on reconnection
- CRDT-based conflict resolution for collaborative buffers

---

## Security on Client

- **Secure Storage:** Platform-native secure enclaves (Keychain/Keystore/DPAPI/kernel keyring)
- **Biometric Auth:** Face ID / Touch ID / fingerprint / Windows Hello
- **Vault Encryption:** Client-side AES-256-GCM with Argon2id key derivation
- **Certificate Pinning:** TLS cert pinning for API endpoints
- **Anti-tampering:** Runtime integrity checks, debug detection

---

## Performance Targets

| Metric | Target |
|--------|--------|
| Cold start (mobile) | < 2s |
| Cold start (desktop) | < 1s |
| Terminal keystroke latency | < 16ms |
| SSH connection establish | < 500ms |
| SFTP file list (1000 files) | < 200ms |
| Frame rate | 60fps minimum |
| Memory footprint (idle) | < 150MB desktop, < 80MB mobile |

---

## Platform-Specific Features

| Feature | macOS | Windows | Linux | iOS | Android | Web |
|---------|-------|---------|-------|-----|---------|-----|
| Secure Enclave / Keystore | Keychain | DPAPI | keyring | Secure Enclave | Keystore | Web Crypto |
| Biometric Auth | Touch ID | Hello | — | Face/Touch ID | Fingerprint | — |
| Native Menubar | Yes | Yes | Yes | — | — | — |
| System Tray | Yes | Yes | Yes | — | — | — |
| Background Execution | Yes | Yes | Yes | Limited | Limited | — |
| Push Notifications | — | — | — | APNS | FCM | — |
| Deep Links | Yes | Yes | Yes | Yes | Yes | — |

---

## Testing Strategy (Client)

| Test Type | Framework | Coverage Target |
|-----------|-----------|-------------------|
| Unit tests | `flutter_test` + `mockito` | ≥ 80% |
| Widget tests | `flutter_test` + `golden_toolkit` | ≥ 80% |
| Integration tests | `integration_test` | Critical paths |
| Device tests | Firebase Test Lab | iOS + Android |
| Accessibility | `flutter_test` a11y matchers | WCAG 2.1 AA |

---

## Diagrams

| Diagram | Source |
|---------|--------|
| Client Architecture (Draw.io) | `diagrams/drawio/08_client_architecture.drawio` |
| Flutter BLoC Class Diagram | `diagrams/mermaid/24_flutter_bloc.mmd` |
| SSH Connection Flow (Draw.io) | `diagrams/drawio/02_ssh_connection_flow.drawio` |

---

## Cross-References

- [02 — System Architecture](../02-system-architecture/) — C4 diagrams, deployment topology
- [04 — API Specification](../04-api-specification/) — REST endpoints consumed by client
- [10 — UX Design System](../10-ux-design-system/) — Design tokens, components, wireframes
- [16 — References](../16-references/) — Canonical facts (CD-4 Flutter 3.24)

---

*Section 06 — Client Specification*  
*Consolidated from: 02_client_specification.md, CANONICAL_FACTS.md (CD-4)*
