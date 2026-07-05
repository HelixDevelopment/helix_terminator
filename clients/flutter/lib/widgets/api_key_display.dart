import 'package:flutter/material.dart';

class ApiKeyDisplay extends StatelessWidget {
  final String apiKey;
  final VoidCallback? onRegenerate;

  const ApiKeyDisplay({super.key, required this.apiKey, this.onRegenerate});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: SelectableText(
            apiKey,
            style: const TextStyle(fontFamily: 'monospace'),
          ),
        ),
        IconButton(icon: const Icon(Icons.copy), onPressed: () {}),
        IconButton(icon: const Icon(Icons.refresh), onPressed: onRegenerate),
      ],
    );
  }
}
