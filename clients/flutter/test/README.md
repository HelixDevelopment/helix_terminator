# HelixTerminator Flutter — Test Suite

**Status:** every test file under `test/` and `integration_test/` exercises
real widget/bloc/model/HTTP behaviour and asserts real outcomes. The former
`expect(true, isTrue)` stubs (§11.4 / §11.4.27 anti-bluff covenant — a test
that passes without exercising the tested behaviour is a critical defect)
have been replaced. Writing these real tests also surfaced and fixed two
genuine, pre-existing product defects (below) — the exact failure mode the
anti-bluff covenant exists to prevent: a stub suite that never actually ran
had let real bugs ship unnoticed.

**Captured evidence (this pass, `ghcr.io/cirruslabs/flutter:3.24.0`,
rootless Podman):** `flutter analyze` → 0 issues, both before and after the
fixes below. `flutter test` → **86/86 passing**, run twice consecutively
with identical `+86: All tests passed!` results (determinism per §11.4.50).

## Real defects found and fixed while writing these tests

1. **Error messages shown to users were meaningless boilerplate.**
   `ApiException`, `AuthException`, `HostServiceException`,
   `VaultServiceException`, `WorkspaceServiceException`, and
   `NotificationServiceException` never overrode `toString()`. Every bloc's
   catch handler renders its caught error via `e.toString()` or `'...: $e'`
   string interpolation — without the override, Dart's default
   `Object.toString()` produced `Instance of 'HostServiceException'`
   (literally that string) instead of the real message, e.g. "name taken" /
   "vault sealed" / "Unauthorized". First caught as RED test failures in
   `host_bloc_test.dart`, `vault_bloc_test.dart`, and
   `workspace_bloc_test.dart`'s failure-path assertions; root-caused and
   fixed with a one-line `@override String toString() => message;` on each
   class (`lib/services/api_client.dart`, `lib/services/auth_service.dart`,
   `lib/services/host_service.dart`, `lib/services/vault_service.dart`,
   `lib/services/workspace_service.dart`, `lib/services/notification_service.dart`)
   rather than weakening the tests to match the broken output
   (§11.4.1/§11.4.4 — FAIL-bluffs and test-interrupt-on-discovery).
2. **A real `RenderFlex` overflow on `LoginScreen`.** The "Don't have an
   account? Register" row (`lib/screens/login_screen.dart`) overflowed its
   400px-constrained parent by 58 logical pixels — an unconstrained `Text`
   plus a `TextButton` whose combined natural width exceeds the available
   space, with no `Flexible`/`Expanded` to absorb it. In debug this renders
   the yellow/black overflow stripe; in release, content is silently
   clipped — a genuine user-visible defect (the "Register" affordance could
   be partially unreachable). Caught by
   `test/app_boot_journey_test.dart` pumping the real `LoginScreen`;
   root-caused and fixed by wrapping the `Text` in `Flexible(...,
   overflow: TextOverflow.ellipsis)` so the descriptive text shrinks and
   the button stays fully visible/tappable.

## What is covered, and how

| File | Real coverage |
|---|---|
| `test/api_client_test.dart` | `ApiClient` GET/POST/PUT/DELETE over a real HTTP request/response cycle (via `package:http`'s `MockClient` — no live network, but every line of `ApiClient`'s own request-building/response-parsing/error-handling logic runs for real): header construction (`Content-Type`, conditional `Authorization: Bearer <token>`), JSON body encoding, empty-body handling, and `ApiException` (message + statusCode) on non-2xx responses. |
| `test/auth_bloc_test.dart` | `AuthBloc` event → state transitions against a mocked `AuthService` (`mocktail`): login success / 2FA-required / failure, register, logout, 2FA verification, and the `AuthCheckRequested` boot-time session check (including its "never call getCurrentUser without a session" and "storage failure still resolves cleanly" branches). Also a dedicated `User.fromJson`/`toJson` round-trip group, including the snake_case wire-format assertion and a null-optional-field case. |
| `test/collaboration_bloc_test.dart` | `CollaborationBloc` against a mocked `CollaborationService`: list/create/join/leave/end session, participants refresh, and the `state is CollaborationActive` guard proven both ways (no-op when inactive, real update when active). |
| `test/host_bloc_test.dart` | `HostBloc` against a mocked `HostService`: load/refresh (including `previousHosts` preservation across a reload), create (list append), update (in-place replace by id), delete (removal by id) — plus `Host.==`/`copyWith` model tests. Uses `isA<T>().having(...)` property matchers rather than raw `HostState.==`, because `HostLoaded`/`HostError`/`HostOperationSuccess` compare their `List<Host>` field with Dart's identity-based `List.==`; `having(..., equals(...))` performs a real structural comparison instead, which is what actually catches regressions. Its failure-path test is what first caught defect #1 above. |
| `test/notification_bloc_test.dart` | `NotificationBloc` against a mocked `NotificationService`: list/filter/mark-as-read (+auto-reload)/mark-all-as-read/delete, and the client-side `NotificationSearchChanged` guard (no-op unless already `Loaded`, else updates `searchQuery` while preserving the rest of the state). |
| `test/terminal_bloc_test.dart` | `TerminalBloc` against a hand-written `FakeTerminalService` (a subclass of the concrete, WebSocket-backed `TerminalService` with its public methods overridden — no mocking framework needed, no source change needed). Covers connect success/failure, message-append while connected vs. the no-op guard while not connected, command send success/failure (including state restoration after a send error), the resize no-op, disconnect, and `close()` disposal. A dedicated integration-style test proves the *real* `onMessage` callback wiring installed by `_onConnectRequested` — an inbound "server" message really flows through `bloc.add()` into a new emitted state, not a synthetic shortcut. |
| `test/vault_bloc_test.dart` | `VaultBloc` against a mocked `VaultService`: load, and the actual (slightly unusual) 4-state create/update/delete sequences (`Loading → Loaded → OperationSuccess → Loaded`) — asserted explicitly so a future refactor that collapses or reorders that sequence is caught. Plus `Secret.copyWith` model test. |
| `test/workspace_bloc_test.dart` | `WorkspaceBloc` against a mocked `WorkspaceService`: load, add-member (+chained reload), and — the one thing a stub could never catch — the *conditional* trailing re-emit in `WorkspaceCreateRequested` (an extra `Loaded` is emitted only when the bloc was already `Loaded` before the event). Both branches of that condition are exercised explicitly. Plus `Workspace.copyWith` model test. |
| `test/terminal_view_golden_test.dart` | *(pre-existing, unmodified)* Host-rendered §11.4.170 golden-image + content-oracle proof for the real `TerminalView` widget (backed by the real `xterm` VT100 engine) in both light and dark theme. Reference pattern for future screen-level golden coverage — see the gap below. |
| `test/widget_test.dart` | *(pre-existing — real, not a stub; one fix landed here)* Pumps the real `HelixTerminatorApp` and asserts the splash screen's `CircularProgressIndicator` is genuinely rendered. Fixed: the original version never drained `SplashScreen`'s real `Future.delayed(2s)` timer, so the Flutter test binding's post-test invariant check ("A Timer is still pending even after the widget tree was disposed.") failed it for a test-harness reason, not a product defect (§11.4.1) — added the same discrete `pump(duration)` sequence used elsewhere in this suite. |
| `test/app_boot_journey_test.dart` | **New — the real, EXECUTED equivalent of the e2e journey.** Pumps the real `SplashScreen` + real `AuthBloc` + real `LoginScreen` (wired exactly like `lib/main.dart`'s provider tree), with a mocked `AuthService` fed into `AuthBloc`'s existing, already-real constructor seam (the same technique `auth_bloc_test.dart` uses) so the journey never touches `flutter_secure_storage`. Covers: splash → real `Navigator.pushReplacement` → `LoginScreen` on the unauthenticated path; empty-submit + malformed-input `Form` validation (and proves `AuthService.login` is never called on client-side validation failure); the password-visibility toggle really flipping `TextField.obscureText`; and a real login attempt calling `AuthService.login` with exactly what the user typed. This suite is what caught defect #2 above. |
| `integration_test/app_test.dart` | Real end-to-end test — rewritten from the former `'end-to-end test stub'`. Drives the real `app.main()` entry point (real DI graph from `lib/main.dart`, no synthetic shortcut). **Confirmed UNEXECUTED in this environment — see below, honestly, not silently.** |

## Running

Inside the project's cached Flutter container (rootless Podman, plain
default userns):

```bash
podman run --rm \
  -e PUB_CACHE=/app/.pub-cache \
  -v "$(pwd)/clients/flutter:/app" \
  -w /app \
  ghcr.io/cirruslabs/flutter:3.24.0 \
  bash -lc 'flutter pub get && flutter analyze && flutter test'
```

(`-e PUB_CACHE=/app/.pub-cache` persists the resolved package cache inside
the mounted volume across separate `podman run` invocations — without it,
each fresh container has an empty pub cache even though
`.dart_tool/package_config.json` on the host still lists the resolved
paths, which otherwise breaks `flutter analyze`/`flutter test` run as a
follow-up command with "Target of URI doesn't exist" errors.)

To run only the widget/bloc/unit suite (`test/`, fast, headless, no device
— this is the 86-test, fully-real suite):

```bash
flutter test
```

To attempt the real on-device end-to-end journey (`integration_test/`) —
see "Confirmed environment limitations" below for why this does not launch
in this container:

```bash
flutter test integration_test/app_test.dart
```

To regenerate the `TerminalView` goldens after a deliberate visual change:

```bash
flutter test --update-goldens test/terminal_view_golden_test.dart
```

## Confirmed environment limitations (§11.4.3 SKIP-with-reason — verified by actually attempting them, not guessed)

`integration_test/app_test.dart` could not be executed in this pass, for
two independently-confirmed reasons:

1. **No platform-runner scaffold is tracked in git.** `git ls-tree
   HEAD:clients/flutter` lists only `design_system/`, `integration_test/`,
   `lib/`, `test/`, `pubspec.lock`, `pubspec.yaml`, and `README.md` — there
   is no `android/`, `ios/`, `linux/`, `macos/`, or `web/` folder committed.
   The `integration_test` package requires a real platform target (a
   connected device, or a `flutter create`-generated desktop/web runner) to
   launch. Running it produced:
   ```
   No supported devices connected.
   The following devices were found, but are not supported by this project:
   Linux (desktop) • linux • linux-x64 • Ubuntu 24.04 LTS ...
   If you would like your app to run on linux, consider running `flutter create .`
   No devices are connected. Ensure that `flutter doctor` shows at least one connected device
   ```
   Generating a full platform-runner scaffold (dozens of new files: CMake
   config, native runner boilerplate, Info.plist, AndroidManifest.xml,
   etc.) is a real, non-trivial structural change to the project outside
   this pass's stated scope (test files primarily, minimal non-test edits
   only where strictly needed for testability).
2. **Independently, `flutter_secure_storage` hangs forever in this headless
   container.** A diagnostic run pumping the real `HelixTerminatorApp()`
   (real `AuthService` backed by real `flutter_secure_storage`) and polling
   every 200ms for 6 seconds observed the splash spinner never clear and
   zero exceptions thrown — `AuthService.isAuthenticated()`'s
   `flutter_secure_storage` Linux backend talks to a D-Bus
   secret-service/keyring daemon that does not exist in this container, and
   the call never times out. On a real device this backend is normally
   present, so this is expected to be a container-only artifact, but that
   is stated as an untested fact, not assumed.

`test/app_boot_journey_test.dart` (see the table above) closes the
practical gap this leaves: it exercises the identical production
`SplashScreen → AuthBloc → LoginScreen` journey for real, executed, GREEN,
right now, by feeding `AuthBloc`'s existing constructor seam a mocked
`AuthService` instead of reaching through the unavailable
`flutter_secure_storage` I/O. `integration_test/app_test.dart` remains the
correct, real, ready-to-run artifact for true on-device E2E once a platform
target exists — its header comment documents both limitations in full for
whoever picks this up next.

## Honest gaps (not silently hidden — §11.4.6 / §11.4.107)

- **No host-rendered §11.4.170 golden coverage yet for any full *screen***
  (`LoginScreen`, `SplashScreen`, `DashboardScreen`, `VaultListScreen`,
  `WorkspaceListScreen`, etc.) — only `TerminalView` (a single widget) has a
  golden pair today. The `app_boot_journey_test.dart` / `integration_test/
  app_test.dart` `LoginScreen` assertions in this pass are *finder-based*
  (widget/text presence + a real stateful-toggle check + a real overflow
  assertion the rendering engine throws for free), which is real proof the
  correct widgets are on screen and not overflowing, but it is **not** the
  device-independent rendered-pixel + OCR dual validation §11.4.170
  mandates for a UI-surface change. Adding `matchesGoldenFile` coverage
  (light + dark) for each screen is the follow-up work item, using
  `test/terminal_view_golden_test.dart` as the template.
- **Most models have no `fromJson`/`toJson` of their own** — `Host`,
  `Secret`, `Workspace`, `Session`, and `Notification` are all
  (de)serialized by hand inside their respective `*Service` classes
  (`_hostFromJson`, `_secretFromJson`, …), not via a method on the model.
  Only `User` has real `fromJson`/`toJson` (covered by the round-trip group
  in `auth_bloc_test.dart`). The bloc tests in this pass mock the
  `*Service` layer directly (the correct isolation boundary for a bloc
  unit test) rather than reaching through to the private JSON mappers, so
  those private mapping functions are currently untested in isolation.
  Promoting them to public `Model.fromJson`/`toJson` methods (mirroring
  `User`) would let each model carry its own round-trip test; that is a
  small, real source change outside this pass's stated scope.
- **`app_boot_journey_test.dart` / `integration_test/app_test.dart` only
  exercise the unauthenticated boot path.** The authenticated path
  (`AuthAuthenticated` → real `DashboardScreen`) and 2FA/error paths are
  covered at the *bloc* level in `auth_bloc_test.dart`, but not yet driven
  through a full widget-tree journey.
- **`Notification` has no `fromJson`/`toJson` at all** (only the private
  `_notificationFromJson` inside `NotificationService`) — flagged in the
  source as `// TODO: add fromJson, toJson`. Untouched by this pass; the
  `NotificationBloc` tests mock `NotificationService` directly so this gap
  does not block bloc-layer coverage, but it does mean there is currently no
  test proving that mapper's own JSON contract.
- **The `RenderFlex` overflow fix (defect #2) was found on exactly one
  Row on `LoginScreen`.** Other screens (`RegisterScreen`,
  `DashboardScreen`, the 25+ other screens under `lib/screens/`) have not
  been pumped by any test in this pass and may have similar undiscovered
  layout defects at a 400px (or other) constrained width — this pass makes
  no claim about them either way.
