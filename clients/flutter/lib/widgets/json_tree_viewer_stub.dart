import 'package:flutter/material.dart';

class JsonTreeViewerStub extends StatelessWidget {
  final Map<String, dynamic> data;

  const JsonTreeViewerStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate json_tree or custom implementation
    return Container(
      padding: const EdgeInsets.all(8),
      color: Colors.grey.shade900,
      child: SelectableText(
        data.toString(),
        style: const TextStyle(color: Colors.green, fontFamily: 'monospace'),
      ),
    );
  }
}
