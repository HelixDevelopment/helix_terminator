import 'package:flutter/material.dart';

class ExpansionTileScale extends StatelessWidget {
  const ExpansionTileScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ExpansionTile(
          title: const Text('Default'),
          children: [ListTile(title: const Text('Child'))],
        ),
        ExpansionTile(
          leading: const Icon(Icons.folder),
          title: const Text('With leading'),
          children: [ListTile(title: const Text('Child'))],
        ),
      ],
    );
  }
}
