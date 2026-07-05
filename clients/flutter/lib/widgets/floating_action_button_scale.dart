import 'package:flutter/material.dart';

class FloatingActionButtonScale extends StatelessWidget {
  const FloatingActionButtonScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 16,
      children: [
        FloatingActionButton(onPressed: () {}, child: const Icon(Icons.add)),
        FloatingActionButton.small(onPressed: () {}, child: const Icon(Icons.add)),
        FloatingActionButton.large(onPressed: () {}, child: const Icon(Icons.add)),
        FloatingActionButton.extended(onPressed: () {}, icon: const Icon(Icons.add), label: const Text('Add')),
      ],
    );
  }
}
