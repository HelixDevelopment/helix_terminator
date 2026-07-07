import 'package:flutter/material.dart';

class BadgeScale extends StatelessWidget {
  const BadgeScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 16,
      children: [
        Badge(child: const Icon(Icons.mail)),
        Badge(label: const Text('3'), child: const Icon(Icons.notifications)),
        const Badge(child: Icon(Icons.shopping_cart)),
      ],
    );
  }
}
