import 'package:flutter/material.dart';

class TooltipOnboarding extends StatelessWidget {
  final String tooltip;
  final Widget child;

  const TooltipOnboarding({super.key, required this.tooltip, required this.child});

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      preferBelow: false,
      child: child,
    );
  }
}
