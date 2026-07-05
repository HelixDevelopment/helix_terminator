import 'package:flutter/material.dart';

class SpotlightOverlay extends StatelessWidget {
  final Rect targetRect;
  final String message;
  final VoidCallback? onNext;

  const SpotlightOverlay({super.key, required this.targetRect, required this.message, this.onNext});

  @override
  Widget build(BuildContext context) {
    // TODO: implement custom painter for spotlight effect
    return Container(
      color: Colors.black54,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(message, style: const TextStyle(color: Colors.white)),
            ElevatedButton(onPressed: onNext, child: const Text('Next')),
          ],
        ),
      ),
    );
  }
}
