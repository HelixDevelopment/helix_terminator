import 'package:flutter/material.dart';

class ChipScale extends StatelessWidget {
  const ChipScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      children: [
        Chip(label: const Text('Chip')),
        const InputChip(label: Text('Input')),
        const ChoiceChip(label: Text('Choice'), selected: true),
        FilterChip(label: const Text('Filter'), selected: false, onSelected: (_) {}),
        const ActionChip(label: Text('Action')),
      ],
    );
  }
}
