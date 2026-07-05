import 'package:flutter/material.dart';

class CheckboxListTileScale extends StatelessWidget {
  const CheckboxListTileScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        CheckboxListTile(
          value: true,
          onChanged: (_) {},
          title: const Text('Checked'),
        ),
        CheckboxListTile(
          value: false,
          onChanged: (_) {},
          title: const Text('Unchecked'),
        ),
      ],
    );
  }
}
