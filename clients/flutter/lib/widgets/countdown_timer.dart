import 'package:flutter/material.dart';

class CountdownTimer extends StatelessWidget {
  final Duration remaining;

  const CountdownTimer({super.key, required this.remaining});

  @override
  Widget build(BuildContext context) {
    final minutes = remaining.inMinutes.toString().padLeft(2, '0');
    final seconds = (remaining.inSeconds % 60).toString().padLeft(2, '0');
    return Text('$minutes:$seconds', style: const TextStyle(fontFamily: 'monospace', fontSize: 24));
  }
}
