import 'package:flutter/material.dart';

class MarkdownViewStub extends StatelessWidget {
  final String data;

  const MarkdownViewStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_markdown
    return Text(data);
  }
}
