import 'package:flutter/material.dart';

class TransformScale extends StatelessWidget {
  const TransformScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Transform.rotate(
      angle: 0.5,
      child: Container(width: 100, height: 100, color: Colors.blue, child: const Center(child: Text('Rotated'))),
    );
  }
}
