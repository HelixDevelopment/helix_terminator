import 'package:flutter/material.dart';

class MinimapStub extends StatelessWidget {
  final String content;

  const MinimapStub({super.key, required this.content});

  @override
  Widget build(BuildContext context) {
    // TODO: implement code minimap
    return Container(
      width: 100,
      color: Colors.grey.shade800,
      child: Text(
        content,
        style: const TextStyle(fontSize: 2, color: Colors.grey),
      ),
    );
  }
}
