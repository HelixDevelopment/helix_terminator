import 'package:flutter/material.dart';

class QuickActionFab extends StatelessWidget {
  final VoidCallback? onPressed;

  const QuickActionFab({super.key, this.onPressed});

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton(
      onPressed: onPressed,
      child: const Icon(Icons.add),
    );
  }
}
