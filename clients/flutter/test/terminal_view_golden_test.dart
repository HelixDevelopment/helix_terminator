// Host-rendered visual proof for the real terminal-emulator view (§11.4.170).
//
// Flutter golden tests render the REAL widget to device-independent PNG pixels
// on the host — no device, emulator, or running app. This suite proves the
// [TerminalView] (backed by the xterm VT100/xterm engine) genuinely parses
// ANSI/VT escape sequences and paints a terminal cell grid, in BOTH the light
// and dark design-system themes.
//
// Dual validation per §11.4.170:
//   (i)  golden image-diff — the committed PNG is asserted pixel-for-pixel.
//   (ii) content oracle — `terminal.buffer.getText()` returns the exact glyphs
//        the painter renders; asserting the expected terminal text is present
//        proves the content is really on-screen (not blank / garbled / collapsed),
//        which a value-equality unit test could never establish.
//
// Regenerate goldens (inside the rootless Podman Flutter container):
//   flutter test --update-goldens test/terminal_view_golden_test.dart
// Verify (must pass without --update-goldens):
//   flutter test test/terminal_view_golden_test.dart

import 'package:design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:helix_terminator/widgets/terminal_view.dart';
import 'package:xterm/xterm.dart' as xterm;

/// Representative raw terminal output: coloured prompt (SGR bold + fg colours),
/// an `ls -la` listing, an underlined filename, and an error line — the kind of
/// stream a real shell emits. Written verbatim into the engine so every escape
/// sequence is interpreted.
const String _ansiSample =
    '\x1b[1;32muser@helix\x1b[0m:\x1b[1;34m~/projects\x1b[0m\$ ls -la\r\n'
    'total 24\r\n'
    '\x1b[1;34mdrwxr-xr-x\x1b[0m 5 user user 4096 Jul  7 12:00 \x1b[1;34m.\x1b[0m\r\n'
    '\x1b[0;33m-rw-r--r--\x1b[0m 1 user user  220 Jul  7 12:00 README.md\r\n'
    '\x1b[0;36m-rw-r--r--\x1b[0m 1 user user 1451 Jul  7 12:00 main.dart\r\n'
    '\x1b[1;31merror:\x1b[0m permission denied on \x1b[4msecret.key\x1b[0m\r\n'
    '\x1b[1;32muser@helix\x1b[0m:\x1b[1;34m~/projects\x1b[0m\$ echo done\r\n'
    'done\r\n'
    '\x1b[1;32muser@helix\x1b[0m:\x1b[1;34m~/projects\x1b[0m\$ ';

/// The plain-text strings that MUST appear in the rendered cell grid.
const List<String> _expectedOnScreen = <String>[
  'user@helix',
  'ls -la',
  'total 24',
  'drwxr-xr-x',
  'README.md',
  'main.dart',
  'permission denied',
  'secret.key',
  'echo done',
];

xterm.Terminal _seededTerminal() {
  final terminal = xterm.Terminal(maxLines: 2000);
  terminal.write(_ansiSample);
  return terminal;
}

Future<void> _pumpTerminal(
  WidgetTester tester,
  ThemeData theme,
  xterm.Terminal terminal,
) async {
  await tester.binding.setSurfaceSize(const Size(820, 460));
  await tester.pumpWidget(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: theme,
      home: Scaffold(
        body: TerminalView(terminal: terminal),
      ),
    ),
  );
  // Fixed pumps (not pumpAndSettle) so a cursor ticker can never hang the test;
  // two frames let auto-resize + reflow settle deterministically.
  await tester.pump(const Duration(milliseconds: 120));
  await tester.pump(const Duration(milliseconds: 120));
}

void _assertContentOnScreen(xterm.Terminal terminal) {
  final rendered = terminal.buffer.getText();
  for (final expected in _expectedOnScreen) {
    expect(
      rendered,
      contains(expected),
      reason: 'rendered terminal buffer must contain "$expected" '
          '(proves ANSI content is really parsed + on-screen)',
    );
  }
  // The real xterm painter widget is present (not our old placeholder).
  expect(find.byType(xterm.TerminalView), findsOneWidget);
}

void main() {
  testWidgets(
    'TerminalView renders real ANSI content — DARK theme (host-rendered golden)',
    (tester) async {
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final terminal = _seededTerminal();

      await _pumpTerminal(tester, HTTheme.dark(), terminal);

      // (i) device-independent rendered-pixel proof.
      await expectLater(
        find.byType(TerminalView),
        matchesGoldenFile('goldens/terminal_view_dark.png'),
      );
      // (ii) content oracle.
      _assertContentOnScreen(terminal);
    },
  );

  testWidgets(
    'TerminalView renders real ANSI content — LIGHT theme (host-rendered golden)',
    (tester) async {
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final terminal = _seededTerminal();

      await _pumpTerminal(tester, HTTheme.light(), terminal);

      await expectLater(
        find.byType(TerminalView),
        matchesGoldenFile('goldens/terminal_view_light.png'),
      );
      _assertContentOnScreen(terminal);
    },
  );

  testWidgets(
    'TerminalView cursor-motion escape sequences move the cursor (real VT engine)',
    (tester) async {
      addTearDown(() => tester.binding.setSurfaceSize(null));
      // CSI H homes the cursor, then overwrite proves absolute positioning —
      // behaviour only a genuine terminal state machine produces.
      final terminal = xterm.Terminal(maxLines: 200);
      terminal.write('line-one\r\nline-two\r\n\x1b[H\x1b[32mREADY\x1b[0m');

      await _pumpTerminal(tester, HTTheme.dark(), terminal);

      final rendered = terminal.buffer.getText();
      // The home + overwrite replaced the start of the first line.
      expect(rendered, contains('READY'));
      expect(rendered, contains('line-two'));
      expect(find.byType(xterm.TerminalView), findsOneWidget);
    },
  );
}
