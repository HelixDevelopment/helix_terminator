import 'package:flutter/material.dart';

class WhatsNewDialog extends StatelessWidget {
  final List<String> features;
  final VoidCallback? onDismiss;

  const WhatsNewDialog({super.key, required this.features, this.onDismiss});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('What\'s New'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: features.map((f) => ListTile(leading: const Icon(Icons.star), title: Text(f))).toList(),
      ),
      actions: [
        TextButton(onPressed: onDismiss, child: const Text('Got it')),
      ],
    );
  }
}
