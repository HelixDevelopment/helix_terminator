import 'package:flutter/material.dart';

class ConstrainedBoxScale extends StatelessWidget {
  const ConstrainedBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ConstrainedBox(
      constraints: const BoxConstraints(minWidth: 100, maxWidth: 200, minHeight: 50, maxHeight: 100),
      child: Container(color: Colors.blue, child: const Center(child: Text('Constrained'))),
    );
  }
}
