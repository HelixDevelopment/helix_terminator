import 'package:flutter/material.dart';

class AlertDialogScale extends StatelessWidget {
  const AlertDialogScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => showDialog(
            context: context,
            builder: (_) => AlertDialog(
              title: const Text('Title'),
              content: const Text('Content'),
              actions: [TextButton(onPressed: () => Navigator.pop(context), child: const Text('OK'))],
            ),
          ),
          child: const Text('Alert Dialog'),
        ),
      ],
    );
  }
}
