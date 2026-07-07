import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:xterm/xterm.dart' as xterm;

import '../bloc/terminal_bloc.dart';
import '../widgets/connection_status.dart';
import '../widgets/terminal_view.dart';

/// Full terminal screen with an emulator view, command input, status bar,
/// and toolbar. Supports color themes and realistic terminal appearance.
class TerminalScreen extends StatefulWidget {
  final String hostId;
  final String hostName;

  const TerminalScreen({
    super.key,
    required this.hostId,
    required this.hostName,
  });

  @override
  State<TerminalScreen> createState() => _TerminalScreenState();
}

class _TerminalScreenState extends State<TerminalScreen> {
  final TextEditingController _inputController = TextEditingController();
  final FocusNode _inputFocusNode = FocusNode();
  // The real VT100/xterm engine that parses host output into a cell grid.
  final xterm.Terminal _terminal = xterm.Terminal(maxLines: 10000);
  // Number of bloc output entries already fed into the engine (delta cursor).
  int _writtenLineCount = 0;
  bool _isFullscreen = false;

  // Terminal color themes.
  final List<Map<String, dynamic>> _themes = [
    {'name': 'Dark', 'bg': const Color(0xFF0C0C0C), 'fg': const Color(0xFFCCCCCC)},
    {'name': 'Solarized', 'bg': const Color(0xFF002B36), 'fg': const Color(0xFF839496)},
    {'name': 'Monokai', 'bg': const Color(0xFF272822), 'fg': const Color(0xFFF8F8F2)},
  ];
  int _currentThemeIndex = 0;

  Color get _backgroundColor => _themes[_currentThemeIndex]['bg'] as Color;
  Color get _foregroundColor => _themes[_currentThemeIndex]['fg'] as Color;

  @override
  void initState() {
    super.initState();
    context.read<TerminalBloc>().add(TerminalConnectRequested(widget.hostId));
  }

  @override
  void dispose() {
    _inputController.dispose();
    _inputFocusNode.dispose();
    super.dispose();
  }

  void _sendCommand() {
    final text = _inputController.text;
    if (text.isEmpty) return;
    context.read<TerminalBloc>().add(TerminalSendCommand('$text\r\n'));
    _inputController.clear();
    _inputFocusNode.requestFocus();
  }

  /// Feeds any bloc output entries not yet written into the [xterm.Terminal]
  /// engine, so escape sequences are interpreted exactly once each. Resets the
  /// engine when the output buffer shrinks (fresh connection).
  void _syncTerminal(List<String> lines) {
    if (lines.length < _writtenLineCount) {
      _writtenLineCount = 0;
    }
    for (var i = _writtenLineCount; i < lines.length; i++) {
      _terminal.write(lines[i]);
    }
    _writtenLineCount = lines.length;
  }

  void _toggleTheme() {
    setState(() {
      _currentThemeIndex = (_currentThemeIndex + 1) % _themes.length;
    });
  }

  void _toggleFullscreen() {
    setState(() => _isFullscreen = !_isFullscreen);
  }

  @override
  Widget build(BuildContext context) {
    return BlocListener<TerminalBloc, TerminalState>(
      listener: (context, state) {
        if (state is TerminalConnected) {
          _syncTerminal(state.outputLines);
        }
      },
      child: BlocBuilder<TerminalBloc, TerminalState>(
        builder: (context, state) {
          final isConnected = state is TerminalConnected;

          return Scaffold(
            backgroundColor: _backgroundColor,
            appBar: _isFullscreen
                ? null
                : AppBar(
                    backgroundColor: _backgroundColor,
                    foregroundColor: _foregroundColor,
                    title: Text(widget.hostName),
                    actions: [
                      ConnectionStatus(connected: isConnected),
                      const SizedBox(width: 12),
                      IconButton(
                        icon: const Icon(Icons.palette),
                        tooltip: 'Theme: ${_themes[_currentThemeIndex]['name']}',
                        onPressed: _toggleTheme,
                      ),
                      IconButton(
                        icon: Icon(_isFullscreen ? Icons.fullscreen_exit : Icons.fullscreen),
                        tooltip: _isFullscreen ? 'Exit fullscreen' : 'Fullscreen',
                        onPressed: _toggleFullscreen,
                      ),
                      IconButton(
                        icon: const Icon(Icons.settings),
                        tooltip: 'Settings',
                        onPressed: () {
                          // Navigate to terminal settings.
                        },
                      ),
                      IconButton(
                        icon: const Icon(Icons.power_settings_new),
                        tooltip: 'Disconnect',
                        onPressed: () {
                          context.read<TerminalBloc>().add(TerminalDisconnectRequested());
                          Navigator.of(context).maybePop();
                        },
                      ),
                      const SizedBox(width: 8),
                    ],
                  ),
            body: Column(
              children: [
                // Real terminal emulator view (xterm VT100/xterm engine).
                Expanded(
                  child: TerminalView(
                    terminal: _terminal,
                    theme: _terminalTheme(),
                    onInput: (data) => context
                        .read<TerminalBloc>()
                        .add(TerminalSendCommand(data)),
                  ),
                ),
                // Status bar.
                Container(
                  color: _backgroundColor.withOpacity(0.9),
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                  child: Row(
                    children: [
                      Icon(
                        Icons.circle,
                        size: 10,
                        color: isConnected ? Colors.green : Colors.red,
                      ),
                      const SizedBox(width: 6),
                      Text(
                        isConnected ? 'Connected' : 'Disconnected',
                        style: TextStyle(
                          color: _foregroundColor,
                          fontSize: 12,
                          fontFamily: 'monospace',
                        ),
                      ),
                      const Spacer(),
                      Text(
                        '${widget.hostId}',
                        style: TextStyle(
                          color: _foregroundColor.withOpacity(0.6),
                          fontSize: 12,
                          fontFamily: 'monospace',
                        ),
                      ),
                    ],
                  ),
                ),
                // Command input.
                Container(
                  color: _backgroundColor.withOpacity(0.95),
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  child: Row(
                    children: [
                      Text(
                        '> ',
                        style: TextStyle(
                          color: _foregroundColor,
                          fontFamily: 'monospace',
                          fontSize: 14,
                        ),
                      ),
                      Expanded(
                        child: TextField(
                          controller: _inputController,
                          focusNode: _inputFocusNode,
                          style: TextStyle(
                            color: _foregroundColor,
                            fontFamily: 'monospace',
                            fontSize: 14,
                          ),
                          decoration: InputDecoration(
                            hintText: 'Enter command...',
                            hintStyle: TextStyle(
                              color: _foregroundColor.withOpacity(0.4),
                              fontFamily: 'monospace',
                            ),
                            border: InputBorder.none,
                            contentPadding: EdgeInsets.zero,
                          ),
                          onSubmitted: (_) => _sendCommand(),
                          cursorColor: _foregroundColor,
                        ),
                      ),
                      IconButton(
                        icon: Icon(Icons.send, color: _foregroundColor.withOpacity(0.7)),
                        onPressed: _sendCommand,
                        tooltip: 'Send',
                      ),
                    ],
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  /// Builds an [xterm.TerminalTheme] from the currently-selected screen palette
  /// so the palette toggle recolours the terminal background + foreground while
  /// the ANSI colour set stays consistent and legible.
  xterm.TerminalTheme _terminalTheme() {
    return xterm.TerminalTheme(
      cursor: _foregroundColor,
      selection: _foregroundColor.withOpacity(0.3),
      foreground: _foregroundColor,
      background: _backgroundColor,
      black: const Color(0xFF0C0C0C),
      red: const Color(0xFFF44747),
      green: const Color(0xFF6A9955),
      yellow: const Color(0xFFD7BA7D),
      blue: const Color(0xFF569CD6),
      magenta: const Color(0xFFC586C0),
      cyan: const Color(0xFF4EC9B0),
      white: _foregroundColor,
      brightBlack: const Color(0xFF808080),
      brightRed: const Color(0xFFF44747),
      brightGreen: const Color(0xFFB5CEA8),
      brightYellow: const Color(0xFFDCDCAA),
      brightBlue: const Color(0xFF9CDCFE),
      brightMagenta: const Color(0xFFD16D9E),
      brightCyan: const Color(0xFF9CDCFE),
      brightWhite: const Color(0xFFFFFFFF),
      searchHitBackground: const Color(0xFFFFD700),
      searchHitBackgroundCurrent: const Color(0xFFFF9632),
      searchHitForeground: const Color(0xFF000000),
    );
  }
}
