import 'package:flutter/material.dart';

class ConfettiOverlay extends StatelessWidget {
  final bool active;
  final Widget child;

  const ConfettiOverlay({super.key, this.active = false, required this.child});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate confetti package
    return Stack(
      children: [
        child,
        if (active) const Center(child: Text('Confetti!')),
      ],
    );
  }
}
