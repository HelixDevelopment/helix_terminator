import 'package:flutter/material.dart';

class SnippetEditor extends StatelessWidget {
  final TextEditingController? controller;

  const SnippetEditor({super.key, this.controller});

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      maxLines: null,
      expands: true,
      style: const TextStyle(fontFamily: 'monospace'),
      decoration: const InputDecoration(hintText: 'Enter code snippet...'),
    );
  }
}
