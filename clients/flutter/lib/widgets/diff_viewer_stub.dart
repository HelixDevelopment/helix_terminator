import 'package:flutter/material.dart';

class DiffViewerStub extends StatelessWidget {
  final String oldText;
  final String newText;

  const DiffViewerStub({super.key, required this.oldText, required this.newText});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate diff_match_patch or custom diff viewer
    return Column(
      children: [
        Expanded(child: Container(color: Colors.red.shade100, child: Text(oldText))),
        Expanded(child: Container(color: Colors.green.shade100, child: Text(newText))),
      ],
    );
  }
}
