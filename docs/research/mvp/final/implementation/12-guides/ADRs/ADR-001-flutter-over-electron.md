# ADR-001: Flutter over Electron for Cross-Platform Desktop

## Status
Accepted

## Context
The helix_terminator project requires a cross-platform desktop client that delivers native-grade performance, a consistent UI/UX across Windows, macOS, and Linux, and long-term maintainability. The client must integrate with backend microservices via gRPC-Web and support offline-first workflows with local SQLite caching.

## Decision
We chose **Flutter** (with `flutter_rust_bridge` for performance-critical modules) over **Electron** for the desktop client.

## Consequences

### Positive
- **Native performance**: Flutter compiles to native machine code via LLVM, avoiding the memory and CPU overhead of bundling a full Chromium runtime.
- **Single UI codebase**: One Dart codebase renders consistently across all three desktop targets, reducing UI drift and maintenance burden.
- **Mobile reuse**: The same Dart codebase can target iOS and Android with minimal changes, preserving optionality for a future mobile release.
- **Custom rendering**: Flutter’s Skia-based renderer allows pixel-perfect control over theming and animations, independent of OS-level UI toolkits.
- **Binary size**: AOT-compiled Flutter desktop binaries are smaller than equivalent Electron apps (no embedded browser engine).

### Negative
- **Learning curve**: The team must adopt Dart/Flutter; existing web/React expertise does not directly transfer.
- **Ecosystem gaps**: Some desktop-native integrations (global hotkeys, menubar tray, deep OS notifications) require platform-channel plugins or FFI bridges.
- **WebView limitations**: If we ever need to embed arbitrary web content, Flutter’s webview support is less mature than Electron’s `<webview>` tag.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **Electron** | Bundles Chromium + Node.js, leading to large binary sizes (~150 MB+), high memory usage, and slower startup times. Also encourages web-tech stack fragmentation when mobile is later required. |
| **Tauri** | Promising Rust-based alternative with smaller binaries, but ecosystem maturity and plugin availability were insufficient at the time of decision; team lacked Rust GUI expertise. |
| **Native per-platform (Swift, WPF, GTK)** | Maximally performant and integrated, but triples UI development and testing effort; rejected due to velocity constraints. |

## References
- `clients/flutter/` — Flutter client source tree
- `docs/guides/flutter_desktop_setup.md` — Build and run instructions
