import 'package:flutter/material.dart';

class AnimatedContainerScale extends StatelessWidget {
  const AnimatedContainerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 300),
      width: 100,
      height: 100,
      color: Colors.blue,
    );
  }
}
