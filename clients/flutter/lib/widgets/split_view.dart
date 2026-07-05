import 'package:flutter/material.dart';

class SplitView extends StatelessWidget {
  final Widget first;
  final Widget second;
  final double ratio;

  const SplitView({super.key, required this.first, required this.second, this.ratio = 0.5});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(flex: (ratio * 100).round(), child: first),
        Expanded(flex: ((1 - ratio) * 100).round(), child: second),
      ],
    );
  }
}
