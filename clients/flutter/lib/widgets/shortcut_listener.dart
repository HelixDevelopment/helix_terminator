import 'package:flutter/material.dart';

class ShortcutListener extends StatelessWidget {
  final Map<ShortcutActivator, VoidCallback> shortcuts;
  final Widget child;

  const ShortcutListener({super.key, required this.shortcuts, required this.child});

  @override
  Widget build(BuildContext context) {
    return CallbackShortcuts(
      bindings: shortcuts,
      child: Focus(
        autofocus: true,
        child: child,
      ),
    );
  }
}
