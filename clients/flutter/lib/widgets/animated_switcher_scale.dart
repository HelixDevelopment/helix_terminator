import 'package:flutter/material.dart';

class AnimatedSwitcherScale extends StatelessWidget {
  const AnimatedSwitcherScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 300),
      child: Container(key: const ValueKey(1), width: 100, height: 100, color: Colors.blue),
    );
  }
}
