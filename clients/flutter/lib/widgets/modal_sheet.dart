import 'package:flutter/material.dart';

class ModalSheet extends StatelessWidget {
  final Widget child;

  const ModalSheet({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: const BoxDecoration(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: child,
    );
  }
}
