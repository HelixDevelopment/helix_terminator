import 'package:flutter/material.dart';

class ShimmerEffect extends StatelessWidget {
  final Widget child;

  const ShimmerEffect({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate shimmer package
    return Opacity(opacity: 0.5, child: child);
  }
}
