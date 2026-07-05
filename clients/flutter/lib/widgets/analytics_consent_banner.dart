import 'package:flutter/material.dart';

class AnalyticsConsentBanner extends StatelessWidget {
  final VoidCallback? onAccept;
  final VoidCallback? onDecline;

  const AnalyticsConsentBanner({super.key, this.onAccept, this.onDecline});

  @override
  Widget build(BuildContext context) {
    return MaterialBanner(
      content: const Text('We use analytics to improve your experience.'),
      actions: [
        TextButton(onPressed: onDecline, child: const Text('Decline')),
        ElevatedButton(onPressed: onAccept, child: const Text('Accept')),
      ],
    );
  }
}
