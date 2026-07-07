import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/terminal_bloc.dart';
import '../widgets/connection_status.dart';

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
  final ScrollController _scrollController = ScrollController();
  final FocusNode _inputFocusNode = FocusNode();
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
    _scrollController.dispose();
    _inputFocusNode.dispose();
    super.dispose();
  }

  void _sendCommand() {
    final text = _inputController.text;
    if (text.isEmpty) return;
    context.read<TerminalBloc>().add(TerminalSendCommand('$text\r\n'));
    _inputController.clear();
    _inputFocusNode.requestFocus();
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      }
    });
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
          _scrollToBottom();
        }
      },
      child: BlocBuilder<TerminalBloc, TerminalState>(
        builder: (context, state) {
          final isConnected = state is TerminalConnected;
          final outputLines = state is TerminalConnected ? state.outputLines : <String>[];

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
                // Terminal emulator view.
                Expanded(
                  child: GestureDetector(
                    onTap: () => _inputFocusNode.requestFocus(),
                    child: Container(
                      color: _backgroundColor,
                      padding: const EdgeInsets.all(8),
                      child: _buildTerminalOutput(outputLines),
                    ),
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

  Widget _buildTerminalOutput(List<String> lines) {
    if (lines.isEmpty) {
      return Center(
        child: Text(
          'Terminal ready. Type a command to begin.',
          style: TextStyle(
            color: _foregroundColor.withOpacity(0.5),
            fontFamily: 'monospace',
            fontSize: 14,
          ),
        ),
      );
    }

    return SelectionArea(
      child: ListView.builder(
        controller: _scrollController,
        itemCount: lines.length,
        itemBuilder: (context, index) {
          return _TerminalLine(
            text: lines[index],
            foregroundColor: _foregroundColor,
          );
        },
      ),
    );
  }
}

/// Renders a single line of terminal output with basic ANSI color parsing.
class _TerminalLine extends StatelessWidget {
  final String text;
  final Color foregroundColor;

  const _TerminalLine({required this.text, required this.foregroundColor});

  @override
  Widget build(BuildContext context) {
    final spans = _parseAnsi(text);
    return RichText(
      text: TextSpan(
        children: spans.isEmpty
            ? [
                TextSpan(
                  text: text,
                  style: TextStyle(
                    color: foregroundColor,
                    fontFamily: 'monospace',
                    fontSize: 14,
                    height: 1.2,
                  ),
                ),
              ]
            : spans,
      ),
    );
  }

  List<TextSpan> _parseAnsi(String input) {
    final spans = <TextSpan>[];
    final regex = RegExp(r'\x1b\[(\d+;?)*m');
    final matches = regex.allMatches(input);

    if (matches.isEmpty) {
      return spans;
    }

    int lastEnd = 0;
    Color currentColor = foregroundColor;

    for (final match in matches) {
      if (match.start > lastEnd) {
        spans.add(TextSpan(
          text: input.substring(lastEnd, match.start),
          style: TextStyle(
            color: currentColor,
            fontFamily: 'monospace',
            fontSize: 14,
            height: 1.2,
          ),
        ));
      }

      final codes = match.group(0)!.replaceAll('\x1b[', '').replaceAll('m', '').split(';');
      for (final code in codes) {
        switch (code) {
          case '30':
            currentColor = Colors.black;
            break;
          case '31':
            currentColor = Colors.red;
            break;
          case '32':
            currentColor = Colors.green;
            break;
          case '33':
            currentColor = Colors.yellow;
            break;
          case '34':
            currentColor = Colors.blue;
            break;
          case '35':
            currentColor = Colors.purple;
            break;
          case '36':
            currentColor = Colors.cyan;
            break;
          case '37':
            currentColor = Colors.white;
            break;
          case '0':
          case '':
            currentColor = foregroundColor;
            break;
        }
      }

      lastEnd = match.end;
    }

    if (lastEnd < input.length) {
      spans.add(TextSpan(
        text: input.substring(lastEnd),
        style: TextStyle(
          color: currentColor,
          fontFamily: 'monospace',
          fontSize: 14,
          height: 1.2,
        ),
      ));
    }

    return spans;
  }
}
