import 'package:flutter/material.dart';

class SessionTimer extends StatelessWidget {
  final Duration elapsed;

  const SessionTimer({super.key, required this.elapsed});

  @override
  Widget build(BuildContext context) {
    final minutes = elapsed.inMinutes.toString().padLeft(2, '0');
    final seconds = (elapsed.inSeconds % 60).toString().padLeft(2, '0');
    return Text('$minutes:$seconds', style: const TextStyle(fontFamily: 'monospace'));
  }
}
