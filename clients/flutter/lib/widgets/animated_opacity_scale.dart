import 'package:flutter/material.dart';

class AnimatedOpacityScale extends StatelessWidget {
  const AnimatedOpacityScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedOpacity(
      opacity: 1.0,
      duration: const Duration(milliseconds: 300),
      child: Container(width: 100, height: 100, color: Colors.blue),
    );
  }
}
