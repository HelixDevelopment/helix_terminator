import 'package:flutter/material.dart';

class CrashReporterStub extends StatelessWidget {
  final String error;
  final VoidCallback? onSendReport;

  const CrashReporterStub({super.key, required this.error, this.onSendReport});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate sentry_flutter or firebase_crashlytics
    return AlertDialog(
      title: const Text('Something went wrong'),
      content: Text(error),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Close')),
        ElevatedButton(onPressed: onSendReport, child: const Text('Send Report')),
      ],
    );
  }
}
