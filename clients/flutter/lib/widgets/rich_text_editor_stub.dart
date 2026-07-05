import 'package:flutter/material.dart';

class RichTextEditorStub extends StatelessWidget {
  final TextEditingController? controller;

  const RichTextEditorStub({super.key, this.controller});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_quill or html_editor_enhanced
    return TextField(
      controller: controller,
      maxLines: null,
      decoration: const InputDecoration(hintText: 'Rich text editor...'),
    );
  }
}
