import 'package:flutter/material.dart';

class WebhookUrlInput extends StatelessWidget {
  final TextEditingController? controller;
  final VoidCallback? onTest;

  const WebhookUrlInput({super.key, this.controller, this.onTest});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: controller,
            decoration: const InputDecoration(hintText: 'https://hooks.example.com/...'),
          ),
        ),
        ElevatedButton(onPressed: onTest, child: const Text('Test')),
      ],
    );
  }
}
