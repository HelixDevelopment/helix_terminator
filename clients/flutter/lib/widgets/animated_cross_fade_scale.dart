import 'package:flutter/material.dart';

class AnimatedCrossFadeScale extends StatelessWidget {
  const AnimatedCrossFadeScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedCrossFade(
      duration: const Duration(milliseconds: 300),
      firstChild: Container(width: 100, height: 100, color: Colors.blue),
      secondChild: Container(width: 100, height: 100, color: Colors.red),
      crossFadeState: CrossFadeState.showFirst,
    );
  }
}
