import 'package:flutter/material.dart';

class FilePreview extends StatelessWidget {
  final String fileName;
  final String? content;

  const FilePreview({super.key, required this.fileName, this.content});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.grey.shade900,
        borderRadius: BorderRadius.circular(8),
      ),
      child: SelectableText(
        content ?? 'No preview available',
        style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
      ),
    );
  }
}
