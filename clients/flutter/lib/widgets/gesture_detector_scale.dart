import 'package:flutter/material.dart';

class GestureDetectorScale extends StatelessWidget {
  const GestureDetectorScale({super.key});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () {},
      onDoubleTap: () {},
      onLongPress: () {},
      child: Container(width: 100, height: 100, color: Colors.blue, child: const Center(child: Text('Tap me'))),
    );
  }
}
