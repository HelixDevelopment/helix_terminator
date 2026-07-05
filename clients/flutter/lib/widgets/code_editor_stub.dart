import 'package:flutter/material.dart';

class CodeEditorStub extends StatelessWidget {
  final String language;
  final TextEditingController? controller;

  const CodeEditorStub({super.key, required this.language, this.controller});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate code_text_field or flutter_code_editor
    return TextField(
      controller: controller,
      maxLines: null,
      style: const TextStyle(fontFamily: 'monospace'),
      decoration: InputDecoration(hintText: 'Code editor ($language)...'),
    );
  }
}
