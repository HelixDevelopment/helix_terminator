import 'package:flutter/material.dart';

class AccessibilityAnnouncer extends StatelessWidget {
  final String message;
  final Widget child;

  const AccessibilityAnnouncer({super.key, required this.message, required this.child});

  @override
  Widget build(BuildContext context) {
    return Semantics(
      liveRegion: true,
      label: message,
      child: child,
    );
  }
}
