import 'package:flutter/material.dart';

class LogViewerStub extends StatelessWidget {
  final List<String> logs;

  const LogViewerStub({super.key, required this.logs});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: logs.length,
      itemBuilder: (context, index) => SelectableText(
        logs[index],
        style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
      ),
    );
  }
}
