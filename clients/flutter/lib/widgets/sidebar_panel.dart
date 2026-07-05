import 'package:flutter/material.dart';

class SidebarPanel extends StatelessWidget {
  final Widget child;

  const SidebarPanel({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 280,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: child,
    );
  }
}
