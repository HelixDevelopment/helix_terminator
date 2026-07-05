import 'package:flutter/material.dart';

class ButtonScale extends StatelessWidget {
  const ButtonScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        ElevatedButton(onPressed: () {}, child: const Text('Elevated')),
        FilledButton(onPressed: () {}, child: const Text('Filled')),
        OutlinedButton(onPressed: () {}, child: const Text('Outlined')),
        TextButton(onPressed: () {}, child: const Text('Text')),
        IconButton(onPressed: () {}, icon: const Icon(Icons.add)),
        FloatingActionButton(onPressed: () {}, child: const Icon(Icons.add)),
      ],
    );
  }
}
