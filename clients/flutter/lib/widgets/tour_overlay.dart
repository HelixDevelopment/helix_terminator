import 'package:flutter/material.dart';

class TourOverlay extends StatelessWidget {
  final String message;
  final VoidCallback? onNext;
  final VoidCallback? onSkip;

  const TourOverlay({super.key, required this.message, this.onNext, this.onSkip});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Colors.black54,
      child: Center(
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(message, style: Theme.of(context).textTheme.bodyLarge),
                const SizedBox(height: 16),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    TextButton(onPressed: onSkip, child: const Text('Skip')),
                    ElevatedButton(onPressed: onNext, child: const Text('Next')),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
