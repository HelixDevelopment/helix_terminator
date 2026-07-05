import 'package:flutter/material.dart';

class TerminalView extends StatelessWidget {
  const TerminalView({super.key});

  @override
  Widget build(BuildContext context) {
    // TODO: implement real terminal emulator (xterm.js bridge or custom)
    return Container(
      color: Colors.black,
      child: const Center(
        child: Text(
          'TerminalView',
          style: TextStyle(color: Colors.green),
        ),
      ),
    );
  }
}
