import 'package:flutter/material.dart';

class RateAppPrompt extends StatelessWidget {
  final VoidCallback? onRate;
  final VoidCallback? onLater;
  final VoidCallback? onNever;

  const RateAppPrompt({super.key, this.onRate, this.onLater, this.onNever});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Enjoying HelixTerminator?'),
      content: const Text('Please rate us on the app store.'),
      actions: [
        TextButton(onPressed: onNever, child: const Text('No thanks')),
        TextButton(onPressed: onLater, child: const Text('Later')),
        ElevatedButton(onPressed: onRate, child: const Text('Rate')),
      ],
    );
  }
}
