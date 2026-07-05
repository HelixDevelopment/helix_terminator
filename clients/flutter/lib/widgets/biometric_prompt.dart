import 'package:flutter/material.dart';

class BiometricPrompt extends StatelessWidget {
  final VoidCallback? onAuth;

  const BiometricPrompt({super.key, this.onAuth});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Authentication Required'),
      content: const Text('Please authenticate to continue.'),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Cancel')),
        ElevatedButton(onPressed: onAuth, child: const Text('Authenticate')),
      ],
    );
  }
}
