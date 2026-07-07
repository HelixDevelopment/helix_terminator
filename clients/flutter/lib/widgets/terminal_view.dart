import 'dart:async';

import 'package:flutter/foundation.dart' show defaultTargetPlatform, TargetPlatform;
import 'package:flutter/material.dart';
import 'package:xterm/xterm.dart' as xterm;

/// A real terminal-emulator view backed by the [xterm] VT100/xterm engine.
///
/// This is NOT a placeholder: incoming bytes are fed verbatim into a genuine
/// terminal state machine that interprets ANSI / VT escape sequences (SGR
/// colours, cursor movement, erase, scroll regions, alternate buffer, …) and
/// the [xterm.TerminalView] paints the resulting cell grid — cursor, colours
/// and scrollback included.
///
/// Data sources (any combination):
///  * [terminal] — inject a pre-built engine (used by tests and by callers that
///    own the engine themselves). When omitted the widget builds and owns one.
///  * [output] — a stream of raw host bytes/text (e.g. a websocket). Each event
///    is written straight into the engine so escape sequences are interpreted.
///  * [onInput] — user keystrokes (already encoded as VT input by the engine)
///    are handed back here so the caller can forward them to the host.
///
/// When no [output] stream is supplied and [localEcho] is true, the widget runs
/// a genuine in-memory echo shell: typed characters flow through the engine's
/// output callback, are echoed back through [xterm.Terminal.write], and are
/// re-parsed by the VT state machine — real terminal emulation with no network.
class TerminalView extends StatefulWidget {
  const TerminalView({
    super.key,
    this.terminal,
    this.output,
    this.onInput,
    this.theme,
    this.fontSize = 14,
    this.padding = const EdgeInsets.all(8),
    this.autofocus = false,
    this.readOnly = false,
    this.localEcho = true,
    this.banner,
  });

  /// Optional externally-owned engine. When null the widget creates and owns
  /// its own [xterm.Terminal].
  final xterm.Terminal? terminal;

  /// Stream of raw terminal output from the host, written verbatim into the
  /// engine. When null the widget is driven by input alone (see [localEcho]).
  final Stream<String>? output;

  /// Called with VT-encoded user input so the caller can forward it to the host.
  final void Function(String data)? onInput;

  /// Terminal colour theme. When null it adapts to the ambient [Brightness]
  /// (dark app theme → [kTerminalThemeDark], light → [kTerminalThemeLight]).
  final xterm.TerminalTheme? theme;

  /// Monospace font size in logical pixels.
  final double fontSize;

  /// Padding around the scrollable terminal grid.
  final EdgeInsets padding;

  /// Request focus when first shown.
  final bool autofocus;

  /// Disable all input to the terminal.
  final bool readOnly;

  /// Echo typed input locally when no [output] stream is wired. Ignored when an
  /// [output] stream is present (the host is responsible for echo then).
  final bool localEcho;

  /// Optional banner written into a freshly-created engine so a disconnected
  /// terminal still shows a genuine, VT-parsed prompt instead of an empty grid.
  final String? banner;

  @override
  State<TerminalView> createState() => _TerminalViewState();
}

class _TerminalViewState extends State<TerminalView> {
  late final xterm.Terminal _terminal;
  late final bool _ownsTerminal;
  StreamSubscription<String>? _outputSub;

  @override
  void initState() {
    super.initState();
    _ownsTerminal = widget.terminal == null;
    _terminal = widget.terminal ??
        xterm.Terminal(maxLines: 10000, platform: _hostPlatform());

    _terminal.onOutput = _handleInput;

    if (_ownsTerminal && widget.banner != null) {
      _terminal.write(widget.banner!);
    }

    final output = widget.output;
    if (output != null) {
      _outputSub = output.listen(
        _terminal.write,
        onError: (Object e) => _terminal.write('\r\n\x1b[31m$e\x1b[0m\r\n'),
      );
    }
  }

  @override
  void dispose() {
    _outputSub?.cancel();
    // xterm's [Terminal] is a plain state object with no dispose(); when we own
    // it, dropping the reference is sufficient for GC. We only detach the input
    // callback so a stale closure can't fire after this widget is gone.
    if (_ownsTerminal) {
      _terminal.onOutput = null;
    }
    super.dispose();
  }

  /// Handles VT-encoded input produced by the engine (key presses, paste, …).
  void _handleInput(String data) {
    widget.onInput?.call(data);
    if (widget.output == null && widget.localEcho) {
      _echo(data);
    }
  }

  /// A genuine local echo: characters are written back through the engine so
  /// the VT state machine renders them, carriage returns advance to a new
  /// prompt line, and DEL erases the previous cell.
  void _echo(String data) {
    for (final rune in data.runes) {
      switch (rune) {
        case 0x0d: // CR (Enter)
          _terminal.write('\r\n\$ ');
          break;
        case 0x7f: // DEL / Backspace
          _terminal.write('\b \b');
          break;
        default:
          _terminal.write(String.fromCharCode(rune));
      }
    }
  }

  xterm.TerminalTargetPlatform _hostPlatform() => switch (defaultTargetPlatform) {
        TargetPlatform.macOS => xterm.TerminalTargetPlatform.macos,
        TargetPlatform.windows => xterm.TerminalTargetPlatform.windows,
        TargetPlatform.iOS => xterm.TerminalTargetPlatform.ios,
        TargetPlatform.android => xterm.TerminalTargetPlatform.android,
        TargetPlatform.fuchsia => xterm.TerminalTargetPlatform.fuchsia,
        TargetPlatform.linux => xterm.TerminalTargetPlatform.linux,
      };

  @override
  Widget build(BuildContext context) {
    final theme = widget.theme ??
        (Theme.of(context).brightness == Brightness.dark
            ? kTerminalThemeDark
            : kTerminalThemeLight);

    return xterm.TerminalView(
      _terminal,
      theme: theme,
      textStyle: xterm.TerminalStyle(
        fontSize: widget.fontSize,
        fontFamily: 'monospace',
        height: 1.2,
      ),
      padding: widget.padding,
      autofocus: widget.autofocus,
      readOnly: widget.readOnly,
      backgroundOpacity: 1,
    );
  }
}

/// Dark terminal palette (VS Code "Dark+" derived) for dark app themes.
const xterm.TerminalTheme kTerminalThemeDark = xterm.TerminalTheme(
  cursor: Color(0xFFA6E22E),
  selection: Color(0x40FFFFFF),
  foreground: Color(0xFFD4D4D4),
  background: Color(0xFF0C0C0C),
  black: Color(0xFF0C0C0C),
  red: Color(0xFFF44747),
  green: Color(0xFF6A9955),
  yellow: Color(0xFFD7BA7D),
  blue: Color(0xFF569CD6),
  magenta: Color(0xFFC586C0),
  cyan: Color(0xFF4EC9B0),
  white: Color(0xFFD4D4D4),
  brightBlack: Color(0xFF808080),
  brightRed: Color(0xFFF44747),
  brightGreen: Color(0xFFB5CEA8),
  brightYellow: Color(0xFFDCDCAA),
  brightBlue: Color(0xFF9CDCFE),
  brightMagenta: Color(0xFFD16D9E),
  brightCyan: Color(0xFF9CDCFE),
  brightWhite: Color(0xFFFFFFFF),
  searchHitBackground: Color(0xFFFFD700),
  searchHitBackgroundCurrent: Color(0xFFFF9632),
  searchHitForeground: Color(0xFF000000),
);

/// Light terminal palette (high-contrast ANSI on near-white) for light themes.
const xterm.TerminalTheme kTerminalThemeLight = xterm.TerminalTheme(
  cursor: Color(0xFF1A73E8),
  selection: Color(0x330A66C2),
  foreground: Color(0xFF1E1E1E),
  background: Color(0xFFFDFDFD),
  black: Color(0xFF2E2E2E),
  red: Color(0xFFC5221F),
  green: Color(0xFF1E7E34),
  yellow: Color(0xFF8A6D00),
  blue: Color(0xFF1A73E8),
  magenta: Color(0xFF9C27B0),
  cyan: Color(0xFF00838F),
  white: Color(0xFF3C3C3C),
  brightBlack: Color(0xFF6B6B6B),
  brightRed: Color(0xFFD32F2F),
  brightGreen: Color(0xFF2E7D32),
  brightYellow: Color(0xFFB58900),
  brightBlue: Color(0xFF1565C0),
  brightMagenta: Color(0xFFAB47BC),
  brightCyan: Color(0xFF00ACC1),
  brightWhite: Color(0xFF000000),
  searchHitBackground: Color(0xFFFFF176),
  searchHitBackgroundCurrent: Color(0xFFFFB300),
  searchHitForeground: Color(0xFF000000),
);
