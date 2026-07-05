import 'package:flutter/material.dart';

class RadioListTileScale extends StatelessWidget {
  const RadioListTileScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        RadioListTile(
          value: 'a',
          groupValue: 'a',
          onChanged: (_) {},
          title: const Text('Selected'),
        ),
        RadioListTile(
          value: 'b',
          groupValue: 'a',
          onChanged: (_) {},
          title: const Text('Unselected'),
        ),
      ],
    );
  }
}
