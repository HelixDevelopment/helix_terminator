import 'package:flutter/material.dart';

class SimpleDialogScale extends StatelessWidget {
  const SimpleDialogScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => showDialog(
            context: context,
            builder: (_) => SimpleDialog(
              title: const Text('Choose an option'),
              children: [
                SimpleDialogOption(onPressed: () => Navigator.pop(context), child: const Text('Option 1')),
                SimpleDialogOption(onPressed: () => Navigator.pop(context), child: const Text('Option 2')),
              ],
            ),
          ),
          child: const Text('Simple Dialog'),
        ),
      ],
    );
  }
}
