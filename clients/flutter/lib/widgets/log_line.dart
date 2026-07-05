import 'package:flutter/material.dart';

class LogLine extends StatelessWidget {
  final String timestamp;
  final String level;
  final String message;

  const LogLine({super.key, required this.timestamp, required this.level, required this.message});

  @override
  Widget build(BuildContext context) {
    return SelectableText(
      '[$timestamp] $level: $message',
      style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
    );
  }
}
