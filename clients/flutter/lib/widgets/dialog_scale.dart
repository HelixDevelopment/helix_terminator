import 'package:flutter/material.dart';

class DialogScale extends StatelessWidget {
  const DialogScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => showDialog(context: context, builder: (_) => const AlertDialog(title: Text('Alert'))),
          child: const Text('Alert Dialog'),
        ),
        ElevatedButton(
          onPressed: () => showDialog(context: context, builder: (_) => const SimpleDialog(title: Text('Simple'))),
          child: const Text('Simple Dialog'),
        ),
      ],
    );
  }
}
