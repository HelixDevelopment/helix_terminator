import 'package:flutter/material.dart';

class TooltipScale extends StatelessWidget {
  const TooltipScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 16,
      children: [
        Tooltip(message: 'Default', child: const Icon(Icons.info)),
        Tooltip(message: 'Rich', child: const Icon(Icons.help)),
        Tooltip(message: 'Long text here', child: const Icon(Icons.description)),
      ],
    );
  }
}
