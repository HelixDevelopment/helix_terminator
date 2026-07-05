import 'package:flutter/material.dart';

class TweenAnimationBuilderScale extends StatelessWidget {
  const TweenAnimationBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0, end: 1),
      duration: const Duration(milliseconds: 500),
      builder: (context, value, child) {
        return Opacity(opacity: value, child: child);
      },
      child: Container(width: 100, height: 100, color: Colors.blue),
    );
  }
}
